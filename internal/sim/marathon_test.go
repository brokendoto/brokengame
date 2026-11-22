package sim

import (
	"testing"
	"time"
)

func TestMarathonWeek(t *testing.T) {
	d := time.Date(2026, 11, 22, 23, 0, 0, 0, time.UTC)
	if w := MarathonWeek(d); w != "2026-W47" {
		t.Fatalf("неделя %s", w)
	}
}

func TestMarathonTracks(t *testing.T) {
	seen := map[World]bool{}
	prev := 0
	for s := 1; s <= 20; s++ {
		tr := MarathonTrack("2026-W47", s)
		if tr.Level < prev || tr.Level < 1 || tr.Level > LevelsPerWorld {
			t.Fatalf("этап %d: уровень %d", s, tr.Level)
		}
		prev = tr.Level
		seen[tr.World] = true
		if tr.Key() != "marathon-2026-W47" {
			t.Fatalf("ключ %s", tr.Key())
		}
		if tr.Seed() == MarathonTrack("2026-W48", s).Seed() {
			t.Fatal("разные недели должны давать разные уровни")
		}
	}
	if len(seen) != int(WorldCount) {
		t.Fatal("марафон должен пройти все миры")
	}
	if MarathonTrack("x", 13).Diff != Hard || MarathonTrack("x", 12).Diff != Normal {
		t.Fatal("сложность после 12-го этапа")
	}
}

func TestMarathonCarry(t *testing.T) {
	if MarathonCarry(2, 4) != 2 || MarathonCarry(2, 5) != 3 || MarathonCarry(3, 5) != 3 {
		t.Fatal("жизнь возвращается каждые 5 этапов и не больше предела")
	}
}

// Бот идёт по марафону; повтор записей обязан дать тот же итог.
func TestMarathonReplay(t *testing.T) {
	const week = "2026-W47"
	var recs [][]Input
	var live MarathonResult
	lives := MarathonLives
	for stage := 1; stage <= 6; stage++ {
		s := NewMarathonState(MarathonLevel(week, stage), lives)
		var rec []Input
		for s.Status == Playing && s.Tick < 400*TPS {
			in := Autopilot(s)
			rec = append(rec, in)
			s.Step(in)
		}
		recs = append(recs, rec)
		res := s.Result()
		if !res.Won {
			break
		}
		live.Stages++
		live.Score += res.Total
		live.Ticks += res.Ticks
		lives = MarathonCarry(s.Lives, live.Stages)
	}
	if live.Stages == 0 {
		t.Fatal("бот не прошёл ни одного этапа")
	}
	if got := RunMarathon(week, recs); got != live {
		t.Fatalf("повтор %+v, вживую %+v", got, live)
	}
	t.Logf("бот прошёл этапов: %d, очков: %d", live.Stages, live.Score)
}
