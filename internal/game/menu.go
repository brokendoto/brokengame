package game

import (
	"math"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/achieve"
	"github.com/brokendoto/brokengame/internal/platform"
	"github.com/brokendoto/brokengame/internal/save"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// --- Главное меню ------------------------------------------------------------

type menuItem int

const (
	miCampaign menuItem = iota
	miDaily
	miMarathon
	miDuel
	miRecords
	miDiff
	miPlayer
	miHero
	miAchievements
	miSettings
	miQuit
)

type menuScene struct {
	sel   int
	nav   nav
	items []menuItem
	t     int
	hits  hits
}

func newMenuScene() *menuScene {
	m := &menuScene{items: []menuItem{miCampaign, miDaily, miMarathon, miDuel, miRecords, miAchievements, miHero, miDiff, miPlayer, miSettings}}
	if CanQuit() {
		m.items = append(m.items, miQuit)
	}
	return m
}

func (m *menuScene) label(a *App, it menuItem) string {
	switch it {
	case miCampaign:
		return "Играть: " + sim.Worlds[a.Prof.World].Name
	case miDaily:
		return "Испытание дня"
	case miMarathon:
		return "Марафон"
	case miDuel:
		return "Дуэль вдвоём"
	case miRecords:
		return "Таблицы рекордов"
	case miDiff:
		return "Сложность:  ‹ " + sim.Difficulties[a.Prof.Diff].Name + " ›"
	case miPlayer:
		return "Игрок: " + a.Prof.Name
	case miHero:
		return "Герой: " + skinName(a.Prof.Skin)
	case miAchievements:
		return "Достижения: " + itoa(len(a.Prof.Achievements)) + " из " + itoa(len(achieve.List))
	case miSettings:
		return "Настройки"
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
	activate := okPressed()
	if id, ok := m.hits.tapped(a); ok {
		m.sel, activate = id, true
		it = m.items[id]
	}
	if !activate {
		return
	}
	a.Sound.Play(sfx.Select)
	switch it {
	case miCampaign:
		a.Goto(newLevelScene(a))
	case miDaily:
		a.Goto(newPlayScene(a, sim.DailyTrack(today())))
	case miMarathon:
		a.Goto(newMarathonPlay(a, newMarathonRun()))
	case miDuel:
		a.Goto(newDuelSetupScene(a))
	case miRecords:
		a.Goto(newRecordsScene(a))
	case miDiff:
		a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
		_ = a.Prof.Save()
	case miPlayer:
		a.Goto(newNameScene(false))
	case miHero:
		a.Goto(newSkinScene(a))
	case miAchievements:
		a.Goto(newAchievementsScene())
	case miSettings:
		a.Goto(newSettingsScene())
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
	// По бокам — предметы всех пяти миров по очереди.
	wl := sim.World((a.Frame / 180) % int(sim.WorldCount))
	wr := (wl + 2) % sim.WorldCount
	drawItem(dst, wl, ScreenW/2-330, 118+math.Sin(t*0.05)*5, 1.4, t, 0)
	drawItem(dst, wr, ScreenW/2+330, 118+math.Sin(t*0.05+2)*5, 1.4, t+40, 0)
	drawText(dst, "Пять миров. Один рекорд.", 22, false, ScreenW/2, 176, Center, colDim)

	// Пункты меню.
	y0 := 220.0
	m.hits.reset()
	for i, it := range m.items {
		y := y0 + float64(i)*37
		m.hits.add(rect{ScreenW/2 - 300, y - 7, 600, 37}, i)
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
		if it == miCampaign || it == miDaily || it == miMarathon || it == miDuel {
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
	if m.items[m.sel] == miMarathon {
		top := a.Rec.Top(sim.MarathonTrack(sim.MarathonWeek(time.Now()), 1).Key())
		s := "Уровни подряд через все миры, жизни переносятся. Как далеко ты зайдёшь?"
		if len(top) > 0 {
			s = "Лидер недели: " + top[0].Name + " · " + itoa(top[0].Stages) + " " + stagesRu(top[0].Stages) + " · " + fmtScore(top[0].Score) + " очков"
		}
		drawText(dst, s, 18, false, ScreenW/2, ScreenH-70, Center, colGold)
	}
	if a.Pointer.touchMode {
		keyHint(dst, "Нажми на пункт меню")
	} else {
		keyHint(dst, "↑↓ — выбор   ·   Enter — ок   ·   ←→ — сложность   ·   F11 — полный экран   ·   M — звук")
	}
	drawText(dst, versionText(), 14, false, ScreenW-16, ScreenH-24, Right, rgba(90, 84, 100, 255))
}

// --- Выбор уровня ------------------------------------------------------------

type levelScene struct {
	world sim.World
	sel   int
	nav   nav
	hits  hits
}

const (
	hitBack = 100 + iota
	hitDiff
	hitWorldPrev
	hitWorldNext
	hitWorld = 200 // +номер мира
)

func newLevelScene(a *App) *levelScene {
	l := &levelScene{world: a.Prof.World}
	l.sel = a.Prof.UnlockedLevel(l.world, a.Prof.Diff) - 1
	if a.Prof.ShowAll {
		l.sel = 0 // в режиме показа открыто всё — начинаем с первого уровня
	}
	return l
}

// worldSwitch — смена мира клавишами Q/E, PageUp/PageDown или бамперами геймпада.
func worldSwitch() int {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyPageUp) || padJust(ebiten.StandardGamepadButtonFrontTopLeft) {
		return -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) || inpututil.IsKeyJustPressed(ebiten.KeyPageDown) || padJust(ebiten.StandardGamepadButtonFrontTopRight) {
		return 1
	}
	return 0
}

func (l *levelScene) setWorld(a *App, w sim.World) {
	if w == l.world {
		return
	}
	l.world = w
	l.sel = a.Prof.UnlockedLevel(w, a.Prof.Diff) - 1
	if a.Prof.ShowAll {
		l.sel = 0
	}
	if a.Prof.WorldOpen(w) {
		a.Prof.World = w
		_ = a.Prof.Save()
	}
	a.Sound.Play(sfx.Menu)
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
	if d := worldSwitch(); d != 0 {
		l.setWorld(a, (l.world+sim.World(d)+sim.WorldCount)%sim.WorldCount)
	}
	tap, tapped := l.hits.tapped(a)
	if tapped && tap == hitBack {
		a.Goto(newMenuScene())
		return
	}
	if tapped && tap >= hitWorld && tap < hitWorld+int(sim.WorldCount) {
		l.setWorld(a, sim.World(tap-hitWorld))
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || (tapped && tap == hitDiff) {
		a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
		_ = a.Prof.Save()
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newMenuScene())
		return
	}
	play := okPressed()
	if tapped && tap < sim.LevelsPerWorld {
		l.sel, play = tap, true
	}
	if play {
		if !a.Prof.WorldOpen(l.world) {
			a.Toast(lockedWorldHint(l.world))
			return
		}
		if l.sel+1 > a.Prof.UnlockedLevel(l.world, a.Prof.Diff) {
			a.Toast("Сначала пройди предыдущий уровень")
			return
		}
		a.Sound.Play(sfx.Select)
		a.Goto(newPlayScene(a, sim.Track{World: l.world, Level: l.sel + 1, Diff: a.Prof.Diff}))
	}
}

func lockedWorldHint(w sim.World) string {
	return "Пройди " + itoa(save.LevelsToUnlockWorld) + " уровня мира «" + sim.Worlds[w-1].Name + "», чтобы открыть этот мир"
}

func (l *levelScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	th := &themes[l.world]
	glow(dst, ScreenW/2, 400, 700, th.itemGlow, 0.08)
	l.hits.reset()
	t := float64(a.Frame)
	// Пять миров — вкладки.
	for i, w := range sim.Worlds {
		wi := sim.World(i)
		x := 140 + float64(i)*250
		open := a.Prof.WorldOpen(wi)
		sel := wi == l.world
		c, bg := colDim, rgba(30, 26, 40, 230)
		if sel {
			c, bg = colText, lerpColor(rgba(60, 46, 96, 240), themes[wi].wall, 0.35)
		}
		roundRect(dst, x-115, 20, 230, 60, 12, bg)
		if sel {
			roundRectStroke(dst, x-115, 20, 230, 60, 12, 2, themes[wi].itemGlow)
		}
		l.hits.add(rect{x - 115, 14, 230, 72}, hitWorld+i)
		if open {
			drawItem(dst, wi, x-90, 50, 0.55, t, 0)
		} else {
			fillRect(dst, x-99, 48, 18, 14, rgba(90, 84, 100, 255))
			strokeCircle(dst, x-90, 47, 6, 3, rgba(90, 84, 100, 255))
		}
		drawText(dst, w.Name, 17, sel, x+10, 28, Center, c)
		sub := w.Hero + " · " + sim.Abilities[sim.WorldRules[wi].Ability].Name
		if !open {
			sub = "закрыт"
		}
		drawText(dst, sub, 14, false, x+10, 52, Center, colDim)
	}
	diff := sim.Difficulties[a.Prof.Diff]
	diffHint := "  (Tab — сменить)"
	if a.Pointer.touchMode {
		diffHint = "  (нажми, чтобы сменить)"
	}
	drawText(dst, "Сложность: "+diff.Name+diffHint, 18, false, ScreenW/2, 96, Center, colDim)
	l.hits.add(rect{ScreenW/2 - 250, 88, 500, 40}, hitDiff)

	if !a.Prof.WorldOpen(l.world) {
		drawPanel(dst, ScreenW/2-360, 200, 720, 320)
		fillRect(dst, ScreenW/2-30, 260, 60, 48, rgba(90, 84, 100, 255))
		strokeCircle(dst, ScreenW/2, 258, 20, 7, rgba(90, 84, 100, 255))
		drawText(dst, sim.Worlds[l.world].Name+" пока закрыт", 28, true, ScreenW/2, 340, Center, colText)
		drawText(dst, lockedWorldHint(l.world), 18, false, ScreenW/2, 390, Center, colDim)
		r := sim.WorldRules[l.world]
		drawText(dst, "Здесь: "+r.ZoneName+", способность «"+sim.Abilities[r.Ability].Name+"»", 18, false, ScreenW/2, 430, Center, colAccent)
	} else {
		l.drawCards(a, dst)
	}
	if a.Pointer.touchMode {
		drawBackButton(dst, &l.hits, hitBack)
	} else {
		keyHint(dst, "Стрелки — уровень   ·   Q/E — мир   ·   Enter — играть   ·   Tab — сложность   ·   Esc — назад")
	}
}

func (l *levelScene) drawCards(a *App, dst *ebiten.Image) {
	th := &themes[l.world]
	unlocked := a.Prof.UnlockedLevel(l.world, a.Prof.Diff)
	for i := 0; i < sim.LevelsPerWorld; i++ {
		x := 100 + float64(i%4)*280
		y := 140 + float64(i/4)*250
		tr := sim.Track{World: l.world, Level: i + 1, Diff: a.Prof.Diff}
		locked := i+1 > unlocked
		l.hits.add(rect{x, y, 240, 220}, i)
		bg := lerpColor(rgba(32, 27, 42, 235), th.floorB, 0.3)
		if i == l.sel {
			bg = lerpColor(rgba(62, 48, 100, 245), th.wall, 0.3)
			glow(dst, x+120, y+110, 200, th.itemGlow, 0.15)
		}
		roundRect(dst, x, y, 240, 220, 16, bg)
		if i == l.sel {
			roundRectStroke(dst, x, y, 240, 220, 16, 3, th.itemGlow)
		}
		drawText(dst, "Уровень "+itoa(i+1), 26, true, x+120, y+16, Center, colText)
		if locked {
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
}

// --- Таблицы рекордов ---------------------------------------------------------

type recordsScene struct {
	world sim.World
	sel   int // 0 — испытание дня, 1..8 — уровни
	diff  sim.Difficulty
	nav   nav
	hits  hits
}

func newRecordsScene(a *App) *recordsScene {
	return &recordsScene{diff: a.Prof.Diff, world: a.Prof.World}
}

func (r *recordsScene) track() sim.Track {
	if r.sel == 0 {
		return sim.DailyTrack(today())
	}
	return sim.Track{World: r.world, Level: r.sel, Diff: r.diff}
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
	if d := worldSwitch(); d != 0 {
		r.world = (r.world + sim.World(d) + sim.WorldCount) % sim.WorldCount
		a.Sound.Play(sfx.Menu)
	}
	if id, ok := r.hits.tapped(a); ok {
		switch {
		case id == hitBack:
			a.Goto(newMenuScene())
			return
		case id == hitDiff:
			r.diff = (r.diff + 1) % sim.DifficultyCount
		case id == hitWorldPrev:
			r.world = (r.world + sim.WorldCount - 1) % sim.WorldCount
		case id == hitWorldNext:
			r.world = (r.world + 1) % sim.WorldCount
		default:
			r.sel = id
		}
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newMenuScene())
	}
}

func (r *recordsScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawText(dst, "Таблицы рекордов", 40, true, ScreenW/2, 24, Center, colText)
	drawPanel(dst, 60, 90, 320, 580)
	r.hits.reset()
	// Переключатель мира.
	drawText(dst, "‹", 30, true, 92, 100, Center, colAccent)
	drawText(dst, "›", 30, true, 348, 100, Center, colAccent)
	drawItem(dst, r.world, 130, 120, 0.5, float64(a.Frame), 0)
	drawText(dst, sim.Worlds[r.world].Name, 19, true, 228, 108, Center, colText)
	r.hits.add(rect{64, 94, 60, 52}, hitWorldPrev)
	r.hits.add(rect{316, 94, 60, 52}, hitWorldNext)
	for i := 0; i <= sim.LevelsPerWorld; i++ {
		y := 166 + float64(i)*54
		r.hits.add(rect{70, y - 10, 300, 52}, i)
		name := "Испытание дня"
		if i > 0 {
			name = themes[r.world].short + " · уровень " + itoa(i)
		}
		if i == r.sel {
			roundRect(dst, 74, y-8, 292, 44, 10, rgba(62, 48, 100, 245))
		}
		c := colDim
		if i == r.sel {
			c = colText
		}
		drawText(dst, name, 20, i == r.sel, 92, y, Left, c)
	}
	drawPanel(dst, 410, 90, 810, 580)
	tr := r.track()
	title := tr.Title()
	drawText(dst, title, 22, true, 815, 108, Center, colText)
	if r.sel > 0 {
		drawText(dst, "‹ ←  "+sim.Difficulties[r.diff].Name+"  → ›", 18, false, 815, 140, Center, colAccent)
		r.hits.add(rect{615, 130, 400, 40}, hitDiff)
	} else {
		drawText(dst, "Сложность всегда обычная — условия равны для всех", 16, false, 815, 142, Center, colDim)
	}
	drawLeaderboard(dst, a.Rec.Top(tr.Key()), 450, 190, 730, a.Prof.Name, 0)
	if a.Pointer.touchMode {
		drawBackButton(dst, &r.hits, hitBack)
	} else {
		keyHint(dst, "↑↓ — трасса   ·   Q/E — мир   ·   ←→ — сложность   ·   Esc — назад")
	}
}

// --- Ввод имени ---------------------------------------------------------------

type nameScene struct {
	name  []rune
	first bool
	buf   []rune
	t     int
	hits  hits
	latin bool
}

const maxName = 16

// Экранная клавиатура для телефонов (системную клавиатуру Ebitengine не вызывает).
var (
	oskRu = []string{"ЙЦУКЕНГШЩЗХ", "ФЫВАПРОЛДЖЭ", "ЯЧСМИТЬБЮЁЪ"}
	oskEn = []string{"QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM"}
)

const (
	keyBackspace = 1000 + iota
	keySpace
	keyLang
	keyDone
	keyBack
)

func newNameScene(first bool) *nameScene { return &nameScene{first: first} }

func (n *nameScene) typeRune(r rune) {
	if len(n.name) < maxName && r >= ' ' && utf8.ValidRune(r) {
		// После первой буквы — строчные, как обычно пишут имена.
		if len(n.name) > 0 && n.name[len(n.name)-1] != ' ' {
			r = unicode.ToLower(r)
		}
		n.name = append(n.name, r)
	}
}

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
	done := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) || padJust(ebiten.StandardGamepadButtonRightBottom)
	back := inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	if id, ok := n.hits.tapped(a); ok {
		a.Sound.Play(sfx.Menu)
		switch {
		case id == keyBackspace:
			if len(n.name) > 0 {
				n.name = n.name[:len(n.name)-1]
			}
		case id == keySpace:
			n.typeRune(' ')
		case id == keyLang:
			n.latin = !n.latin
		case id == keyDone:
			done = true
		case id == keyBack:
			back = true
		default:
			n.typeRune(rune(id))
		}
	}
	if back && !n.first {
		a.Goto(newMenuScene())
		return
	}
	if done {
		name := trimSpaces(string(n.name))
		if name == "" {
			a.Toast("Введи имя — оно появится в таблице рекордов")
			return
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
	n.hits.reset()
	touch := a.Pointer.touchMode
	top := 0.0
	if touch {
		top = -230 // клавиатура занимает низ экрана
	}
	if n.first && !touch {
		drawTextShadow(dst, "ПЯТИМИРЬЕ", 72, true, ScreenW/2, 110, Center, colText)
		drawText(dst, "Добро пожаловать в шахту!", 24, false, ScreenW/2, 210, Center, colDim)
	}
	drawPanel(dst, ScreenW/2-300, 280+top, 600, 200)
	drawText(dst, "Как тебя зовут?", 30, true, ScreenW/2, 302+top, Center, colText)
	roundRect(dst, ScreenW/2-240, 360+top, 480, 56, 10, rgba(14, 12, 20, 255))
	s := string(n.name)
	if (n.t/30)%2 == 0 {
		s += "_"
	}
	drawText(dst, s, 30, false, ScreenW/2, 368+top, Center, colAccent)
	if !touch {
		drawText(dst, "Имя будет в таблице рекордов. Несколько игроков на одном", 16, false, ScreenW/2, 500, Center, colDim)
		drawText(dst, "компьютере могут соревноваться, меняя игрока в меню.", 16, false, ScreenW/2, 522, Center, colDim)
		keyHint(dst, "Enter — готово")
		return
	}
	n.drawKeyboard(dst)
}

func (n *nameScene) drawKeyboard(dst *ebiten.Image) {
	rows := oskRu
	if n.latin {
		rows = oskEn
	}
	const kw, kh, gap = 96.0, 74.0, 8.0
	key := func(x, y, w float64, label string, id int, c rgbaT) {
		roundRect(dst, x, y, w, kh, 10, c)
		drawText(dst, label, 28, true, x+w/2, y+kh/2-17, Center, colText)
		n.hits.add(rect{x - gap/2, y - gap/2, w + gap, kh + gap}, id)
	}
	y := 270.0
	// Цифры.
	digits := "1234567890"
	x0 := (ScreenW - (float64(len(digits))*(kw+gap) - gap)) / 2
	for i, r := range digits {
		key(x0+float64(i)*(kw+gap), y, kw, string(r), int(r), rgba(36, 30, 50, 240))
	}
	y += kh + gap
	for _, row := range rows {
		rs := []rune(row)
		x0 := (ScreenW - (float64(len(rs))*(kw+gap) - gap)) / 2
		for i, r := range rs {
			key(x0+float64(i)*(kw+gap), y, kw, string(r), int(r), rgba(50, 42, 70, 240))
		}
		y += kh + gap
	}
	lang := "ABC"
	if n.latin {
		lang = "АБВ"
	}
	x := (ScreenW - (9*kw + 3*gap)) / 2
	key(x, y, kw*1.5, lang, keyLang, rgba(36, 30, 50, 240))
	x += kw*1.5 + gap
	key(x, y, kw*4, "пробел", keySpace, rgba(50, 42, 70, 240))
	x += kw*4 + gap
	key(x, y, kw*1.5, "←", keyBackspace, rgba(36, 30, 50, 240))
	x += kw*1.5 + gap
	key(x, y, kw*2, "Готово", keyDone, rgba(40, 110, 80, 255))
	if !n.first {
		drawBackButton(dst, &n.hits, keyBack)
	}
}

// Version — версия игры.
const Version = "0.13"

func versionText() string {
	v := "v" + Version + " · Go + Ebitengine"
	if n := platform.Current.Name(); n != "" {
		v += " · " + n
	}
	return v
}
