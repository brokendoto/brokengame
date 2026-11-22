package sim

import (
	"fmt"
	"testing"
)

func runDuel(lv *Level, maxTicks int) *Duel {
	d := NewDuel(lv)
	for i := 0; i < maxTicks && !d.Over; i++ {
		d.Step([2]Input{d.Autopilot(0), d.Autopilot(1)})
	}
	return d
}

// Два бота на уровне без опасностей: все предметы собраны, кто-то вышел,
// очки сходятся.
func TestDuelBotsFinish(t *testing.T) {
	for w := Mine; w < WorldCount; w++ {
		for _, l := range []int{1, 4, 8} {
			lv := Generate(Track{World: w, Level: l, Diff: Normal})
			lv.Spikes, lv.Chasers, lv.Holes, lv.Cursed = nil, nil, nil, nil
			lv.Cfg.RockEvery, lv.Cfg.CaveEvery = 0, 0
			d := runDuel(lv, 1<<20)
			key := lv.Track.Key()
			if !d.Over || d.Exited < 0 {
				t.Errorf("%s: дуэль не закончилась выходом", key)
				continue
			}
			if got := d.D[0].Got + d.D[1].Got; got != len(lv.Items) {
				t.Errorf("%s: собрано %d из %d", key, got, len(lv.Items))
			}
			if d.Winner < 0 && d.D[0].ItemScore != d.D[1].ItemScore {
				t.Errorf("%s: ничья при разных очках", key)
			}
		}
	}
}

// Дуэль детерминирована: одинаковые нажатия — одинаковый итог.
func TestDuelDeterministic(t *testing.T) {
	for _, l := range []int{3, 8} {
		for w := Mine; w < WorldCount; w++ {
			lv := Generate(Track{World: w, Level: l, Diff: Hard})
			a, b := runDuel(lv, 60*TPS), runDuel(lv, 60*TPS)
			if fmt.Sprint(a.D) != fmt.Sprint(b.D) || a.S.Tick != b.S.Tick {
				t.Fatalf("%s: дуэль недетерминирована", lv.Track.Key())
			}
		}
	}
}

// Рывок в соперника оглушает его.
func TestDuelPush(t *testing.T) {
	lv := Generate(Track{World: Mine, Level: 1, Diff: Normal})
	d := NewDuel(lv)
	// Ставим второго игрока справа от первого, первый делает рывок вправо.
	d.D[1].P.X, d.D[1].P.Y = d.D[0].P.X+0.9, d.D[0].P.Y
	if lv.Solid(int(d.D[1].P.X), int(d.D[1].P.Y)) {
		t.Skip("справа от старта стена")
	}
	stunned := false
	for i := 0; i < 20; i++ {
		in := Input(0)
		if i == 0 {
			in = InRight | InDash
		}
		d.Step([2]Input{in, 0})
		if d.D[1].Stun > 0 {
			stunned = true
		}
	}
	if !stunned {
		t.Fatal("толчок рывком не оглушил соперника")
	}
}

// Опасности бьют обоих, выбитый игрок возвращается на старт, проигрыша нет.
func TestDuelKnockoutRespawn(t *testing.T) {
	knocks := 0
	for w := Mine; w < WorldCount; w++ {
		lv := Generate(Track{World: w, Level: 8, Diff: Hard})
		d := NewDuel(lv)
		for !d.Over {
			d.Step([2]Input{d.Autopilot(0), d.Autopilot(1)})
			for i := range d.D {
				if d.D[i].Down == 0 && d.D[i].Lives <= 0 {
					t.Fatalf("%s: игрок %d на поле без жизней", lv.Track.Key(), i)
				}
			}
		}
		if d.S.LoseReason != "" && d.S.LoseReason != "Время вышло" {
			t.Fatalf("%s: дуэль кончилась поражением «%s»", lv.Track.Key(), d.S.LoseReason)
		}
		knocks += d.D[0].Knocks + d.D[1].Knocks
	}
	if knocks == 0 {
		t.Fatal("за пять сложных уровней никого не выбили — опасности в дуэли не работают")
	}
}
