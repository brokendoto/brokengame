package save

import (
	"path/filepath"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Profile — настройки и прогресс игрока.
type Profile struct {
	Name       string         `json:"name"`
	Diff       sim.Difficulty `json:"difficulty"`
	Muted      bool           `json:"muted"`
	Fullscreen bool           `json:"fullscreen"`
	ShowGhost  bool           `json:"show_ghost"`
	World      sim.World      `json:"world"` // последний выбранный мир
	// Unlocked: "mine-normal" → номер последнего открытого уровня.
	Unlocked map[string]int `json:"unlocked"`
	// Medals: ключ трассы → лучшая медаль.
	Medals map[string]sim.Medal `json:"medals"`

	path string
}

// LoadProfile читает профиль из папки dir (или создаёт новый).
func LoadProfile(dir string) *Profile {
	p := &Profile{Diff: sim.Normal, ShowGhost: true}
	if dir != "" {
		p.path = filepath.Join(dir, "profile.json")
	}
	_ = readJSON(p.path, p)
	if p.Unlocked == nil {
		p.Unlocked = map[string]int{}
	}
	if p.Medals == nil {
		p.Medals = map[string]sim.Medal{}
	}
	if p.Diff < 0 || p.Diff >= sim.DifficultyCount {
		p.Diff = sim.Normal
	}
	if p.World < 0 || p.World >= sim.WorldCount || !p.WorldOpen(p.World) {
		p.World = sim.Mine
	}
	return p
}

// Save записывает профиль.
func (p *Profile) Save() error { return writeJSON(p.path, p) }

func unlockKey(w sim.World, d sim.Difficulty) string {
	return sim.Worlds[w].ID + "-" + sim.Difficulties[d].ID
}

// UnlockedLevel — сколько уровней мира открыто на сложности.
func (p *Profile) UnlockedLevel(w sim.World, d sim.Difficulty) int {
	if n := p.Unlocked[unlockKey(w, d)]; n > 0 {
		return n
	}
	return 1
}

// Complete отмечает прохождение трассы: медаль и открытие следующего уровня.
func (p *Profile) Complete(t sim.Track, m sim.Medal) {
	if m > p.Medals[t.Key()] {
		p.Medals[t.Key()] = m
	}
	if t.Daily != "" {
		return
	}
	k := unlockKey(t.World, t.Diff)
	if next := t.Level + 1; next <= sim.LevelsPerWorld && next > p.Unlocked[k] {
		p.Unlocked[k] = next
	}
}

// LevelsToUnlockWorld — сколько уровней мира нужно пройти (на любой
// сложности), чтобы открылся следующий мир.
const LevelsToUnlockWorld = 4

// WorldOpen — открыт ли мир. Первый открыт всегда, каждый следующий —
// после LevelsToUnlockWorld пройденных уровней предыдущего.
func (p *Profile) WorldOpen(w sim.World) bool {
	if w <= sim.Mine {
		return true
	}
	for d := sim.Difficulty(0); d < sim.DifficultyCount; d++ {
		if p.UnlockedLevel(w-1, d) > LevelsToUnlockWorld {
			return true
		}
	}
	return false
}
