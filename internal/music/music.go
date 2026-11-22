// Package music сочиняет и синтезирует фоновую музыку прямо в коде: у
// каждого мира и у меню своя пьеса. Ни нот, ни звуковых файлов из чужих
// источников нет. Мелодии и ритмы строятся по правилам (гамма, аккорды,
// мотивы) из зерна, поэтому пьеса всегда одна и та же, а права на неё
// целиком принадлежат автору игры.
//
// Пакет не зависит от движка: Render возвращает готовый 16-битный стерео
// PCM, который игра крутит по кругу.
package music

import "math"

// SampleRate — частота дискретизации (как у звуковых эффектов).
const SampleRate = 44100

// Instrument — тембр партии.
type Instrument int

const (
	Pluck   Instrument = iota // щипок: арфа, кото
	Lead                      // напевный голос с вибрато (флейта)
	Bass                      // мягкий квадратный бас
	SawBass                   // жёсткий бас для передовой
	Pad                       // тянущийся аккорд
	Bell                      // колокольчик
	Chip                      // 8-битный квадрат
)

// Drum — удар.
type Drum int

const (
	Kick Drum = iota
	Snare
	Hat
	Taiko
	Shaker
)

// Style — характер пьесы.
type Style struct {
	Name    string
	Seed    uint64
	BPM     float64
	Root    int   // нота тоники (MIDI)
	Scale   []int // ступени гаммы в полутонах
	Chords  []int // ступени аккордов, по аккорду на 2 такта
	Melody  Instrument
	Arp     Instrument
	Bass    Instrument
	Pad     bool
	ArpRate int   // нот арпеджио в такте (0 — без арпеджио)
	BassPat []int // доли такта (из 16), где звучит бас
	Kick    []int // доли бочки
	Snare   []int // доли малого
	Hats    []int // доли хай-хэта
	DrumKit [3]Drum
	Density float64 // насколько густая мелодия, 0..1
	Swing   float64
}

var (
	minor     = []int{0, 2, 3, 5, 7, 8, 10}
	major     = []int{0, 2, 4, 5, 7, 9, 11}
	dorian    = []int{0, 2, 3, 5, 7, 9, 10}
	lydian    = []int{0, 2, 4, 6, 7, 9, 11}
	hirajoshi = []int{0, 2, 3, 7, 8} // японская пентатоника
	mixolyd   = []int{0, 2, 4, 5, 7, 9, 10}
)

// Styles: меню и пять миров (порядок миров как в sim).
var (
	Menu = Style{
		Name: "menu", Seed: 11, BPM: 96, Root: 53, Scale: major, Chords: []int{0, 5, 3, 4},
		Melody: Bell, Arp: Pluck, Bass: Bass, Pad: true, ArpRate: 8,
		BassPat: []int{0, 8}, Hats: []int{4, 12}, DrumKit: [3]Drum{Kick, Snare, Shaker},
		Density: 0.45,
	}
	Worlds = [5]Style{
		{ // Кристальная шахта: таинственно, капающие кристаллы
			Name: "mine", Seed: 21, BPM: 104, Root: 50, Scale: dorian, Chords: []int{0, 6, 5, 6},
			Melody: Bell, Arp: Pluck, Bass: Bass, ArpRate: 8,
			BassPat: []int{0, 3, 8, 11}, Kick: []int{0, 8}, Snare: []int{4, 12}, Hats: []int{2, 6, 10, 14},
			DrumKit: [3]Drum{Kick, Snare, Hat}, Density: 0.5,
		},
		{ // Осенний лес: тёплая флейта, лёгкий шаг
			Name: "forest", Seed: 33, BPM: 112, Root: 55, Scale: mixolyd, Chords: []int{0, 6, 3, 0},
			Melody: Lead, Arp: Pluck, Bass: Bass, Pad: true, ArpRate: 8,
			BassPat: []int{0, 6, 8, 14}, Kick: []int{0, 10}, Snare: []int{8}, Hats: []int{0, 4, 8, 12},
			DrumKit: [3]Drum{Kick, Snare, Shaker}, Density: 0.6, Swing: 0.12,
		},
		{ // Ночной храм: кото и тайко, много воздуха
			Name: "temple", Seed: 47, BPM: 84, Root: 57, Scale: hirajoshi, Chords: []int{0, 3, 1, 0},
			Melody: Pluck, Arp: Bell, Bass: Bass, Pad: true, ArpRate: 4,
			BassPat: []int{0}, Kick: []int{0, 6}, Snare: []int{12}, Hats: nil,
			DrumKit: [3]Drum{Taiko, Taiko, Shaker}, Density: 0.4,
		},
		{ // Руины передовой: быстрый напор
			Name: "front", Seed: 59, BPM: 132, Root: 52, Scale: minor, Chords: []int{0, 5, 2, 6},
			Melody: Chip, Arp: Chip, Bass: SawBass, ArpRate: 16,
			BassPat: []int{0, 2, 4, 6, 8, 10, 12, 14}, Kick: []int{0, 4, 8, 12}, Snare: []int{4, 12}, Hats: []int{2, 6, 10, 14},
			DrumKit: [3]Drum{Kick, Snare, Hat}, Density: 0.55,
		},
		{ // Открытый космос: медленные пады, звёздные колокольчики
			Name: "space", Seed: 71, BPM: 88, Root: 48, Scale: lydian, Chords: []int{0, 1, 5, 4},
			Melody: Bell, Arp: Bell, Bass: Bass, Pad: true, ArpRate: 8,
			BassPat: []int{0}, Kick: []int{0}, Snare: nil, Hats: []int{8},
			DrumKit: [3]Drum{Kick, Snare, Shaker}, Density: 0.35,
		},
	}
)

// Bars — длина пьесы в тактах (потом она повторяется по кругу).
const Bars = 16

type note struct {
	start, dur float64 // в секундах
	midi       int
	vel        float64
	inst       Instrument
	pan        float64 // -1 слева, 1 справа
}

type hit struct {
	start float64
	drum  Drum
	vel   float64
}

type rng struct{ s uint64 }

func (r *rng) next() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}
func (r *rng) intn(n int) int        { return int(r.next() % uint64(n)) }
func (r *rng) float() float64        { return float64(r.next()%10000) / 10000 }
func (r *rng) chance(p float64) bool { return r.float() < p }

// pitch — нота ступени deg (может быть отрицательной или выше октавы).
func (s *Style) pitch(deg int) int {
	n := len(s.Scale)
	oct := int(math.Floor(float64(deg) / float64(n)))
	return s.Root + 12*oct + s.Scale[deg-oct*n]
}

// Compose раскладывает пьесу на ноты и удары. Длительность пьесы — в
// возвращаемом length (секунды).
func compose(s Style) (notes []note, hits []hit, length float64) {
	r := rng{s.Seed*0x9E3779B97F4A7C15 + 1}
	step := 60 / s.BPM / 4 // шестнадцатая
	bar := step * 16
	length = bar * Bars
	at := func(b, st int) float64 {
		t := float64(b)*bar + float64(st)*step
		if st%2 == 1 {
			t += s.Swing * step
		}
		return t
	}
	// Мотив: ритм и шаги мелодии на 2 такта; по кругу он варьируется.
	type mstep struct {
		pos, dur, move int
	}
	makeMotif := func() []mstep {
		var m []mstep
		for pos := 0; pos < 32; {
			dur := []int{2, 2, 4, 4, 6, 8}[r.intn(6)]
			if pos+dur > 32 {
				dur = 32 - pos
			}
			if r.chance(s.Density) || pos == 0 {
				m = append(m, mstep{pos, dur, r.intn(5) - 2})
			}
			pos += dur
		}
		return m
	}
	motifA, motifB := makeMotif(), makeMotif()
	chordsPerLoop := len(s.Chords)
	for b := 0; b < Bars; b++ {
		ci := (b / 2) % chordsPerLoop
		deg := s.Chords[ci]
		chord := []int{deg, deg + 2, deg + 4}
		if len(s.Scale) == 5 {
			chord = []int{deg, deg + 1, deg + 3}
		}
		// Бас.
		for i, st := range s.BassPat {
			end := 16
			if i+1 < len(s.BassPat) {
				end = s.BassPat[i+1]
			}
			d := chord[0] - len(s.Scale)
			if i%4 == 3 {
				d = chord[2] - len(s.Scale) // проходящая квинта
			}
			notes = append(notes, note{at(b, st), float64(end-st) * step * 0.9, s.pitch(d), 0.55, s.Bass, 0})
		}
		// Пэд держит аккорд оба такта, без повторной атаки.
		if s.Pad && b%2 == 0 {
			for _, d := range chord {
				notes = append(notes, note{at(b, 0), 2 * bar, s.pitch(d), 0.12, Pad, 0})
			}
		}
		// Арпеджио вверх-вниз по аккорду.
		if s.ArpRate > 0 && b >= 2 {
			every := 16 / s.ArpRate
			pattern := []int{0, 1, 2, 3, 2, 1}
			for i := 0; i < s.ArpRate; i++ {
				k := pattern[(i+b)%len(pattern)]
				d := chord[k%3] + len(s.Scale)*(k/3)
				notes = append(notes, note{at(b, i*every), float64(every) * step * 1.6, s.pitch(d), 0.22, s.Arp, 0.4 - 0.8*float64(i%2)})
			}
		}
		// Барабаны: вступление первые 2 такта без бочки.
		kit := s.DrumKit
		for _, st := range s.Kick {
			if b >= 2 {
				hits = append(hits, hit{at(b, st), kit[0], 0.9})
			}
		}
		for _, st := range s.Snare {
			if b >= 2 {
				hits = append(hits, hit{at(b, st), kit[1], 0.6})
			}
		}
		for _, st := range s.Hats {
			hits = append(hits, hit{at(b, st), kit[2], 0.35})
		}
		if b%4 == 3 && b >= 2 { // сбивка в конце фразы
			hits = append(hits, hit{at(b, 14), kit[1], 0.45}, hit{at(b, 15), kit[1], 0.55})
		}
	}
	// Мелодия: с 4-го такта, фразы A A' B A по два такта.
	for phrase := 2; phrase < Bars/2; phrase++ {
		m := motifA
		if phrase%4 == 2 {
			m = motifB
		}
		deg := s.Chords[phrase%chordsPerLoop] + len(s.Scale) // на октаву выше аккорда
		cur := deg
		for i, ms := range m {
			cur += ms.move
			if phrase%4 == 3 && i == 0 {
				cur += 2 // вариация: начинаем выше
			}
			// Держимся около аккорда.
			if cur > deg+5 {
				cur -= 3
			} else if cur < deg-3 {
				cur += 3
			}
			if i == len(m)-1 {
				cur = deg // фраза заканчивается на тонике аккорда
			}
			b, st := phrase*2+ms.pos/16, ms.pos%16
			notes = append(notes, note{at(b, st), float64(ms.dur) * step * 0.95, s.pitch(cur), 0.32, s.Melody, 0.15})
		}
	}
	return notes, hits, length
}

// fsin — синус по таблице: синтез пьесы в браузере и на телефоне
// заметно быстрее, а на слух разницы нет.
func fsin(x float64) float64 {
	const n = 1 << 12
	p := x * (n / (2 * math.Pi))
	i := math.Floor(p)
	fr := p - i
	k := int(i) & (n - 1)
	return sinTab[k] + (sinTab[k+1]-sinTab[k])*fr
}

var sinTab = func() (t [1<<12 + 1]float64) {
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / (1 << 12))
	}
	return
}()

func mtof(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

// voice — значение инструмента в момент t от начала ноты (секунды),
// включая затухание после отпускания (rel).
func voice(in Instrument, f, t, dur float64) float64 {
	tw := 2 * math.Pi * f * t
	switch in {
	case Pluck:
		e := math.Exp(-t * 6)
		return (fsin(tw) + 0.5*fsin(2*tw)*math.Exp(-t*12) + 0.25*fsin(3*tw)*math.Exp(-t*18)) * e * 0.6
	case Bell:
		e := math.Exp(-t * 3.5)
		return (fsin(tw) + 0.4*fsin(2.76*tw)*math.Exp(-t*6) + 0.2*fsin(5.4*tw)*math.Exp(-t*10)) * e * 0.5
	case Lead:
		vib := 1 + 0.006*fsin(2*math.Pi*5.5*t)*math.Min(1, t*3)
		tw = 2 * math.Pi * f * t * vib
		return (fsin(tw) + 0.15*fsin(2*tw) + 0.05*fsin(3*tw)) * adsr(t, dur, 0.05, 0.15) * 0.55
	case Bass:
		return (fsin(tw) + 0.3*fsin(2*tw) + 0.1*fsin(3*tw)) * adsr(t, dur, 0.01, 0.08) * 0.8
	case SawBass:
		v := 0.0
		for k := 1.0; k <= 6; k++ {
			v += fsin(k*tw) / k
		}
		return v * adsr(t, dur, 0.005, 0.05) * math.Exp(-t*2) * 0.6
	case Pad:
		v := 0.0
		for _, det := range []float64{0.997, 1.003} {
			for k := 1.0; k <= 3; k++ {
				v += fsin(k*tw*det) / (k * k)
			}
		}
		return v * adsr(t, dur, 0.6, 0.8) * 0.35
	case Chip:
		v := 0.0
		for k := 1.0; k <= 7; k += 2 {
			v += fsin(k*tw) / k
		}
		return v * adsr(t, dur, 0.005, 0.05) * 0.4
	}
	return 0
}

// release — сколько звучит хвост после конца ноты.
func release(in Instrument) float64 {
	switch in {
	case Pluck, Bell:
		return 0.9
	case Pad:
		return 0.8
	case Lead:
		return 0.15
	}
	return 0.08
}

// adsr — простая огибающая: атака, удержание, затухание после dur.
func adsr(t, dur, attack, rel float64) float64 {
	if t < attack {
		return t / attack
	}
	if t < dur {
		return 1
	}
	return math.Max(0, 1-(t-dur)/rel)
}

func drum(d Drum, t float64, n *rng) float64 {
	switch d {
	case Kick:
		f := 50 + 90*math.Exp(-t*30)
		return math.Sin(2*math.Pi*f*t) * math.Exp(-t*9)
	case Snare:
		return (n.float()*2-1)*math.Exp(-t*18)*0.6 + math.Sin(2*math.Pi*190*t)*math.Exp(-t*25)*0.4
	case Hat:
		return (n.float()*2 - 1) * math.Exp(-t*60) * 0.35
	case Shaker:
		return (n.float()*2 - 1) * math.Min(1, t*200) * math.Exp(-t*35) * 0.25
	case Taiko:
		f := 70 + 40*math.Exp(-t*15)
		return (math.Sin(2*math.Pi*f*t) + 0.2*(n.float()*2-1)*math.Exp(-t*40)) * math.Exp(-t*5)
	}
	return 0
}

func drumLen(d Drum) float64 {
	switch d {
	case Kick, Taiko:
		return 0.6
	case Snare:
		return 0.3
	}
	return 0.12
}

// Render синтезирует пьесу в 16-битный стерео PCM (little-endian). Конец
// пьесы бесшовно переходит в начало: хвосты нот заворачиваются в начало.
func Render(s Style) []byte {
	notes, hits, length := compose(s)
	n := int(length * SampleRate)
	l := make([]float32, n)
	rr := make([]float32, n)
	add := func(i int, v, pan float64) {
		i %= n
		l[i] += float32(v * (1 - math.Max(0, pan)) * 0.9)
		rr[i] += float32(v * (1 + math.Min(0, pan)) * 0.9)
	}
	for _, nt := range notes {
		f := mtof(nt.midi)
		total := nt.dur + release(nt.inst)
		start := int(nt.start * SampleRate)
		cnt := int(total * SampleRate)
		for k := 0; k < cnt; k++ {
			t := float64(k) / SampleRate
			add(start+k, voice(nt.inst, f, t, nt.dur)*nt.vel, nt.pan)
		}
	}
	nr := rng{s.Seed + 99}
	for _, h := range hits {
		start := int(h.start * SampleRate)
		cnt := int(drumLen(h.drum) * SampleRate)
		for k := 0; k < cnt; k++ {
			add(start+k, drum(h.drum, float64(k)/SampleRate, &nr)*h.vel*0.7, 0)
		}
	}
	// Нормируем по пику.
	peak := float32(0.001)
	for i := range l {
		peak = max(peak, abs32(l[i]), abs32(rr[i]))
	}
	g := 0.85 / peak
	buf := make([]byte, n*4)
	for i := 0; i < n; i++ {
		a, b := int16(l[i]*g*32767), int16(rr[i]*g*32767)
		buf[i*4], buf[i*4+1] = byte(a), byte(a>>8)
		buf[i*4+2], buf[i*4+3] = byte(b), byte(b>>8)
	}
	return buf
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
