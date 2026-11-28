package sim

import (
	"fmt"
	"testing"
)

func runDuel(lv *Level, mode DuelMode, maxTicks int) *Duel {
	d := NewDuel(lv, mode)
	for i := 0; i < maxTicks && !d.Over; i++ {
		d.Step([2]Input{d.Autopilot(0), d.Autopilot(1)})
	}
	return d
}

func calmLevel(w World, l int) *Level {
	lv := Generate(Track{World: w, Level: l, Diff: Normal})
	lv.Spikes, lv.Chasers, lv.Holes, lv.Cursed = nil, nil, nil, nil
	lv.Cfg.RockEvery, lv.Cfg.CaveEvery = 0, 0
	return lv
}

// Два бота на уровне без опасностей собирают (и сдают) всё раньше времени.
func TestDuelBotsFinish(t *testing.T) {
	for _, mode := range []DuelMode{DuelRounds, DuelBank} {
		for w := Mine; w < WorldCount; w++ {
			for _, l := range []int{1, 4, 8} {
				lv := calmLevel(w, l)
				d := runDuel(lv, mode, 1<<20)
				key := fmt.Sprint(DuelModes[mode].Name, " ", lv.Track.Key())
				if !d.Over || d.TimedOut() {
					t.Errorf("%s: раунд не закончился сбором (осталось %d)", key, d.Left())
					continue
				}
				if d.D[0].Got+d.D[1].Got < len(lv.Items) {
					t.Errorf("%s: подобрано %d из %d", key, d.D[0].Got+d.D[1].Got, len(lv.Items))
				}
				if d.D[0].Carry != 0 || d.D[1].Carry != 0 {
					t.Errorf("%s: мешки не сданы", key)
				}
				if want := d.Leader(); d.Winner != want {
					t.Errorf("%s: победитель %d, а ведёт %d", key, d.Winner, want)
				}
			}
		}
	}
}

// Дуэль детерминирована: одинаковые нажатия — одинаковый итог.
func TestDuelDeterministic(t *testing.T) {
	for mode := DuelMode(0); mode < DuelModeCount; mode++ {
		for _, l := range []int{3, 8} {
			for w := Mine; w < WorldCount; w++ {
				lv1 := Generate(Track{World: w, Level: l, Diff: Hard})
				lv2 := Generate(Track{World: w, Level: l, Diff: Hard})
				a, b := runDuel(lv1, mode, 60*TPS), runDuel(lv2, mode, 60*TPS)
				if fmt.Sprint(a.D) != fmt.Sprint(b.D) || a.S.Tick != b.S.Tick {
					t.Fatalf("%s: дуэль недетерминирована", lv1.Track.Key())
				}
			}
		}
	}
}

// pushSetup ставит второго игрока справа от первого; первый делает рывок вправо.
func pushSetup(t *testing.T, mode DuelMode) *Duel {
	lv := Generate(Track{World: Mine, Level: 1, Diff: Normal})
	d := NewDuel(lv, mode)
	d.D[1].P.X, d.D[1].P.Y = d.D[0].P.X+0.9, d.D[0].P.Y
	if lv.Solid(int(d.D[1].P.X), int(d.D[1].P.Y)) {
		t.Skip("справа от старта стена")
	}
	return d
}

func dashRight(d *Duel) (stunned bool) {
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
	return
}

// Рывок в соперника оглушает его и выбивает предмет на поле.
func TestDuelPushSteals(t *testing.T) {
	d := pushSetup(t, DuelRounds)
	d.D[1].Pts = 3
	items := len(d.S.Items)
	if !dashRight(d) {
		t.Fatal("толчок рывком не оглушил соперника")
	}
	if d.D[1].Pts != 2 || len(d.S.Items) != items+1 || d.D[0].Steals != 1 {
		t.Fatalf("предмет не выбит: очки %d, предметов %d→%d", d.D[1].Pts, items, len(d.S.Items))
	}
	it := d.S.Items[len(d.S.Items)-1]
	if d.S.Lv.Solid(it.P.X, it.P.Y) || it.Taken {
		t.Fatal("выбитый предмет упал в стену")
	}
}

// «Банк»: толчок вытряхивает весь мешок, сданное не отнять.
func TestDuelBankPush(t *testing.T) {
	d := pushSetup(t, DuelBank)
	d.D[1].Carry, d.D[1].Pts = 4, 5
	dashRight(d)
	if d.D[1].Carry != 0 || d.D[1].Pts != 5 {
		t.Fatalf("мешок %d, банк %d", d.D[1].Carry, d.D[1].Pts)
	}
}

// «Выживание»: толчок ранит, без жизней — проигрыш раунда.
func TestDuelSurvivalPush(t *testing.T) {
	d := pushSetup(t, DuelSurvival)
	d.D[1].Lives = 1
	dashRight(d)
	if !d.Over || d.Winner != 0 || !d.Out(1) {
		t.Fatalf("раунд не кончился: over=%v winner=%d lives=%d", d.Over, d.Winner, d.D[1].Lives)
	}
}

// Опасности бьют обоих, выбитый игрок возвращается на старт, проигрыша нет.
func TestDuelKnockoutRespawn(t *testing.T) {
	knocks := 0
	for w := Mine; w < WorldCount; w++ {
		lv := Generate(Track{World: w, Level: 8, Diff: Hard})
		d := NewDuel(lv, DuelRounds)
		for !d.Over {
			d.Step([2]Input{d.Autopilot(0), d.Autopilot(1)})
			for i := range d.D {
				if d.D[i].Down == 0 && d.D[i].Lives <= 0 {
					t.Fatalf("%s: игрок %d на поле без жизней", lv.Track.Key(), i)
				}
				if d.D[i].Pts < 0 {
					t.Fatalf("%s: отрицательный счёт", lv.Track.Key())
				}
			}
		}
		if d.S.LoseReason != "" {
			t.Fatalf("%s: дуэль кончилась поражением «%s»", lv.Track.Key(), d.S.LoseReason)
		}
		knocks += d.D[0].Knocks + d.D[1].Knocks
	}
	if knocks == 0 {
		t.Fatal("за пять сложных уровней никого не выбили — опасности в дуэли не работают")
	}
}

// Матч всех видов доходит до конца и не длиннее пяти раундов.
func TestDuelMatch(t *testing.T) {
	for mode := DuelMode(0); mode < DuelModeCount; mode++ {
		m := NewDuelMatch(Track{World: Forest, Level: 7, Diff: Normal}, mode)
		for !m.Over {
			d := m.D
			for !d.Over {
				d.Step([2]Input{d.Autopilot(0), d.Autopilot(1)})
			}
			if !m.EndRound() {
				m.NextRound()
			}
			if m.Round > DuelMaxRounds {
				t.Fatalf("%s: раундов больше %d", DuelModes[mode].Name, DuelMaxRounds)
			}
		}
		if m.Winner >= 0 && m.Wins[m.Winner] <= m.Wins[1-m.Winner] {
			t.Fatalf("%s: победитель матча с меньшим числом побед %v", DuelModes[mode].Name, m.Wins)
		}
		t.Logf("%s: раундов %d, счёт %v, история %v", DuelModes[mode].Name, m.Round, m.Wins, m.History)
	}
}
