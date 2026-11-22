// Package achieve — достижения. Условия проверяются по итогам забега и по
// прогрессу профиля, графики здесь нет. Идентификаторы стабильные: потом
// их можно один в один завести как достижения Steam.
package achieve

import "github.com/brokendoto/brokengame/internal/sim"

// ID — идентификатор достижения.
type ID string

// Info — описание достижения.
type Info struct {
	ID     ID
	Name   string
	Desc   string
	Hidden bool // описание скрыто, пока не получено
}

const (
	FirstWin   ID = "first_win"
	Gold       ID = "gold"
	NoHit      ID = "no_hit"
	MaxCombo   ID = "max_combo"
	Speedrun   ID = "speedrun"
	HardFinale ID = "hard_finale"
	Daily      ID = "daily"
	BeatGhost  ID = "beat_ghost"
	DuelWin    ID = "duel_win"
	MineAll    ID = "world_mine"
	ForestAll  ID = "world_forest"
	TempleAll  ID = "world_temple"
	FrontAll   ID = "world_front"
	SpaceAll   ID = "world_space"
	GoldWorld  ID = "gold_world"
	AllGolden  ID = "all_golden"
	PureSoul   ID = "pure_soul"
	EventHoriz ID = "event_horizon"
	Robot      ID = "robot"
	Marathon5  ID = "marathon_5"
	Marathon15 ID = "marathon_15"
)

// List — все достижения в порядке показа.
var List = []Info{
	{FirstWin, "Первый выход", "Пройди любой уровень", false},
	{Gold, "Золотая лихорадка", "Получи золотую медаль", false},
	{NoHit, "Ни царапины", "Пройди уровень, ни разу не получив удар", false},
	{MaxCombo, "Серия ×3", "Собери 9 предметов подряд без перерыва", false},
	{Speedrun, "Быстрее эталона", "Пройди уровень быстрее эталонного времени", false},
	{HardFinale, "Хардкор", "Пройди 8-й уровень любого мира на сложной", false},
	{Daily, "Испытание принято", "Пройди испытание дня", false},
	{BeatGhost, "Обогнать призрака", "Набери больше очков, чем призрак соперника", false},
	{DuelWin, "Дуэлянт", "Выиграй дуэль", false},
	{MineAll, "Покоритель шахты", "Пройди все уровни Кристальной шахты", false},
	{ForestAll, "Хозяйка леса", "Пройди все уровни Осеннего леса", false},
	{TempleAll, "Путь самурая", "Пройди все уровни Ночного храма", false},
	{FrontAll, "Ас передовой", "Пройди все уровни Руин передовой", false},
	{SpaceAll, "Звёздный странник", "Пройди все уровни Открытого космоса", false},
	{Marathon5, "Марафонец", "Пройди 5 этапов марафона за один забег", false},
	{Marathon15, "Железная воля", "Пройди 15 этапов марафона за один забег", false},
	{GoldWorld, "Золотой мир", "Получи золото на всех уровнях одного мира", false},
	{AllGolden, "Ни жёлудя врагу", "Собери все золотые жёлуди на уровне леса", false},
	{PureSoul, "Чистая душа", "Пройди уровень храма, не тронув проклятых душ", false},
	{EventHoriz, "Горизонт событий", "Узнай, что внутри чёрной дыры", true},
	{Robot, "Я, робот", "Посмотри, как играет автопилот", true},
}

// ByID — описание по идентификатору.
func ByID(id ID) (Info, bool) {
	for _, a := range List {
		if a.ID == id {
			return a, true
		}
	}
	return Info{}, false
}

var worldAll = [sim.WorldCount]ID{MineAll, ForestAll, TempleAll, FrontAll, SpaceAll}

// Run — что произошло в одиночном забеге (итог и счётчики событий).
type Run struct {
	Track       sim.Track
	Result      sim.Result
	ParTicks    int
	GoldenTotal int
	GoldenTaken int
	CursedTotal int
	CursedTaken int
	HoleHit     bool
	GhostScore  int // очки призрака; 0 — призрака не было
	UsedBot     bool
	Medals      map[string]sim.Medal // медали профиля уже с учётом этого забега
}

// CheckRun возвращает достижения, заработанные забегом (в том числе уже
// полученные раньше: отсеивает их Unlock).
func CheckRun(r Run) []ID {
	if r.UsedBot {
		return []ID{Robot}
	}
	var out []ID
	if r.HoleHit {
		out = append(out, EventHoriz)
	}
	if r.Result.BestCombo >= 9 {
		out = append(out, MaxCombo)
	}
	if !r.Result.Won {
		return out
	}
	out = append(out, FirstWin)
	res := r.Result
	if res.Medal == sim.Gold {
		out = append(out, Gold)
	}
	if res.NoHit > 0 {
		out = append(out, NoHit)
	}
	if r.ParTicks > 0 && res.Ticks <= r.ParTicks {
		out = append(out, Speedrun)
	}
	if r.Track.Daily != "" {
		out = append(out, Daily)
	} else if r.Track.Marathon == "" && r.Track.Level == sim.LevelsPerWorld && r.Track.Diff == sim.Hard {
		out = append(out, HardFinale)
	}
	if r.GhostScore > 0 && res.Total > r.GhostScore {
		out = append(out, BeatGhost)
	}
	if r.GoldenTotal > 0 && r.GoldenTaken == r.GoldenTotal {
		out = append(out, AllGolden)
	}
	if r.CursedTotal > 0 && r.CursedTaken == 0 {
		out = append(out, PureSoul)
	}
	for w := sim.Mine; w < sim.WorldCount; w++ {
		done, gold := WorldProgress(r.Medals, w)
		if done == sim.LevelsPerWorld {
			out = append(out, worldAll[w])
		}
		if gold == sim.LevelsPerWorld {
			out = append(out, GoldWorld)
		}
	}
	return out
}

// CheckMarathon — достижения за число пройденных этапов марафона.
func CheckMarathon(stages int) []ID {
	var out []ID
	if stages >= 5 {
		out = append(out, Marathon5)
	}
	if stages >= 15 {
		out = append(out, Marathon15)
	}
	return out
}

// WorldProgress — сколько уровней мира пройдено и сколько на золото
// (на любой сложности).
func WorldProgress(medals map[string]sim.Medal, w sim.World) (done, gold int) {
	for l := 1; l <= sim.LevelsPerWorld; l++ {
		best := sim.NoMedal
		for d := sim.Difficulty(0); d < sim.DifficultyCount; d++ {
			if m := medals[sim.Track{World: w, Level: l, Diff: d}.Key()]; m > best {
				best = m
			}
		}
		if best > sim.NoMedal {
			done++
		}
		if best == sim.Gold {
			gold++
		}
	}
	return
}

// Progress — прогресс для достижений-счётчиков («5 из 8»); ok=false, если
// у достижения нет счётчика.
func Progress(id ID, medals map[string]sim.Medal) (cur, total int, ok bool) {
	for w, wid := range worldAll {
		if id == wid {
			done, _ := WorldProgress(medals, sim.World(w))
			return done, sim.LevelsPerWorld, true
		}
	}
	if id == GoldWorld {
		best := 0
		for w := sim.Mine; w < sim.WorldCount; w++ {
			_, g := WorldProgress(medals, w)
			best = max(best, g)
		}
		return best, sim.LevelsPerWorld, true
	}
	return 0, 0, false
}

// Unlock отмечает достижения в наборе got (id → дата) и возвращает новые.
func Unlock(got map[string]string, ids []ID, date string) []ID {
	var fresh []ID
	for _, id := range ids {
		if _, ok := got[string(id)]; ok {
			continue
		}
		if _, known := ByID(id); !known {
			continue
		}
		got[string(id)] = date
		fresh = append(fresh, id)
	}
	return fresh
}
