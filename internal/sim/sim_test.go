package sim

import (
	"fmt"
	"testing"
)

func allTracks() []Track {
	var ts []Track
	for w := Mine; w < WorldCount; w++ {
		for d := Easy; d < DifficultyCount; d++ {
			for l := 1; l <= LevelsPerWorld; l++ {
				ts = append(ts, Track{World: w, Level: l, Diff: d})
			}
		}
	}
	for day := 1; day <= 28; day++ {
		ts = append(ts, DailyTrack(fmt.Sprintf("2026-02-%02d", day)))
	}
	return ts
}

func TestGenerateDeterministic(t *testing.T) {
	for _, tr := range allTracks() {
		a, b := Generate(tr), Generate(tr)
		if a.Grid != b.Grid || a.Start != b.Start || a.Exit != b.Exit || fmt.Sprint(a.Items) != fmt.Sprint(b.Items) {
			t.Fatalf("%s: генерация недетерминирована", tr.Key())
		}
	}
}

func TestLevelsReachable(t *testing.T) {
	for _, tr := range allTracks() {
		lv := Generate(tr)
		var d DistField
		lv.BFS(lv.Start, &d)
		if d[lv.Exit.Y][lv.Exit.X] < 0 {
			t.Errorf("%s: выход недостижим", tr.Key())
		}
		for _, p := range lv.Items {
			if d[p.Y][p.X] < 0 {
				t.Errorf("%s: кристалл %v недостижим", tr.Key(), p)
			}
		}
		if len(lv.Items) != lv.Cfg.Items {
			t.Errorf("%s: кристаллов %d, нужно %d", tr.Key(), len(lv.Items), lv.Cfg.Items)
		}
	}
}

// Без опасностей бот обязан пройти каждый уровень и уложиться в золото:
// это проверяет и коллизии (нигде не застревает), и расчёт эталона.
func TestAutopilotClearsLevels(t *testing.T) {
	for _, tr := range allTracks() {
		lv := Generate(tr)
		lv.Spikes, lv.Chasers, lv.Holes, lv.Cursed = nil, nil, nil, nil
		lv.Cfg.RockEvery, lv.Cfg.CaveEvery = 0, 0
		s := NewState(lv)
		for s.Status == Playing {
			s.Step(Autopilot(s))
		}
		if s.Status != Won {
			t.Errorf("%s: бот не прошёл (%s), собрано %d/%d, позиция %.2f,%.2f",
				tr.Key(), s.LoseReason, s.Collected, len(s.Items), s.P.X, s.P.Y)
			continue
		}
		want := Gold
		if lv.Rules().Zone == Fog {
			// В тумане управление наоборот, и бот на границе тумана теряет время.
			want = Silver
		}
		if m := lv.Medal(s.FinishTick); m < want {
			t.Errorf("%s: бот получил %s (%d шагов, эталон %d)", tr.Key(), MedalNames[m], s.FinishTick, lv.ParTicks)
		}
	}
}

// Повтор записи нажатий даёт тот же результат, что и живая игра.
func TestReplayMatchesLive(t *testing.T) {
	for _, tr := range allTracks() {
		lv := Generate(tr)
		s := NewState(lv)
		var rec []Input
		rng := NewRNG(42)
		for s.Status == Playing {
			in := Autopilot(s)
			if rng.Intn(40) == 0 {
				in |= InDash
			}
			if rng.Intn(200) == 0 {
				in |= InAbility
			}
			rec = append(rec, in)
			s.Step(in)
		}
		live := s.Result()
		again := Run(Generate(tr), rec)
		if live != again {
			t.Fatalf("%s: повтор разошёлся: %+v != %+v", tr.Key(), live, again)
		}
	}
}

func TestHazardsHurt(t *testing.T) {
	// На последнем уровне со всеми опасностями бот, который их не избегает,
	// должен хотя бы раз пострадать — иначе опасности не работают.
	hits := 0
	for l := 5; l <= LevelsPerWorld; l++ {
		lv := Generate(Track{World: Mine, Level: l, Diff: Hard})
		s := NewState(lv)
		for s.Status == Playing {
			s.Step(Autopilot(s))
		}
		hits += s.Hits
	}
	if hits == 0 {
		t.Fatal("ни одного попадания за 4 уровня")
	}
}

func TestScoreBreakdown(t *testing.T) {
	lv := Generate(Track{World: Mine, Level: 1, Diff: Normal})
	s := NewState(lv)
	for s.Status == Playing {
		s.Step(Autopilot(s))
	}
	r := s.Result()
	if !r.Won || r.Total != r.Items+r.TimeBonus+r.LifeBonus+r.NoHit || r.NoHit != NoHitBonus {
		t.Fatalf("странный итог: %+v", r)
	}
}

// Жук, оказавшийся ровно в точке игрока, не должен ломать симуляцию
// (раньше здесь было деление на ноль и NaN в координатах).
func TestChaserOnTopOfPlayer(t *testing.T) {
	lv := Generate(Track{World: Mine, Level: 6, Diff: Hard})
	s := NewState(lv)
	s.Chasers[0].Spawn = 0
	s.Chasers[0].X, s.Chasers[0].Y = s.P.X, s.P.Y
	s.P.Invuln = 1000
	for i := 0; i < 300 && s.Status == Playing; i++ {
		s.Step(0)
	}
	for _, c := range s.Chasers {
		if c.X != c.X || c.Y != c.Y {
			t.Fatal("NaN в координатах жука")
		}
	}
}

// Долгий прогон со случайными нажатиями на всех уровнях: симуляция не должна падать.
func TestFuzzRandomInputs(t *testing.T) {
	rng := NewRNG(1)
	for _, tr := range allTracks() {
		lv := Generate(tr)
		lv.TimeLimit = 1 << 30
		s := NewState(lv)
		s.Lives = 1 << 20
		var in Input
		for i := 0; i < 5000; i++ {
			if rng.Intn(10) == 0 {
				in = Input(rng.Intn(64))
			}
			s.Step(in)
			if s.P.X != s.P.X || s.P.X < 0 || s.P.Y < 0 || s.P.X > W || s.P.Y > H {
				t.Fatalf("%s: игрок вылетел за поле: %.2f,%.2f", tr.Key(), s.P.X, s.P.Y)
			}
			for _, c := range s.Chasers {
				if c.X != c.X || c.Y != c.Y {
					t.Fatalf("%s: NaN у жука", tr.Key())
				}
			}
		}
	}
}
