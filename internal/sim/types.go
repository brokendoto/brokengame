// Package sim — игровая логика «Пятимирья» без графики и звука.
//
// Симуляция идёт фиксированными шагами (TPS раз в секунду) и зависит только
// от уровня и последовательности нажатий. Это даёт:
//   - честные таблицы рекордов: любой результат можно перепроверить по записи нажатий;
//   - «призраков»: лучший забег воспроизводится параллельно с игроком;
//   - испытание дня: у всех игроков в один день одинаковый уровень.
package sim

import "fmt"

// Version меняется при любом изменении правил, влияющем на результат.
// Записи забегов старых версий не используются как призраки.
const Version = 2

const (
	TPS = 60 // шагов симуляции в секунду
	W   = 25 // ширина поля в клетках
	H   = 13 // высота поля в клетках
)

// Input — состояние кнопок игрока на одном шаге (битовая маска).
type Input uint8

const (
	InUp Input = 1 << iota
	InDown
	InLeft
	InRight
	InDash
	InAbility
)

// Difficulty — уровень сложности.
type Difficulty int

const (
	Easy Difficulty = iota
	Normal
	Hard
	DifficultyCount
)

// DiffParams — параметры сложности.
type DiffParams struct {
	Name       string
	ID         string
	Lives      int
	EnemySpeed float64
	TimeMul    float64
}

var Difficulties = [DifficultyCount]DiffParams{
	{Name: "Лёгкая", ID: "easy", Lives: 4, EnemySpeed: 0.85, TimeMul: 1.3},
	{Name: "Обычная", ID: "normal", Lives: 3, EnemySpeed: 1.0, TimeMul: 1.0},
	{Name: "Сложная", ID: "hard", Lives: 2, EnemySpeed: 1.15, TimeMul: 0.85},
}

// World — один из пяти миров. В этой версии готова шахта.
type World int

const (
	Mine World = iota
	Forest
	Temple
	Front
	Space
	WorldCount
)

// WorldInfo — описание мира для меню.
type WorldInfo struct {
	ID, Name, Hero, Item, ItemsGen, ItemsPl string
	Ready                                   bool
}

var Worlds = [WorldCount]WorldInfo{
	{ID: "mine", Name: "Кристальная шахта", Hero: "Шахтёр", Item: "кристалл", ItemsGen: "кристаллов", ItemsPl: "кристаллы", Ready: true},
	{ID: "forest", Name: "Осенний лес", Hero: "Белочка", Item: "жёлудь", ItemsGen: "желудей", ItemsPl: "жёлуди", Ready: true},
	{ID: "temple", Name: "Ночной храм", Hero: "Ронин", Item: "душа", ItemsGen: "душ", ItemsPl: "души", Ready: true},
	{ID: "front", Name: "Руины передовой", Hero: "Пилот", Item: "энергоядро", ItemsGen: "энергоядер", ItemsPl: "энергоядра", Ready: true},
	{ID: "space", Name: "Открытый космос", Hero: "Космонавт", Item: "звезда", ItemsGen: "звёзд", ItemsPl: "звёзды", Ready: true},
}

// LevelsPerWorld — число уровней кампании в каждом мире.
const LevelsPerWorld = 8

// DailyLevel — какой уровень по сложности соответствует испытанию дня.
const DailyLevel = 6

// Track — «трасса»: всё, что однозначно задаёт уровень. У каждой трассы своя
// таблица рекордов.
type Track struct {
	World World
	Level int // 1..LevelsPerWorld
	Diff  Difficulty
	Daily string // дата ГГГГ-ММ-ДД для испытания дня, иначе пусто

	// Марафон: неделя (ГГГГ-Wнн) и номер этапа с 1. Пусто — не марафон.
	Marathon string
	Stage    int
}

// Key — строковый ключ трассы для таблиц рекордов.
func (t Track) Key() string {
	if t.Marathon != "" {
		return "marathon-" + t.Marathon // одна таблица на весь марафон недели
	}
	if t.Daily != "" {
		return "daily-" + t.Daily
	}
	return fmt.Sprintf("%s-%02d-%s", Worlds[t.World].ID, t.Level, Difficulties[t.Diff].ID)
}

// Seed — зерно генерации. Не зависит от компьютера, поэтому уровень
// одинаков у всех игроков (важно для будущих онлайн-таблиц Steam).
func (t Track) Seed() uint64 {
	if t.Marathon != "" {
		return Hash(fmt.Sprintf("pyatimirye/marathon/%s/%d", t.Marathon, t.Stage))
	}
	if t.Daily != "" {
		return Hash("pyatimirye/daily/" + t.Daily)
	}
	// Сложность намеренно не входит в зерно: карта уровня одна,
	// меняются только скорость врагов, жизни и время.
	return Hash(fmt.Sprintf("pyatimirye/%s/%d", Worlds[t.World].ID, t.Level))
}

// Title — человекочитаемое название трассы.
func (t Track) Title() string {
	if t.Marathon != "" {
		return fmt.Sprintf("Марафон · этап %d · %s", t.Stage, Worlds[t.World].Name)
	}
	if t.Daily != "" {
		return "Испытание дня " + t.Daily + " · " + Worlds[t.World].Name
	}
	return fmt.Sprintf("%s · уровень %d · %s", Worlds[t.World].Name, t.Level, Difficulties[t.Diff].Name)
}

// Pt — клетка поля.
type Pt struct{ X, Y int }

// Cell — тип клетки.
type Cell uint8

const (
	Floor Cell = iota
	Wall
	Mud // замедляет (грязь, лужи)
	Fog // управление наоборот (туман)
	Ice // скольжение (масло, невесомость)
)

// Medal — награда за прохождение уровня.
type Medal int

const (
	NoMedal Medal = iota
	Bronze
	Silver
	Gold
)

var MedalNames = [...]string{"—", "Бронза", "Серебро", "Золото"}
