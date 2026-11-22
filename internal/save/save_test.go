package save

import "testing"

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
