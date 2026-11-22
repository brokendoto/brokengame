package achieve

import (
	"testing"

	"github.com/brokendoto/brokengame/internal/sim"
)

func has(ids []ID, id ID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func TestListUnique(t *testing.T) {
	seen := map[ID]bool{}
	for _, a := range List {
		if seen[a.ID] || a.Name == "" || a.Desc == "" {
			t.Fatalf("плохое достижение %+v", a)
		}
		seen[a.ID] = true
	}
}

// Настоящий забег бота на уровне без опасностей: первая победа, без ударов, золото.
func TestCheckRealRun(t *testing.T) {
	lv := sim.Generate(sim.Track{World: sim.Mine, Level: 8, Diff: sim.Hard})
	lv.Spikes, lv.Chasers = nil, nil
	lv.Cfg.RockEvery, lv.Cfg.CaveEvery = 0, 0
	s := sim.NewState(lv)
	for s.Status == sim.Playing {
		s.Step(sim.Autopilot(s))
	}
	res := s.Result()
	ids := CheckRun(Run{Track: lv.Track, Result: res, ParTicks: lv.ParTicks, Medals: map[string]sim.Medal{}})
	for _, want := range []ID{FirstWin, NoHit, HardFinale, Gold} {
		if !has(ids, want) {
			t.Errorf("нет %s в %v", want, ids)
		}
	}
	// С ботом за рулём — только «Я, робот».
	ids = CheckRun(Run{Track: lv.Track, Result: res, UsedBot: true})
	if len(ids) != 1 || ids[0] != Robot {
		t.Errorf("бот получил %v", ids)
	}
}

func TestWorldAndUnlock(t *testing.T) {
	medals := map[string]sim.Medal{}
	for l := 1; l <= sim.LevelsPerWorld; l++ {
		d := sim.Easy
		if l%2 == 0 {
			d = sim.Hard // разные сложности тоже засчитываются
		}
		medals[sim.Track{World: sim.Forest, Level: l, Diff: d}.Key()] = sim.Gold
	}
	if c, tot, ok := Progress(ForestAll, medals); !ok || c != 8 || tot != 8 {
		t.Fatalf("прогресс леса %d/%d", c, tot)
	}
	ids := CheckRun(Run{Track: sim.Track{World: sim.Forest, Level: 3}, Result: sim.Result{Won: true}, Medals: medals})
	if !has(ids, ForestAll) || !has(ids, GoldWorld) || has(ids, MineAll) {
		t.Fatalf("неверно: %v", ids)
	}
	got := map[string]string{}
	if fresh := Unlock(got, ids, "2026-11-22"); len(fresh) != len(ids) {
		t.Fatalf("первый раз всё новое: %v", fresh)
	}
	if fresh := Unlock(got, ids, "2026-11-23"); len(fresh) != 0 || got[string(ForestAll)] != "2026-11-22" {
		t.Fatalf("повторно ничего не выдаётся: %v", fresh)
	}
}

func TestEventsCounters(t *testing.T) {
	r := Run{Result: sim.Result{Won: true, BestCombo: 9}, GoldenTotal: 3, GoldenTaken: 3, CursedTotal: 2, HoleHit: true, GhostScore: 10, Medals: map[string]sim.Medal{}}
	r.Result.Total = 11
	ids := CheckRun(r)
	for _, want := range []ID{MaxCombo, AllGolden, PureSoul, EventHoriz, BeatGhost} {
		if !has(ids, want) {
			t.Errorf("нет %s", want)
		}
	}
	r.CursedTaken, r.GoldenTaken = 1, 2
	ids = CheckRun(r)
	if has(ids, PureSoul) || has(ids, AllGolden) {
		t.Error("условия со счётчиками не проверяются")
	}
}

// Идентификаторы идут в Steam как API-имена достижений: только латиница,
// цифры и подчёркивание.
func TestSteamAPINames(t *testing.T) {
	for _, a := range List {
		for _, c := range a.ID {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
				t.Fatalf("%q не годится как API-имя Steam", a.ID)
			}
		}
	}
}
