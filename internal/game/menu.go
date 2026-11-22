package game

import (
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// --- Главное меню ------------------------------------------------------------

type menuItem int

const (
	miCampaign menuItem = iota
	miDaily
	miRecords
	miDiff
	miPlayer
	miGhost
	miSound
	miFullscreen
	miQuit
)

type menuScene struct {
	sel   int
	nav   nav
	items []menuItem
	t     int
}

func newMenuScene() *menuScene {
	m := &menuScene{items: []menuItem{miCampaign, miDaily, miRecords, miDiff, miPlayer, miGhost, miSound, miFullscreen}}
	if CanQuit() {
		m.items = append(m.items, miQuit)
	}
	return m
}

func (m *menuScene) label(a *App, it menuItem) string {
	on := func(b bool) string {
		if b {
			return "вкл"
		}
		return "выкл"
	}
	switch it {
	case miCampaign:
		return "Играть: " + sim.Worlds[sim.Mine].Name
	case miDaily:
		return "Испытание дня"
	case miRecords:
		return "Таблицы рекордов"
	case miDiff:
		return "Сложность:  ‹ " + sim.Difficulties[a.Prof.Diff].Name + " ›"
	case miPlayer:
		return "Игрок: " + a.Prof.Name
	case miGhost:
		return "Призрак соперника: " + on(a.Prof.ShowGhost)
	case miSound:
		return "Звук: " + on(!a.Sound.Muted)
	case miFullscreen:
		return "Полный экран: " + on(ebiten.IsFullscreen())
	case miQuit:
		return "Выход"
	}
	return ""
}

func (m *menuScene) Update(a *App) {
	m.t++
	if m.nav.dir(navUp) {
		m.sel = (m.sel + len(m.items) - 1) % len(m.items)
		a.Sound.Play(sfx.Menu)
	}
	if m.nav.dir(navDown) {
		m.sel = (m.sel + 1) % len(m.items)
		a.Sound.Play(sfx.Menu)
	}
	it := m.items[m.sel]
	if it == miDiff {
		d := 0
		if m.nav.dir(navLeft) {
			d = -1
		}
		if m.nav.dir(navRight) {
			d = 1
		}
		if d != 0 {
			a.Prof.Diff = (a.Prof.Diff + sim.Difficulty(d) + sim.DifficultyCount) % sim.DifficultyCount
			_ = a.Prof.Save()
			a.Sound.Play(sfx.Menu)
		}
	}
	if backPressed() && CanQuit() {
		m.sel = len(m.items) - 1
	}
	if !okPressed() {
		return
	}
	a.Sound.Play(sfx.Select)
	switch it {
	case miCampaign:
		a.Goto(newLevelScene(a))
	case miDaily:
		a.Goto(newPlayScene(a, sim.DailyTrack(today())))
	case miRecords:
		a.Goto(newRecordsScene(a))
	case miDiff:
		a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
		_ = a.Prof.Save()
	case miPlayer:
		a.Goto(newNameScene(false))
	case miGhost:
		a.Prof.ShowGhost = !a.Prof.ShowGhost
		_ = a.Prof.Save()
	case miSound:
		a.SetMuted(!a.Sound.Muted)
	case miFullscreen:
		a.SetFullscreen(!ebiten.IsFullscreen())
	case miQuit:
		a.Quit()
	}
}

func (m *menuScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	t := float64(a.Frame)
	// Заголовок с кристаллами по бокам.
	glow(dst, ScreenW/2, 120, 360, rgba(80, 60, 160, 255), 0.35)
	drawTextShadow(dst, "ПЯТИМИРЬЕ", 84, true, ScreenW/2, 70, Center, colText)
	drawCrystal(dst, ScreenW/2-330, 118+math.Sin(t*0.05)*5, 1.4, t)
	drawCrystal(dst, ScreenW/2+330, 118+math.Sin(t*0.05+2)*5, 1.4, t+40)
	drawText(dst, "Пять миров. Один рекорд.", 22, false, ScreenW/2, 176, Center, colDim)

	// Пункты меню.
	y0 := 236.0
	for i, it := range m.items {
		y := y0 + float64(i)*42
		sel := i == m.sel
		if sel {
			w := textWidth(m.label(a, it), 24, true) + 60
			roundRect(dst, ScreenW/2-w/2, y-6, w, 40, 12, rgba(60, 46, 96, 230))
			glow(dst, ScreenW/2, y+14, w*0.6, rgba(120, 90, 255, 255), 0.15)
		}
		c := colDim
		if sel {
			c = colText
		}
		if it == miCampaign || it == miDaily {
			if !sel {
				c = colAccent
			}
		}
		drawText(dst, m.label(a, it), 24, sel, ScreenW/2, y, Center, c)
	}

	// Подсказка испытания дня.
	if m.items[m.sel] == miDaily {
		top := a.Rec.Top(sim.DailyTrack(today()).Key())
		s := "Один уровень на весь день, одинаковый у всех. Побей лучший результат!"
		if len(top) > 0 {
			s = "Лидер дня: " + top[0].Name + " · " + fmtScore(top[0].Score) + " очков"
		}
		drawText(dst, s, 18, false, ScreenW/2, ScreenH-70, Center, colGold)
	}
	keyHint(dst, "↑↓ — выбор   ·   Enter — ок   ·   ←→ — сложность   ·   F11 — полный экран   ·   M — звук")
	drawText(dst, "v0.1 · Go + Ebitengine", 14, false, ScreenW-16, ScreenH-24, Right, rgba(90, 84, 100, 255))
}

// --- Выбор уровня ------------------------------------------------------------

type levelScene struct {
	sel int
	nav nav
}

func newLevelScene(a *App) *levelScene {
	return &levelScene{sel: a.Prof.UnlockedLevel(sim.Mine, a.Prof.Diff) - 1}
}

func (l *levelScene) Update(a *App) {
	n := sim.LevelsPerWorld
	moved := false
	if l.nav.dir(navLeft) && l.sel%4 > 0 {
		l.sel--
		moved = true
	}
	if l.nav.dir(navRight) && l.sel%4 < 3 && l.sel < n-1 {
		l.sel++
		moved = true
	}
	if l.nav.dir(navUp) && l.sel >= 4 {
		l.sel -= 4
		moved = true
	}
	if l.nav.dir(navDown) && l.sel+4 < n {
		l.sel += 4
		moved = true
	}
	if moved {
		a.Sound.Play(sfx.Menu)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
		_ = a.Prof.Save()
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newMenuScene())
		return
	}
	if okPressed() {
		if l.sel+1 > a.Prof.UnlockedLevel(sim.Mine, a.Prof.Diff) {
			a.Toast("Сначала пройди предыдущий уровень")
			return
		}
		a.Sound.Play(sfx.Select)
		a.Goto(newPlayScene(a, sim.Track{World: sim.Mine, Level: l.sel + 1, Diff: a.Prof.Diff}))
	}
}

func (l *levelScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	// Пять миров — вкладки, пока открыт первый.
	for i, w := range sim.Worlds {
		x := 140 + float64(i)*220
		c, bg := colDim, rgba(30, 26, 40, 230)
		if i == int(sim.Mine) {
			c, bg = colText, rgba(60, 46, 96, 240)
		}
		roundRect(dst, x-100, 24, 200, 54, 12, bg)
		drawText(dst, w.Name, 17, i == int(sim.Mine), x, 32, Center, c)
		sub := w.Hero
		if !w.Ready {
			sub = "скоро"
		}
		drawText(dst, sub, 14, false, x, 54, Center, colDim)
	}
	diff := sim.Difficulties[a.Prof.Diff]
	drawText(dst, "Сложность: "+diff.Name+"  (Tab — сменить)", 18, false, ScreenW/2, 96, Center, colDim)

	unlocked := a.Prof.UnlockedLevel(sim.Mine, a.Prof.Diff)
	for i := 0; i < sim.LevelsPerWorld; i++ {
		x := 100 + float64(i%4)*280
		y := 140 + float64(i/4)*250
		tr := sim.Track{World: sim.Mine, Level: i + 1, Diff: a.Prof.Diff}
		locked := i+1 > unlocked
		bg := rgba(32, 27, 42, 235)
		if i == l.sel {
			bg = rgba(62, 48, 100, 245)
			glow(dst, x+120, y+110, 200, rgba(120, 90, 255, 255), 0.18)
		}
		roundRect(dst, x, y, 240, 220, 16, bg)
		if i == l.sel {
			roundRectStroke(dst, x, y, 240, 220, 16, 3, colAccent)
		}
		drawText(dst, "Уровень "+itoa(i+1), 26, true, x+120, y+16, Center, colText)
		if locked {
			// Замок.
			fillRect(dst, x+100, y+100, 40, 32, rgba(90, 84, 100, 255))
			strokeCircle(dst, x+120, y+98, 14, 5, rgba(90, 84, 100, 255))
			drawText(dst, "закрыт", 18, false, x+120, y+150, Center, colDim)
			continue
		}
		m := a.Prof.Medals[tr.Key()]
		drawMedal(dst, x+120, y+100, 22, m)
		best, ok := a.Rec.PersonalBest(tr.Key(), a.Prof.Name)
		if ok {
			drawText(dst, "рекорд "+fmtScore(best.Score), 18, false, x+120, y+140, Center, colText)
			drawText(dst, fmtTime(best.Ticks), 16, false, x+120, y+166, Center, colDim)
		} else {
			drawText(dst, "ещё не пройден", 18, false, x+120, y+148, Center, colDim)
		}
		top := a.Rec.Top(tr.Key())
		if len(top) > 0 && top[0].Name != a.Prof.Name {
			drawText(dst, "лидер: "+top[0].Name, 14, false, x+120, y+192, Center, colGold)
		}
	}
	keyHint(dst, "Стрелки — выбор   ·   Enter — играть   ·   Tab — сложность   ·   Esc — назад")
}

// --- Таблицы рекордов ---------------------------------------------------------

type recordsScene struct {
	sel  int // 0 — испытание дня, 1..8 — уровни
	diff sim.Difficulty
	nav  nav
}

func newRecordsScene(a *App) *recordsScene { return &recordsScene{diff: a.Prof.Diff} }

func (r *recordsScene) track() sim.Track {
	if r.sel == 0 {
		return sim.DailyTrack(today())
	}
	return sim.Track{World: sim.Mine, Level: r.sel, Diff: r.diff}
}

func (r *recordsScene) Update(a *App) {
	if r.nav.dir(navUp) {
		r.sel = (r.sel + sim.LevelsPerWorld) % (sim.LevelsPerWorld + 1)
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navDown) {
		r.sel = (r.sel + 1) % (sim.LevelsPerWorld + 1)
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navLeft) {
		r.diff = (r.diff + sim.DifficultyCount - 1) % sim.DifficultyCount
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navRight) {
		r.diff = (r.diff + 1) % sim.DifficultyCount
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newMenuScene())
	}
}

func (r *recordsScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawText(dst, "Таблицы рекордов", 40, true, ScreenW/2, 24, Center, colText)
	drawPanel(dst, 60, 90, 320, 560)
	for i := 0; i <= sim.LevelsPerWorld; i++ {
		y := 110 + float64(i)*58
		name := "Испытание дня"
		if i > 0 {
			name = "Шахта · уровень " + itoa(i)
		}
		if i == r.sel {
			roundRect(dst, 74, y-8, 292, 46, 10, rgba(62, 48, 100, 245))
		}
		c := colDim
		if i == r.sel {
			c = colText
		}
		drawText(dst, name, 20, i == r.sel, 92, y, Left, c)
	}
	drawPanel(dst, 410, 90, 810, 560)
	tr := r.track()
	title := tr.Title()
	drawText(dst, title, 22, true, 815, 108, Center, colText)
	if r.sel > 0 {
		drawText(dst, "‹ ←  "+sim.Difficulties[r.diff].Name+"  → ›", 18, false, 815, 140, Center, colAccent)
	} else {
		drawText(dst, "Сложность всегда обычная — условия равны для всех", 16, false, 815, 142, Center, colDim)
	}
	drawLeaderboard(dst, a.Rec.Top(tr.Key()), 450, 190, 730, a.Prof.Name, 0)
	keyHint(dst, "↑↓ — трасса   ·   ←→ — сложность   ·   Esc — назад")
}

// --- Ввод имени ---------------------------------------------------------------

type nameScene struct {
	name  []rune
	first bool
	buf   []rune
	t     int
}

func newNameScene(first bool) *nameScene { return &nameScene{first: first} }

const maxName = 16

func (n *nameScene) Update(a *App) {
	n.t++
	if n.t == 1 && !n.first {
		n.name = []rune(a.Prof.Name)
	}
	n.buf = ebiten.AppendInputChars(n.buf[:0])
	for _, r := range n.buf {
		if len(n.name) < maxName && r >= ' ' && utf8.ValidRune(r) {
			n.name = append(n.name, r)
		}
	}
	if repeatKey(ebiten.KeyBackspace) && len(n.name) > 0 {
		n.name = n.name[:len(n.name)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && !n.first {
		a.Goto(newMenuScene())
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) || padJust(ebiten.StandardGamepadButtonRightBottom) {
		name := trimSpaces(string(n.name))
		if name == "" {
			if len(pads()) == 0 {
				a.Toast("Введи имя — оно появится в таблице рекордов")
				return
			}
			name = "Игрок"
		}
		a.Prof.Name = name
		_ = a.Prof.Save()
		a.Sound.Play(sfx.Select)
		a.Goto(newMenuScene())
	}
}

func repeatKey(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > 25 && d%4 == 0)
}

func trimSpaces(s string) string {
	r := []rune(s)
	for len(r) > 0 && r[0] == ' ' {
		r = r[1:]
	}
	for len(r) > 0 && r[len(r)-1] == ' ' {
		r = r[:len(r)-1]
	}
	return string(r)
}

func (n *nameScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	if n.first {
		drawTextShadow(dst, "ПЯТИМИРЬЕ", 72, true, ScreenW/2, 110, Center, colText)
		drawText(dst, "Добро пожаловать в шахту!", 24, false, ScreenW/2, 210, Center, colDim)
	}
	drawPanel(dst, ScreenW/2-300, 280, 600, 200)
	drawText(dst, "Как тебя зовут?", 30, true, ScreenW/2, 302, Center, colText)
	roundRect(dst, ScreenW/2-240, 360, 480, 56, 10, rgba(14, 12, 20, 255))
	s := string(n.name)
	if (n.t/30)%2 == 0 {
		s += "_"
	}
	drawText(dst, s, 30, false, ScreenW/2, 368, Center, colAccent)
	drawText(dst, "Имя будет в таблице рекордов. Несколько игроков на одном", 16, false, ScreenW/2, 500, Center, colDim)
	drawText(dst, "компьютере могут соревноваться, меняя игрока в меню.", 16, false, ScreenW/2, 522, Center, colDim)
	keyHint(dst, "Enter — готово")
}
