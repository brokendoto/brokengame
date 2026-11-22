package replay

import (
	"testing"

	"github.com/brokendoto/brokengame/internal/sim"
)

func TestRoundTrip(t *testing.T) {
	in := []sim.Input{0, 0, 0, sim.InUp, sim.InUp, sim.InUp | sim.InDash, 0}
	for i := 0; i < 1000; i++ {
		in = append(in, sim.InRight)
	}
	out, err := Decode(Encode(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("длина %d != %d", len(out), len(in))
	}
	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("шаг %d: %v != %v", i, out[i], in[i])
		}
	}
	if len(Encode(in)) > 20 {
		t.Fatalf("RLE не сжимает: %d", len(Encode(in)))
	}
}

func TestDecodeGarbage(t *testing.T) {
	if _, err := Decode("!!!"); err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if _, err := Decode(Encode(nil) + "AP____8P"); err == nil {
		t.Fatal("ожидалась ошибка на огромной длине")
	}
}
