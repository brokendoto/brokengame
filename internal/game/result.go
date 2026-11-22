package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/controls"
	"github.com/brokendoto/brokengame/internal/replay"
	"github.com/brokendoto/brokengame/internal/save"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// resultScene — итоги уровня: разбивка очков, медаль, место в таблице,
// сравнение с призраком.
type resultScene struct {
	track    sim.Track
	lv       *sim.Level
	res      sim.Result
	reason   string
	name     string
	got      int
	total    int
	rank     int
	prevBest save.Entry
	hadPrev  bool
	ghost    save.Entry
	hasGhost bool
	top      []save.Entry
	t        int
	sel      int
	nav      nav
	items    []string
	hits     hits
}

func newResultScene(a *App, p *playScene) *resultScene {
	r := &resultScene{
		track:    p.track,
		lv:       p.lv,
		res:      p.st.Result(),
		reason:   p.st.LoseReason,
		got:      p.st.Collected,
		total:    len(p.st.Items),
		ghost:    p.ghostEntry,
		hasGhost: p.ghost != nil,
	}
	key := p.track.Key()
	r.name = a.Prof.Name
	if p.usedBot {
		r.name = BotName
	}
	r.prevBest, r.hadPrev = a.Rec.PersonalBest(key, r.name)
	if r.res.Won {
		r.rank = a.Records.Submit(key, save.Entry{
			Name:   r.name,
			Score:  r.res.Total,
			Ticks:  r.res.Ticks,
			Medal:  int(r.res.Medal),
			Replay: replay.Encode(p.rec),
		})
		if !p.usedBot {
			next := p.track.World + 1
			wasOpen := next >= sim.WorldCount || a.Prof.WorldOpen(next)
			a.Prof.Complete(p.track, r.res.Medal)
			_ = a.Prof.Save()
			if !wasOpen && a.Prof.WorldOpen(next) {
				a.Toast("Открыт новый мир: " + sim.Worlds[next].Name + "!")
				a.Sound.Play(sfx.ExitOpen)
			}
		}
	}
	r.top = a.Records.Top(key)
	if r.res.Won && p.track.Daily == "" && p.track.Level < sim.LevelsPerWorld {
		r.items = append(r.items, "Следующий уровень")
	}
	r.items = append(r.items, "Ещё раз", "Выйти в меню")
	return r
}

func (r *resultScene) Update(a *App) {
	r.t++
	if r.nav.dir(navUp) {
		r.sel = (r.sel + len(r.items) - 1) % len(r.items)
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navDown) {
		r.sel = (r.sel + 1) % len(r.items)
		a.Sound.Play(sfx.Menu)
	}
	if retryPressed() {
		a.Goto(newPlayScene(a, r.track))
		return
	}
	if backPressed() {
		a.Goto(newLevelScene(a))
		return
	}
	activate := okPressed()
	if id, ok := r.hits.tapped(a); ok {
		r.sel, activate = id, true
	}
	if activate && r.t > 15 {
		a.Sound.Play(sfx.Select)
		switch r.items[r.sel] {
		case "Следующий уровень":
			t := r.track
			t.Level++
			a.Goto(newPlayScene(a, t))
		case "Ещё раз":
			a.Goto(newPlayScene(a, r.track))
		default:
			if r.track.Daily != "" {
				a.Goto(newMenuScene())
			} else {
				a.Goto(newLevelScene(a))
			}
		}
	}
}

func (r *resultScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawText(dst, r.track.Title(), 20, false, ScreenW/2, 28, Center, colDim)

	// Левая панель — итог.
	drawPanel(dst, 60, 70, 640, 590)
	if !r.res.Won {
		drawText(dst, "Не получилось", 48, true, 380, 100, Center, colRed)
		drawText(dst, r.reason, 26, false, 380, 168, Center, colText)
		drawText(dst, "Собрано "+itoa(r.got)+" из "+itoa(r.total), 22, false, 380, 214, Center, colDim)
		drawText(dst, "Совет: рывок ("+hintKey(controls.Solo, controls.Dash)+") неуязвим — им можно проскочить", 18, false, 380, 290, Center, colDim)
		drawText(dst, "сквозь жука, а щит (E) отбрасывает врагов.", 18, false, 380, 316, Center, colDim)
	} else {
		drawText(dst, "Уровень пройден!", 44, true, 380, 92, Center, colGreen)
		drawMedal(dst, 380, 200, 26, r.res.Medal)
		drawText(dst, sim.MedalNames[r.res.Medal], 22, true, 380, 232, Center, medalColor(r.res.Medal))

		rows := []struct {
			name string
			v    int
		}{
			{"Кристаллы с сериями (лучшая ×" + itoa(r.res.BestCombo) + ")", r.res.Items},
			{"Бонус за оставшееся время", r.res.TimeBonus},
			{"Бонус за жизни", r.res.LifeBonus},
			{"Без единой царапины", r.res.NoHit},
		}
		y := 280.0
		for i, row := range rows {
			k := clamp(float64(r.t-i*12)/30, 0, 1)
			drawText(dst, row.name, 20, false, 100, y, Left, withAlpha(colText, k))
			drawText(dst, fmtScore(int(float64(row.v)*k)), 20, true, 660, y, Right, withAlpha(colText, k))
			y += 34
		}
		line(dst, 100, y+4, 660, y+4, 2, rgba(90, 80, 120, 255))
		k := clamp(float64(r.t-50)/40, 0, 1)
		drawText(dst, "ИТОГО", 28, true, 100, y+16, Left, colGold)
		drawText(dst, fmtScore(int(float64(r.res.Total)*k)), 34, true, 660, y+12, Right, colGold)

		drawText(dst, "Время: "+fmtTime(r.res.Ticks)+"   ·   золото до "+fmtTime(r.lv.GoldTicks())+", серебро до "+fmtTime(r.lv.SilverTicks()),
			17, false, 380, y+66, Center, colDim)

		// Соревновательная часть.
		msg, mc := "", colText
		switch {
		case r.hadPrev && r.res.Total > r.prevBest.Score:
			msg, mc = "Новый личный рекорд! +"+fmtScore(r.res.Total-r.prevBest.Score), colGold
		case !r.hadPrev:
			msg, mc = "Первый результат на этой трассе", colAccent
		default:
			msg, mc = "До личного рекорда не хватило "+fmtScore(r.prevBest.Score-r.res.Total), colDim
		}
		if r.t > 90 {
			pulse := 1.0
			if mc == colGold {
				pulse = 0.75 + 0.25*math.Sin(float64(r.t)*0.15)
			}
			drawText(dst, msg, 22, true, 380, y+100, Center, withAlpha(mc, pulse))
			if r.hasGhost && r.ghost.Name != r.name {
				d := r.res.Total - r.ghost.Score
				g := ""
				if d > 0 {
					g = "Ты обошёл призрака " + r.ghost.Name + " на " + fmtScore(d) + " очков!"
				} else {
					g = "Призрак " + r.ghost.Name + " впереди на " + fmtScore(-d) + " очков"
				}
				drawText(dst, g, 18, false, 380, y+132, Center, rgba(170, 210, 255, 255))
			}
		}
	}

	// Меню.
	r.hits.reset()
	for i, it := range r.items {
		y := 578 + float64(i)*27
		if len(r.items) == 2 {
			y += 15
		}
		r.hits.add(rect{100, y - 4, 560, 27}, i)
		c := colDim
		if i == r.sel {
			c = colAccent
			drawText(dst, "›", 22, true, 250, y-2, Center, colAccent)
		}
		drawText(dst, it, 19, i == r.sel, 380, y, Center, c)
	}

	// Правая панель — таблица.
	drawPanel(dst, 730, 70, 490, 590)
	drawText(dst, "Таблица рекордов", 26, true, 975, 90, Center, colText)
	drawLeaderboard(dst, r.top, 750, 140, 450, r.name, r.rank)
	if r.rank > 0 {
		drawText(dst, "Твоё место: "+itoa(r.rank), 22, true, 975, 600, Center, colGold)
	}
	if !a.Pointer.touchMode {
		keyHint(dst, "Enter — выбрать   ·   R — сразу ещё раз   ·   Esc — к уровням")
	}
}

// drawLeaderboard рисует таблицу рекордов.
func drawLeaderboard(dst *ebiten.Image, top []save.Entry, x, y, w float64, me string, highlight int) {
	if len(top) == 0 {
		drawText(dst, "Пока пусто — стань первым!", 20, false, x+w/2, y+60, Center, colDim)
		return
	}
	drawText(dst, "#", 15, false, x+10, y, Left, colDim)
	drawText(dst, "Игрок", 15, false, x+44, y, Left, colDim)
	drawText(dst, "Время", 15, false, x+w-130, y, Right, colDim)
	drawText(dst, "Очки", 15, false, x+w-10, y, Right, colDim)
	for i, e := range top {
		ry := y + 30 + float64(i)*38
		if i+1 == highlight {
			roundRect(dst, x, ry-6, w, 34, 8, rgba(80, 64, 30, 255))
		} else if e.Name == me {
			roundRect(dst, x, ry-6, w, 34, 8, rgba(40, 50, 70, 255))
		}
		c := colText
		if i < 3 {
			c = []rgbaT{colGold, colSilver, colBronze}[i]
		}
		drawText(dst, itoa(i+1), 20, true, x+10, ry, Left, c)
		drawMedal(dst, x+w-205, ry+11, 7, sim.Medal(e.Medal))
		name := e.Name
		if len([]rune(name)) > 14 {
			name = string([]rune(name)[:14]) + "…"
		}
		drawText(dst, name, 20, false, x+44, ry, Left, colText)
		drawText(dst, fmtTime(e.Ticks), 18, false, x+w-130, ry+1, Right, colDim)
		drawText(dst, fmtScore(e.Score), 20, true, x+w-10, ry, Right, colText)
	}
}
