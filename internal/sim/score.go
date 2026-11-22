package sim

import "math"

// Очки за прохождение.
const (
	TimeBonusPerSec = 50
	LifeBonus       = 250
	NoHitBonus      = 1000
)

// Result — итог партии с разбивкой очков.
type Result struct {
	Won       bool
	Ticks     int
	Items     int // очки за кристаллы с учётом серий
	TimeBonus int
	LifeBonus int
	NoHit     int
	Total     int
	BestCombo int
	Medal     Medal
}

// Result подводит итог партии.
func (s *State) Result() Result {
	r := Result{Won: s.Status == Won, Ticks: s.FinishTick, Items: s.ItemScore, BestCombo: s.BestCombo}
	if !r.Won {
		return r
	}
	r.TimeBonus = (s.Lv.TimeLimit - s.FinishTick) * TimeBonusPerSec / TPS
	r.LifeBonus = s.Lives * LifeBonus
	if s.Hits == 0 {
		r.NoHit = NoHitBonus
	}
	r.Total = r.Items + r.TimeBonus + r.LifeBonus + r.NoHit
	r.Medal = s.Lv.Medal(s.FinishTick)
	return r
}

// Run проигрывает запись нажатий на уровне и возвращает итог.
// Так любой результат из таблицы рекордов можно перепроверить.
func Run(lv *Level, inputs []Input) Result {
	s := NewState(lv)
	for _, in := range inputs {
		if s.Status != Playing {
			break
		}
		s.Step(in)
	}
	return s.Result()
}

// Autopilot — простой бот: идёт к ближайшему кристаллу по кратчайшему пути,
// потом к выходу. Опасностей не избегает. Нужен для тестов (проверка, что
// уровни проходимы) и для фона главного меню.
func Autopilot(s *State) Input {
	var target Pt
	var d DistField
	cur := Pt{int(s.P.X), int(s.P.Y)}
	if s.ExitOpen {
		target = s.Lv.Exit
	} else {
		s.Lv.BFS(cur, &d)
		best := int16(math.MaxInt16)
		for _, it := range s.Items {
			if !it.Taken && d[it.P.Y][it.P.X] >= 0 && d[it.P.Y][it.P.X] < best {
				best, target = d[it.P.Y][it.P.X], it.P
			}
		}
	}
	s.Lv.BFS(target, &d)
	next := cur
	if cur != target {
		for _, dv := range dirs {
			n := Pt{cur.X + dv.X, cur.Y + dv.Y}
			if !s.Lv.Solid(n.X, n.Y) && d[n.Y][n.X] >= 0 && d[n.Y][n.X] < d[cur.Y][cur.X] {
				next = n
				break
			}
		}
	}
	tx, ty := float64(next.X)+0.5, float64(next.Y)+0.5
	var in Input
	const eps = 0.06
	dx, dy := tx-s.P.X, ty-s.P.Y
	// Сначала выравниваемся по оси, перпендикулярной движению, чтобы не цеплять углы.
	if next.X != cur.X && math.Abs(float64(cur.Y)+0.5-s.P.Y) > 0.15 {
		dx, dy = 0, float64(cur.Y)+0.5-s.P.Y
	} else if next.Y != cur.Y && math.Abs(float64(cur.X)+0.5-s.P.X) > 0.15 {
		dx, dy = float64(cur.X)+0.5-s.P.X, 0
	}
	if dx > eps {
		in |= InRight
	} else if dx < -eps {
		in |= InLeft
	}
	if dy > eps {
		in |= InDown
	} else if dy < -eps {
		in |= InUp
	}
	return in
}
