package music

import (
	"bytes"
	"testing"
)

func all() []Style { return append([]Style{Menu}, Worlds[:]...) }

// Каждая пьеса звучит (не тишина), не перегружена и одинакова при
// повторном синтезе.
func TestRender(t *testing.T) {
	for _, s := range all() {
		pcm := Render(s)
		if len(pcm)%4 != 0 || len(pcm) == 0 {
			t.Fatalf("%s: длина %d", s.Name, len(pcm))
		}
		var loud, clip int
		for i := 0; i+1 < len(pcm); i += 2 {
			v := int16(pcm[i]) | int16(pcm[i+1])<<8
			if v > 3000 || v < -3000 {
				loud++
			}
			if v == 32767 || v == -32768 {
				clip++
			}
		}
		if loud < len(pcm)/2/20 {
			t.Fatalf("%s: слишком тихо (%d громких отсчётов)", s.Name, loud)
		}
		if clip > 0 {
			t.Fatalf("%s: перегруз (%d)", s.Name, clip)
		}
		if !bytes.Equal(pcm, Render(s)) {
			t.Fatalf("%s: синтез не детерминирован", s.Name)
		}
	}
}

// Все ноты мелодий лежат в гамме своей пьесы.
func TestNotesInScale(t *testing.T) {
	for _, s := range all() {
		in := map[int]bool{}
		for _, d := range s.Scale {
			in[d] = true
		}
		notes, _, length := compose(s)
		if len(notes) < 50 || length <= 0 {
			t.Fatalf("%s: пустая пьеса", s.Name)
		}
		for _, n := range notes {
			if !in[((n.midi-s.Root)%12+12)%12] {
				t.Fatalf("%s: нота %d вне гаммы", s.Name, n.midi)
			}
		}
	}
}

func BenchmarkRender(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Render(Worlds[2])
	}
}
