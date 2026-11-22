// Package controls хранит раскладки клавиш: какие клавиши отвечают за
// движение, рывок и способность в одиночной игре и у каждого игрока дуэли.
//
// Клавиши записаны строками в формате Ebitengine (Key.String(): "W",
// "ArrowUp", "Space", "ShiftLeft"…), поэтому пакет не зависит от движка и
// проверяется обычными тестами. Перевод строк в ebiten.Key делает пакет game.
package controls

// Action — игровое действие.
type Action int

const (
	Up Action = iota
	Down
	Left
	Right
	Dash
	Ability
	ActionCount
)

// ActionNames — подписи действий в настройках.
var ActionNames = [ActionCount]string{"Вверх", "Вниз", "Влево", "Вправо", "Рывок", "Способность"}

// Slots — сколько клавиш можно назначить на одно действие.
const Slots = 2

// Layout — раскладка одного игрока: до двух клавиш на действие ("" — пусто).
type Layout [ActionCount][Slots]string

// Раскладки.
const (
	Solo  = iota // одиночная игра
	Duel1        // левый игрок дуэли
	Duel2        // правый игрок дуэли
	LayoutCount
)

// LayoutNames — подписи раскладок.
var LayoutNames = [LayoutCount]string{"Одиночная игра", "Дуэль: игрок 1", "Дуэль: игрок 2"}

// LayoutIDs — ключи раскладок в профиле.
var LayoutIDs = [LayoutCount]string{"solo", "duel1", "duel2"}

var actionIDs = [ActionCount]string{"up", "down", "left", "right", "dash", "ability"}

// Defaults — раскладки по умолчанию.
func Defaults() [LayoutCount]Layout {
	return [LayoutCount]Layout{
		Solo: {
			Up: {"W", "ArrowUp"}, Down: {"S", "ArrowDown"}, Left: {"A", "ArrowLeft"}, Right: {"D", "ArrowRight"},
			Dash: {"Space", "ShiftLeft"}, Ability: {"E", "Q"},
		},
		Duel1: {
			Up: {"W"}, Down: {"S"}, Left: {"A"}, Right: {"D"},
			Dash: {"Space", "ShiftLeft"}, Ability: {"E", "Q"},
		},
		Duel2: {
			Up: {"ArrowUp"}, Down: {"ArrowDown"}, Left: {"ArrowLeft"}, Right: {"ArrowRight"},
			Dash: {"ShiftRight", "Enter"}, Ability: {"ControlRight", "Slash"},
		},
	}
}

// Set назначает клавишу key на действие a в ячейку slot раскладки l.
// Если клавиша уже занята другим действием этой раскладки, она оттуда
// снимается. В дуэли одна клавиша не может принадлежать обоим игрокам,
// поэтому у соперника она тоже снимается. Возвращает подпись действия,
// у которого клавишу забрали ("" — ни у кого).
func Set(all *[LayoutCount]Layout, l int, a Action, slot int, key string) (taken string) {
	if key == "" {
		all[l][a][slot] = ""
		return ""
	}
	clear := func(li int) {
		for ai := Action(0); ai < ActionCount; ai++ {
			for si := 0; si < Slots; si++ {
				if all[li][ai][si] == key && !(li == l && ai == a && si == slot) {
					all[li][ai][si] = ""
					if li == l {
						taken = ActionNames[ai]
					} else {
						taken = LayoutNames[li] + ", " + ActionNames[ai]
					}
				}
			}
		}
	}
	clear(l)
	if l == Duel1 {
		clear(Duel2)
	} else if l == Duel2 {
		clear(Duel1)
	}
	all[l][a][slot] = key
	return taken
}

// Empty — действия раскладки, на которые не назначено ни одной клавиши.
func (lay *Layout) Empty() []Action {
	var out []Action
	for a := Action(0); a < ActionCount; a++ {
		if lay[a][0] == "" && lay[a][1] == "" {
			out = append(out, a)
		}
	}
	return out
}

// First — первая назначенная клавиша действия (для подсказок), "" — нет.
func (lay *Layout) First(a Action) string {
	for _, k := range lay[a] {
		if k != "" {
			return k
		}
	}
	return ""
}

// ToMap переводит раскладки в словарь для профиля: "solo.dash" → ["Space", "ShiftLeft"].
// Сохраняются только раскладки, отличные от умолчания.
func ToMap(all [LayoutCount]Layout) map[string][]string {
	def := Defaults()
	m := map[string][]string{}
	for l := 0; l < LayoutCount; l++ {
		if all[l] == def[l] {
			continue
		}
		for a := Action(0); a < ActionCount; a++ {
			m[LayoutIDs[l]+"."+actionIDs[a]] = []string{all[l][a][0], all[l][a][1]}
		}
	}
	return m
}

// FromMap восстанавливает раскладки из профиля. Неизвестные ключи и
// клавиши, которых нет в valid, пропускаются; раскладка без записей в
// профиле остаётся по умолчанию.
func FromMap(m map[string][]string, valid func(string) bool) [LayoutCount]Layout {
	all := Defaults()
	for l := 0; l < LayoutCount; l++ {
		has := false
		for a := Action(0); a < ActionCount; a++ {
			if _, ok := m[LayoutIDs[l]+"."+actionIDs[a]]; ok {
				has = true
			}
		}
		if !has {
			continue
		}
		all[l] = Layout{}
		for a := Action(0); a < ActionCount; a++ {
			keys := m[LayoutIDs[l]+"."+actionIDs[a]]
			for s := 0; s < Slots && s < len(keys); s++ {
				if keys[s] != "" && valid(keys[s]) {
					all[l][a][s] = keys[s]
				}
			}
		}
	}
	return all
}
