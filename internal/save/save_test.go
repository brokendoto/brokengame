package save

import (
	"testing"

	"github.com/brokendoto/brokengame/internal/sim"
)

func TestSubmitRanking(t *testing.T) {
	r := LoadRecords(t.TempDir())
	if got := r.Submit("x", Entry{Name: "Аня", Score: 100, Ticks: 50}); got != 1 {
		t.Fatalf("место %d", got)
	}
	if got := r.Submit("x", Entry{Name: "Боря", Score: 200, Ticks: 50}); got != 1 {
		t.Fatalf("место %d", got)
	}
	if got := r.Submit("x", Entry{Name: "Аня", Score: 90, Ticks: 10}); got != 0 {
		t.Fatalf("худший результат того же игрока не должен попасть: %d", got)
	}
	if got := r.Submit("x", Entry{Name: "Аня", Score: 200, Ticks: 40}); got != 1 {
		t.Fatalf("при равных очках быстрее — выше: %d", got)
	}
	top := r.Top("x")
	if len(top) != 2 || top[0].Name != "Аня" || top[1].Name != "Боря" {
		t.Fatalf("таблица %+v", top)
	}
	r2 := LoadRecords(r.path[:len(r.path)-len("/records.json")])
	if len(r2.Top("x")) != 2 {
		t.Fatal("таблица не сохранилась на диск")
	}
}

func TestTopSizeLimit(t *testing.T) {
	r := LoadRecords("")
	for i := 0; i < 20; i++ {
		r.Submit("x", Entry{Name: string(rune('A' + i)), Score: i})
	}
	if n := len(r.Top("x")); n != TopSize {
		t.Fatalf("строк %d", n)
	}
}

func TestWorldUnlock(t *testing.T) {
	p := LoadProfile("")
	if !p.WorldOpen(sim.Mine) || p.WorldOpen(sim.Forest) {
		t.Fatal("в начале открыта только шахта")
	}
	for lvl := 1; lvl <= LevelsToUnlockWorld; lvl++ {
		if p.WorldOpen(sim.Forest) {
			t.Fatalf("лес открылся слишком рано, после %d уровней", lvl-1)
		}
		p.Complete(sim.Track{World: sim.Mine, Level: lvl, Diff: sim.Hard}, sim.Bronze)
	}
	if !p.WorldOpen(sim.Forest) || p.WorldOpen(sim.Temple) {
		t.Fatal("после 4 уровней шахты должен открыться лес, но не храм")
	}
	// Испытание дня прогресс кампании не двигает.
	p.Complete(sim.DailyTrack("2026-11-23"), sim.Gold)
	if p.WorldOpen(sim.Temple) {
		t.Fatal("испытание дня не открывает миры")
	}
}
