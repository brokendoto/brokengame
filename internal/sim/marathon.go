package sim

import (
	"fmt"
	"time"
)

// Марафон — бесконечная череда уровней по всем пяти мирам. Жизни
// переносятся с этапа на этап, очки складываются, уровни постепенно
// усложняются. Набор этапов задаётся неделей: всю неделю у всех игроков
// одинаковые уровни, поэтому результаты можно честно сравнивать.

// MarathonLives — сколько жизней в начале марафона (и больше не бывает).
const MarathonLives = 3

// MarathonHealEvery — каждые столько пройденных этапов возвращается жизнь.
const MarathonHealEvery = 5

// MarathonWeek — неделя марафона для момента t (по UTC), вида 2026-W47.
func MarathonWeek(t time.Time) string {
	y, w := t.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// MarathonTrack — трасса этапа stage (с 1) марафона недели week. Миры идут
// по кругу, начиная с недельного; уровень растёт каждые два этапа, а с
// 13-го этапа включается сложная сложность.
func MarathonTrack(week string, stage int) Track {
	start := int(Hash("pyatimirye/marathon/"+week) % uint64(WorldCount))
	t := Track{
		World:    World((start + stage - 1) % int(WorldCount)),
		Level:    min(LevelsPerWorld, 2+(stage-1)/2),
		Diff:     Normal,
		Marathon: week,
		Stage:    stage,
	}
	if stage > 12 {
		t.Diff = Hard
	}
	return t
}

// MarathonLevel строит уровень этапа. Предел жизней в марафоне общий,
// независимо от сложности этапа.
func MarathonLevel(week string, stage int) *Level {
	lv := Generate(MarathonTrack(week, stage))
	lv.Lives = MarathonLives
	return lv
}

// NewMarathonState начинает этап с жизнями, оставшимися с прошлого.
func NewMarathonState(lv *Level, lives int) *State {
	s := NewState(lv)
	s.Lives = lives
	return s
}

// MarathonCarry — сколько жизней будет на следующем этапе после того, как
// пройдено stagesDone этапов и осталось lives жизней.
func MarathonCarry(lives, stagesDone int) int {
	if stagesDone%MarathonHealEvery == 0 && lives < MarathonLives {
		lives++
	}
	return lives
}

// MarathonResult — итог марафона.
type MarathonResult struct {
	Stages int // пройдено этапов
	Score  int
	Ticks  int // суммарное время пройденных этапов
}

// RunMarathon проигрывает записи этапов подряд и подсчитывает итог. Им
// проверяются результаты из таблицы, как и у обычных уровней.
func RunMarathon(week string, stages [][]Input) MarathonResult {
	var r MarathonResult
	lives := MarathonLives
	for i, in := range stages {
		lv := MarathonLevel(week, i+1)
		s := NewMarathonState(lv, lives)
		for _, x := range in {
			if s.Status != Playing {
				break
			}
			s.Step(x)
		}
		res := s.Result()
		if !res.Won {
			break
		}
		r.Stages++
		r.Score += res.Total
		r.Ticks += res.Ticks
		lives = MarathonCarry(s.Lives, r.Stages)
	}
	return r
}
