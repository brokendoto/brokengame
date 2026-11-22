// Package sfx синтезирует звуковые эффекты прямо в коде (как и оригинал):
// никаких звуковых файлов, только синусоиды, шум и огибающие.
package sfx

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

// Kind — вид звука.
type Kind int

const (
	Pickup Kind = iota // + номер серии (0..9) задаёт высоту тона
	_
	_
	_
	_
	_
	_
	_
	_
	_
	ExitOpen
	Hit
	Block
	Dash
	Shield
	Warn
	Crash
	Heal
	Win
	Lose
	Tick
	Go
	Menu
	Select
	kindCount
)

// Bank — набор заранее синтезированных звуков.
type Bank struct {
	ctx   *audio.Context
	data  [kindCount][]byte
	Muted bool
	// Volume — громкость от 0 до 1.
	Volume float64
}

// NewBank синтезирует все звуки.
func NewBank() *Bank {
	b := &Bank{ctx: audio.NewContext(sampleRate), Volume: 0.8}
	for i := 0; i < 10; i++ {
		f := 660 * math.Pow(2, float64(i)/12*2) // каждая серия — на тон выше
		b.data[Pickup+Kind(i)] = render(0.16, func(t float64) float64 {
			return (sine(f, t) + 0.5*sine(f*2, t)) * env(t, 0.005, 0.16) * 0.35
		})
	}
	b.data[ExitOpen] = render(0.7, func(t float64) float64 {
		v := 0.0
		for i, f := range []float64{523, 659, 784, 1047} {
			st := float64(i) * 0.09
			if t >= st {
				v += sine(f, t-st) * env(t-st, 0.01, 0.5)
			}
		}
		return v * 0.18
	})
	b.data[Hit] = render(0.35, func(t float64) float64 {
		return (noise(t)*0.6 + sine(110*(1-t), t)) * env(t, 0.002, 0.33) * 0.5
	})
	b.data[Block] = render(0.25, func(t float64) float64 {
		return (sine(1400, t) + sine(2100, t)*0.5) * env(t, 0.002, 0.24) * 0.22
	})
	b.data[Dash] = render(0.18, func(t float64) float64 {
		return noise(t) * env(t, 0.01, 0.17) * 0.25 * (1 - t*4)
	})
	b.data[Shield] = render(0.45, func(t float64) float64 {
		return sine(300+900*t, t) * env(t, 0.02, 0.42) * 0.25
	})
	b.data[Warn] = render(0.2, func(t float64) float64 {
		return square(880, t) * env(t, 0.005, 0.18) * 0.08
	})
	b.data[Crash] = render(0.6, func(t float64) float64 {
		return (noise(t) + sine(55, t)) * env(t, 0.005, 0.58) * 0.45
	})
	b.data[Heal] = render(0.4, func(t float64) float64 {
		return (sine(784, t) + sine(1175, t)) * env(t, 0.01, 0.38) * 0.2
	})
	b.data[Win] = render(1.2, func(t float64) float64 {
		v := 0.0
		for i, f := range []float64{523, 659, 784, 1047, 1319} {
			st := float64(i) * 0.11
			if t >= st {
				v += (sine(f, t-st) + 0.3*sine(f*2, t-st)) * env(t-st, 0.01, 0.8)
			}
		}
		return v * 0.15
	})
	b.data[Lose] = render(1.0, func(t float64) float64 {
		return square(220*math.Pow(0.5, t), t) * env(t, 0.01, 0.95) * 0.12
	})
	b.data[Tick] = render(0.12, func(t float64) float64 { return sine(660, t) * env(t, 0.003, 0.11) * 0.3 })
	b.data[Go] = render(0.35, func(t float64) float64 {
		return (sine(1320, t) + sine(990, t)) * env(t, 0.003, 0.33) * 0.22
	})
	b.data[Menu] = render(0.05, func(t float64) float64 { return sine(1200, t) * env(t, 0.002, 0.05) * 0.15 })
	b.data[Select] = render(0.14, func(t float64) float64 {
		return (sine(880, t) + sine(1320, t)) * env(t, 0.003, 0.13) * 0.18
	})
	return b
}

// Play проигрывает звук.
func (b *Bank) Play(k Kind) {
	if b == nil || b.Muted || b.Volume <= 0 || k < 0 || k >= kindCount || b.data[k] == nil {
		return
	}
	p := b.ctx.NewPlayerFromBytes(b.data[k])
	p.SetVolume(b.Volume)
	p.Play()
}

// PlayPickup — звук кристалла, тем выше, чем длиннее серия.
func (b *Bank) PlayPickup(combo int) {
	if combo < 1 {
		combo = 1
	}
	if combo > 10 {
		combo = 10
	}
	b.Play(Pickup + Kind(combo-1))
}

func sine(f, t float64) float64 { return math.Sin(2 * math.Pi * f * t) }

func square(f, t float64) float64 {
	if math.Sin(2*math.Pi*f*t) >= 0 {
		return 1
	}
	return -1
}

var noiseState uint32 = 2463534242

func noise(float64) float64 {
	noiseState ^= noiseState << 13
	noiseState ^= noiseState >> 17
	noiseState ^= noiseState << 5
	return float64(noiseState)/float64(math.MaxUint32)*2 - 1
}

// env — огибающая: быстрая атака и экспоненциальное затухание.
func env(t, attack, length float64) float64 {
	if t < 0 || t > length {
		return 0
	}
	if t < attack {
		return t / attack
	}
	return math.Exp(-4 * (t - attack) / length)
}

// render превращает функцию сигнала в 16-битный стерео PCM.
func render(seconds float64, f func(t float64) float64) []byte {
	n := int(seconds * sampleRate)
	buf := make([]byte, n*4)
	for i := 0; i < n; i++ {
		v := f(float64(i) / sampleRate)
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		s := int16(v * 32767)
		buf[i*4], buf[i*4+1] = byte(s), byte(s>>8)
		buf[i*4+2], buf[i*4+3] = byte(s), byte(s>>8)
	}
	return buf
}
