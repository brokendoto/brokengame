package save

import (
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/brokendoto/brokengame/internal/sim"
)

// TopSize — сколько мест в таблице.
const TopSize = 10

// Entry — строка таблицы рекордов.
type Entry struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Ticks  int    `json:"ticks"`
	Medal  int    `json:"medal"`
	Date   string `json:"date"`
	Sim    int    `json:"sim"`              // версия правил, на которой поставлен рекорд
	Replay string `json:"replay"`           // запись нажатий (для призрака и проверки)
	Stages int    `json:"stages,omitempty"` // марафон: сколько этапов пройдено
}

// Better — лучше ли запись a, чем b: больше очков, при равенстве быстрее.
func Better(a, b Entry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.Ticks < b.Ticks
}

// Leaderboard — интерфейс таблицы рекордов. Сейчас реализован локально
// (файл на диске); для Steam добавится реализация через Steamworks
// Leaderboards с тем же интерфейсом, и остальной код менять не придётся.
type Leaderboard interface {
	Top(track string) []Entry
	Submit(track string, e Entry) (rank int)
}

// Records — локальная таблица рекордов (все игроки на этом компьютере).
type Records struct {
	mu     sync.Mutex
	Boards map[string][]Entry `json:"boards"`
	path   string
}

// LoadRecords читает таблицы из папки dir.
func LoadRecords(dir string) *Records {
	r := &Records{Boards: map[string][]Entry{}}
	if dir != "" {
		r.path = filepath.Join(dir, "records.json")
	}
	_ = readJSON(r.path, r)
	if r.Boards == nil {
		r.Boards = map[string][]Entry{}
	}
	return r
}

// Top возвращает копию таблицы трассы.
func (r *Records) Top(track string) []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Entry(nil), r.Boards[track]...)
}

// Submit добавляет результат и возвращает место (1..TopSize) или 0,
// если в таблицу не попал. От одного имени в таблице хранится один,
// лучший, результат — чтобы один игрок не занял все строки.
func (r *Records) Submit(track string, e Entry) int {
	r.mu.Lock()
	if e.Date == "" {
		e.Date = time.Now().Format("2006-01-02")
	}
	if e.Sim == 0 {
		e.Sim = sim.Version
	}
	b := r.Boards[track]
	for i, old := range b {
		if old.Name == e.Name {
			if !Better(e, old) {
				r.mu.Unlock()
				return 0
			}
			b = append(b[:i], b[i+1:]...)
			break
		}
	}
	b = append(b, e)
	sort.SliceStable(b, func(i, j int) bool { return Better(b[i], b[j]) })
	if len(b) > TopSize {
		b = b[:TopSize]
	}
	r.Boards[track] = b
	rank := 0
	for i, x := range b {
		if x.Name == e.Name {
			rank = i + 1
		}
	}
	r.mu.Unlock()
	_ = writeJSON(r.path, r)
	return rank
}

// PersonalBest — лучший результат игрока на трассе.
func (r *Records) PersonalBest(track, name string) (Entry, bool) {
	for _, e := range r.Top(track) {
		if e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

// Ghost выбирает соперника-призрака: лучший забег на трассе, поставленный
// на текущей версии правил. Если игрок сам лидер — его же лучший забег.
func (r *Records) Ghost(track string) (Entry, bool) {
	for _, e := range r.Top(track) {
		if e.Sim == sim.Version && e.Replay != "" {
			return e, true
		}
	}
	return Entry{}, false
}
