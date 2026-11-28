package game

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/achieve"
	"github.com/brokendoto/brokengame/internal/controls"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// Дуэль вдвоём на одном экране (или одном планшете). Правила — в sim/duel.go.

var duelNames = [2]string{"Игрок 1", "Игрок 2"}

// --- Выбор вида дуэли -----------------------------------------------------------

type duelModeScene struct {
	sel  sim.DuelMode
	nav  nav
	hits hits
}

const (
	hitModeBack = 600
	hitModeCard = 610 // +вид
)

func newDuelModeScene(a *App) *duelModeScene { return &duelModeScene{sel: a.duelMode} }

func (m *duelModeScene) Update(a *App) {
	n := sim.DuelModeCount
	if m.nav.dir(navLeft) || m.nav.dir(navUp) {
		m.sel = (m.sel + n - 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if m.nav.dir(navRight) || m.nav.dir(navDown) {
		m.sel = (m.sel + 1) % n
		a.Sound.Play(sfx.Menu)
	}
	start := okPressed()
	if tap, ok := m.hits.tapped(a); ok {
		switch {
		case tap == hitModeBack:
			a.Goto(newMenuScene())
			return
		case tap >= hitModeCard:
			// Первое касание выбирает, второе — открывает.
			t := sim.DuelMode(tap - hitModeCard)
			start = t == m.sel
			m.sel = t
			a.Sound.Play(sfx.Menu)
		}
	}
	if backPressed() {
		a.Goto(newMenuScene())
		return
	}
	if start {
		a.Sound.Play(sfx.Select)
		a.duelMode = m.sel
		a.Goto(newDuelSetupScene(a))
	}
}

// drawModeIcon — значок вида дуэли.
func drawModeIcon(dst *ebiten.Image, mode sim.DuelMode, x, y, t float64) {
	switch mode {
	case sim.DuelRounds:
		drawItem(dst, sim.Mine, x-34, y+6, 0.8, t, 0)
		drawItem(dst, sim.Mine, x+34, y+6, 0.8, t+40, 0)
		fillPoly(dst, []float64{x - 12, y - 4, x + 8, y - 4, x + 8, y - 12, x + 20, y + 2, x + 8, y + 16, x + 8, y + 8, x - 12, y + 8}, duelColors[0])
	case sim.DuelBank:
		for k := 0; k < 3; k++ {
			strokeCircle(dst, x, y, 30-float64(k)*9, 3, withAlpha(exitGlow, 0.9-0.25*float64(k)))
		}
		glow(dst, x, y, 50, exitGlow, 0.25+0.1*math.Sin(t*0.08))
		drawItem(dst, sim.Space, x+38, y-20, 0.5, t, 0)
	default:
		drawHeart(dst, x-22, y, 1.5, rgba(255, 80, 100, 255))
		drawHeart(dst, x+22, y, 1.5, rgba(70, 50, 60, 255))
	}
}

func (m *duelModeScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	m.hits.reset()
	t := float64(a.Frame)
	drawTextShadow(dst, "Дуэль вдвоём", 40, true, ScreenW/2, 18, Center, colText)
	drawText(dst, "Выбери, в какую дуэль играть", 20, false, ScreenW/2, 72, Center, colDim)
	for i := sim.DuelMode(0); i < sim.DuelModeCount; i++ {
		info := sim.DuelModes[i]
		x := ScreenW/2 + float64(int(i)-1)*400
		sel := i == m.sel
		bg := rgba(30, 26, 40, 230)
		if sel {
			bg = rgba(62, 48, 100, 245)
			glow(dst, x, 260, 220, rgba(120, 90, 255, 255), 0.15)
		}
		roundRect(dst, x-180, 120, 360, 280, 18, bg)
		if sel {
			roundRectStroke(dst, x-180, 120, 360, 280, 18, 3, colAccent)
		}
		drawModeIcon(dst, i, x, 200, t)
		c := colDim
		if sel {
			c = colText
		}
		drawText(dst, info.Name, 28, true, x, 268, Center, c)
		drawText(dst, info.Short, 15, false, x, 318, Center, c)
		drawText(dst, "матч до "+itoa(sim.DuelWinsNeeded)+" побед", 15, false, x, 350, Center, colGold)
		m.hits.add(rect{x - 185, 115, 370, 290}, hitModeCard+int(i))
	}
	for k, line := range sim.DuelModes[m.sel].Rules {
		drawText(dst, line, 18, false, ScreenW/2, 432+float64(k)*30, Center, colText)
	}
	drawText(dst, "Счёт матчей за сессию: "+itoa(a.duelWins[0])+" : "+itoa(a.duelWins[1]), 18, false, ScreenW/2, 568, Center, colGold)
	if a.Pointer.touchMode {
		drawText(dst, "Коснись выбранной карточки ещё раз, чтобы продолжить", 16, false, ScreenW/2, 610, Center, colDim)
		drawBackButton(dst, &m.hits, hitModeBack)
	} else {
		keyHint(dst, "←→ — вид дуэли   ·   Enter — дальше   ·   Esc — назад")
	}
}

// --- Выбор уровня для дуэли ---------------------------------------------------

type duelSetupScene struct {
	world sim.World
	level int // 1..8
	nav   nav
	hits  hits
}

const (
	hitDuelStart = 300 + iota
	hitDuelBack
	hitDuelDiff
	hitDuelWorld = 400 // +мир
	hitDuelLevel = 500 // +уровень
)

func newDuelSetupScene(a *App) *duelSetupScene {
	return &duelSetupScene{world: a.duelWorld, level: max(1, a.duelLevel)}
}

func (d *duelSetupScene) Update(a *App) {
	moved := false
	if d.nav.dir(navLeft) && d.level > 1 {
		d.level--
		moved = true
	}
	if d.nav.dir(navRight) && d.level < sim.LevelsPerWorld {
		d.level++
		moved = true
	}
	dw := worldSwitch()
	if d.nav.dir(navUp) {
		dw = -1
	}
	if d.nav.dir(navDown) {
		dw = 1
	}
	if dw != 0 {
		d.world = (d.world + sim.World(dw) + sim.WorldCount) % sim.WorldCount
		moved = true
	}
	if moved {
		a.Sound.Play(sfx.Menu)
	}
	start := okPressed()
	tap, tapped := d.hits.tapped(a)
	if tapped {
		switch {
		case tap == hitDuelStart:
			start = true
		case tap == hitDuelBack:
			a.Goto(newDuelModeScene(a))
			return
		case tap == hitDuelDiff:
			a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
			_ = a.Prof.Save()
		case tap >= hitDuelLevel:
			d.level = tap - hitDuelLevel
		case tap >= hitDuelWorld:
			d.world = sim.World(tap - hitDuelWorld)
		}
		a.Sound.Play(sfx.Menu)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.Prof.Diff = (a.Prof.Diff + 1) % sim.DifficultyCount
		_ = a.Prof.Save()
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newDuelModeScene(a))
		return
	}
	if start {
		a.Sound.Play(sfx.Select)
		a.duelWorld, a.duelLevel = d.world, d.level
		a.Goto(newDuelScene(a, sim.Track{World: d.world, Level: d.level, Diff: a.Prof.Diff}, a.duelMode))
	}
}

func (d *duelSetupScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	d.hits.reset()
	t := float64(a.Frame)
	drawTextShadow(dst, "Дуэль · "+sim.DuelModes[a.duelMode].Name, 40, true, ScreenW/2, 18, Center, colText)
	score := "Матч до " + itoa(sim.DuelWinsNeeded) + " побед, каждый раунд — следующий уровень   ·   Счёт матчей: " + itoa(a.duelWins[0]) + " : " + itoa(a.duelWins[1])
	drawText(dst, score, 18, false, ScreenW/2, 70, Center, colGold)

	// Миры: в дуэли открыты все.
	for i, w := range sim.Worlds {
		wi := sim.World(i)
		x := 140 + float64(i)*250
		sel := wi == d.world
		bg := rgba(30, 26, 40, 230)
		if sel {
			bg = lerpColor(rgba(60, 46, 96, 240), themes[wi].wall, 0.35)
		}
		roundRect(dst, x-115, 104, 230, 52, 12, bg)
		if sel {
			roundRectStroke(dst, x-115, 104, 230, 52, 12, 2, themes[wi].itemGlow)
		}
		drawItem(dst, wi, x-90, 130, 0.55, t, 0)
		c := colDim
		if sel {
			c = colText
		}
		drawText(dst, w.Name, 17, sel, x+10, 119, Center, c)
		d.hits.add(rect{x - 115, 100, 230, 60}, hitDuelWorld+i)
	}
	// Уровни.
	drawText(dst, "Уровень", 18, false, 248, 186, Right, colDim)
	for i := 1; i <= sim.LevelsPerWorld; i++ {
		x := 270 + float64(i-1)*64
		sel := i == d.level
		bg := rgba(32, 27, 42, 235)
		if sel {
			bg = rgba(62, 48, 100, 245)
		}
		roundRect(dst, x, 174, 54, 46, 10, bg)
		if sel {
			roundRectStroke(dst, x, 174, 54, 46, 10, 2, colAccent)
		}
		drawText(dst, itoa(i), 22, sel, x+27, 183, Center, colText)
		d.hits.add(rect{x - 4, 170, 62, 54}, hitDuelLevel+i)
	}
	diffHint := " (Tab)"
	if a.Pointer.touchMode {
		diffHint = ""
	}
	drawText(dst, "Сложность: "+sim.Difficulties[a.Prof.Diff].Name+diffHint, 18, false, 1030, 186, Center, colDim)
	d.hits.add(rect{900, 170, 260, 54}, hitDuelDiff)

	// Управление игроков.
	ab := sim.Abilities[sim.WorldRules[d.world].Ability]
	keys := func(l int, act controls.Action) string {
		var out []string
		for _, k := range layouts[l][act] {
			if k != "" {
				out = append(out, keyLabel(k))
			}
		}
		if len(out) == 0 {
			return "не назначено"
		}
		return strings.Join(out, " или ")
	}
	move := func(l int) string {
		return keyLabel(layouts[l].First(controls.Up)) + " " + keyLabel(layouts[l].First(controls.Left)) + " " +
			keyLabel(layouts[l].First(controls.Down)) + " " + keyLabel(layouts[l].First(controls.Right))
	}
	panels := [2][]string{
		{"Движение — " + move(controls.Duel1), "Рывок — " + keys(controls.Duel1, controls.Dash), "«" + ab.Name + "» — " + keys(controls.Duel1, controls.Ability), "или первый из двух геймпадов"},
		{"Движение — " + move(controls.Duel2), "Рывок — " + keys(controls.Duel2, controls.Dash), "«" + ab.Name + "» — " + keys(controls.Duel2, controls.Ability), "или геймпад (если он один)"},
	}
	if a.Pointer.touchMode {
		panels = [2][]string{
			{"Левая половина экрана:", "джойстик пальцем,", "кнопки «Рывок» и «" + ab.Name + "»", "у середины экрана"},
			{"Правая половина экрана:", "джойстик пальцем,", "кнопки «Рывок» и «" + ab.Name + "»", "у правого края"},
		}
	}
	for i := 0; i < 2; i++ {
		x := 100 + float64(i)*560
		drawPanel(dst, x, 244, 520, 190)
		roundRect(dst, x, 244, 520, 6, 3, duelColors[i])
		drawText(dst, duelNames[i], 24, true, x+24, 262, Left, duelColors[i])
		for k, line := range panels[i] {
			drawText(dst, line, 18, false, x+24, 304+float64(k)*30, Left, colText)
		}
	}
	for k, line := range sim.DuelModes[a.duelMode].Rules {
		drawText(dst, line, 17, false, ScreenW/2, 452+float64(k)*26, Center, colDim)
	}
	roundRect(dst, ScreenW/2-140, 572, 280, 56, 14, rgba(80, 60, 150, 250))
	glow(dst, ScreenW/2, 600, 200, rgba(120, 90, 255, 255), 0.2)
	drawText(dst, "В бой!", 28, true, ScreenW/2, 582, Center, colText)
	d.hits.add(rect{ScreenW/2 - 150, 566, 300, 68}, hitDuelStart)
	if a.Pointer.touchMode {
		drawBackButton(dst, &d.hits, hitDuelBack)
	} else {
		keyHint(dst, "↑↓ — мир   ·   ←→ — уровень   ·   Tab — сложность   ·   Enter — начать   ·   Esc — вид дуэли")
	}
}

// --- Сама дуэль ------------------------------------------------------------------

type duelScene struct {
	track     sim.Track // уровень, с которого начат матч
	mode      sim.DuelMode
	m         *sim.DuelMatch
	lv        *sim.Level
	d         *sim.Duel
	view      *fieldView
	fx        *fxSystem
	countdown int
	endT      int
	paused    bool
	pauseSel  int
	nav       nav
	hits      hits
	touch     [2]touchControls
	bots      int // 0 — нет, 1 — бот за второго игрока, 2 — оба бота (F9)
	banner    string
	bannerT   int
}

func newDuelScene(a *App, t sim.Track, mode sim.DuelMode) *duelScene {
	s := &duelScene{
		track: t, mode: mode, m: sim.NewDuelMatch(t, mode),
		touch: [2]touchControls{duelTouch(0), duelTouch(1)},
	}
	s.startRound()
	return s
}

// startRound готовит поле текущего раунда матча.
func (s *duelScene) startRound() {
	s.d = s.m.D
	s.lv = s.d.S.Lv
	s.view = newFieldView(s.lv)
	s.view.duel = s.d
	s.fx = newFX()
	s.countdown = countdownTicks
	s.endT = 0
	s.bannerT = 0
}

func (s *duelScene) Update(a *App) {
	s.fx.update()
	var touchIn [2]sim.Input
	for i := range s.touch {
		touchIn[i] = s.touch[i].update(a.Pointer)
	}
	if s.bannerT > 0 {
		s.bannerT--
	}
	if s.paused {
		s.updatePause(a)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		s.bots = (s.bots + 1) % 3
		a.Toast([...]string{"Боты выключены", "Игрок 2 — бот: можно сыграть одному", "Оба игрока — боты"}[s.bots])
	}
	if pausePressed() || s.touch[0].pauseTap {
		s.paused, s.pauseSel = true, 0
		return
	}
	if s.countdown > 0 {
		if s.countdown%sim.TPS == 0 {
			a.Sound.Play(sfx.Tick)
		}
		s.countdown--
		if s.countdown == 0 {
			a.Sound.Play(sfx.Go)
		}
		return
	}
	if s.d.Over {
		s.endT++
		if s.endT > 170 || (s.endT > 40 && (okPressed() || a.Pointer.tapped)) {
			if s.m.Over {
				a.Goto(newDuelResultScene(a, s))
			} else {
				s.m.NextRound()
				s.startRound()
			}
		}
		return
	}
	in := [2]sim.Input{readDuelInput(0) | touchIn[0], readDuelInput(1) | touchIn[1]}
	if s.bots >= 1 {
		in[1] = s.d.Autopilot(1)
	}
	if s.bots == 2 {
		in[0] = s.d.Autopilot(0)
	}
	s.d.Step(in)
	eventFX(a, s.fx, s.lv.Track.World, s.d.S.Events, true, func(t string) { s.banner, s.bannerT = t, 120 })
	if s.d.Over {
		if w := s.d.Winner; w >= 0 {
			a.Sound.Play(sfx.Win)
		}
		if s.m.EndRound() {
			a.duelWins[0] += b2i(s.m.Winner == 0)
			a.duelWins[1] += b2i(s.m.Winner == 1)
			if s.m.Winner >= 0 && s.bots == 0 {
				a.Grant(achieve.DuelWin)
			}
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

var duelPauseItems = []string{"Продолжить", "Начать матч заново", "Выбрать уровень", "Выйти в меню"}

func (s *duelScene) updatePause(a *App) {
	n := len(duelPauseItems)
	if s.nav.dir(navUp) {
		s.pauseSel = (s.pauseSel + n - 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if s.nav.dir(navDown) {
		s.pauseSel = (s.pauseSel + 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if pausePressed() {
		s.paused = false
		return
	}
	activate := okPressed()
	if id, ok := s.hits.tapped(a); ok {
		s.pauseSel, activate = id, true
	}
	if !activate {
		return
	}
	a.Sound.Play(sfx.Select)
	switch s.pauseSel {
	case 0:
		s.paused = false
	case 1:
		a.Goto(newDuelScene(a, s.track, s.mode))
	case 2:
		a.Goto(newDuelSetupScene(a))
	default:
		a.Goto(newMenuScene())
	}
}

// roundGoal — подсказка перед раундом.
func (s *duelScene) roundGoal() (string, string) {
	items := sim.Worlds[s.lv.Track.World].ItemsPl
	switch s.mode {
	case sim.DuelBank:
		return "Собирай " + items + " и неси их в портал — засчитывается только сданное", "Рывок в соперника вытряхивает его мешок"
	case sim.DuelSurvival:
		return "Останься с жизнями дольше соперника", "Рывок в соперника ранит, каждые 3 " + items + " лечат"
	}
	return "Собери больше " + sim.Worlds[s.lv.Track.World].ItemsGen + ", чем соперник", "Рывок в соперника выбивает у него предмет"
}

func (s *duelScene) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(themes[s.lv.Track.World].bg)
	s.view.drawWorld(dst, s.d.S, nil, a.Frame, s.fx, nil, "")
	s.drawHUD(a, dst)
	if a.Pointer.touchMode && !s.d.Over && !s.paused {
		ab := s.lv.Rules().Ability
		for i := range s.touch {
			s.touch[i].draw(dst, s.d.D[i].P.DashCD, s.d.D[i].P.AbilityCD, ab)
		}
	}
	if s.countdown > 0 {
		n := s.countdown/sim.TPS + 1
		k := float64(s.countdown%sim.TPS) / sim.TPS
		size := int(80 + 40*k)
		fillRect(dst, 0, ScreenH/2-150, ScreenW, 310, rgba(0, 0, 0, 110))
		drawTextShadow(dst, "Раунд "+itoa(s.m.Round)+"  ·  "+itoa(s.m.Wins[0])+" : "+itoa(s.m.Wins[1]), 30, true, ScreenW/2, ScreenH/2-140, Center, colGold)
		drawTextShadow(dst, itoa(n), size, true, ScreenW/2, ScreenH/2-float64(size)/2, Center, withAlpha(colText, 0.4+0.6*k))
		drawTextShadow(dst, sim.DuelModes[s.mode].Name+" · "+s.lv.Track.Title(), 22, false, ScreenW/2, ScreenH/2+70, Center, colText)
		goal, hint := s.roundGoal()
		drawTextShadow(dst, goal, 18, false, ScreenW/2, ScreenH/2+104, Center, colDim)
		drawTextShadow(dst, hint, 16, false, ScreenW/2, ScreenH/2+130, Center, colAccent)
	} else if st := s.d.S; st.Tick < 40 && !s.d.Over {
		drawTextShadow(dst, "В БОЙ!", 72, true, ScreenW/2, ScreenH/2-50, Center, withAlpha(colGold, 1-float64(st.Tick)/40))
	}
	if s.bannerT > 0 {
		al := math.Min(1, float64(s.bannerT)/20)
		drawTextShadow(dst, s.banner, 28, true, ScreenW/2, FieldY+20, Center, withAlpha(exitGlow, al))
	}
	if s.d.Over {
		s.drawRoundEnd(a, dst)
	}
	if s.paused {
		fillRect(dst, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 160))
		drawPanel(dst, ScreenW/2-200, 200, 400, 310)
		drawText(dst, "Пауза", 36, true, ScreenW/2, 220, Center, colText)
		s.hits.reset()
		for i, it := range duelPauseItems {
			y := 285 + float64(i)*52
			s.hits.add(rect{ScreenW/2 - 160, y, 320, 48}, i)
			c := colDim
			if i == s.pauseSel {
				c = colAccent
				roundRect(dst, ScreenW/2-160, y, 320, 44, 10, rgba(60, 50, 90, 255))
			}
			drawText(dst, it, 24, i == s.pauseSel, ScreenW/2, y+7, Center, c)
		}
	}
}

// drawRoundEnd — итог раунда поверх поля.
func (s *duelScene) drawRoundEnd(a *App, dst *ebiten.Image) {
	k := math.Min(1, float64(s.endT)/50)
	fillRect(dst, 0, 0, ScreenW, ScreenH, withAlpha(rgba(0, 0, 0, 255), 0.5*k))
	// Полоса под надписями, чтобы их не перебивали предметы и враги на поле.
	fillRect(dst, 0, ScreenH/2-130, ScreenW, 280, withAlpha(rgba(0, 0, 0, 255), 0.45*k))
	d := s.d
	txt, c := "Ничья в раунде", colText
	if d.Winner >= 0 {
		txt, c = "Раунд за "+duelNames[d.Winner]+"!", duelColors[d.Winner]
	}
	if s.m.Over {
		txt, c = "Матч: ничья!", colText
		if s.m.Winner >= 0 {
			txt, c = "Победил "+duelNames[s.m.Winner]+"!", duelColors[s.m.Winner]
		}
	}
	drawTextShadow(dst, txt, 60, true, ScreenW/2, ScreenH/2-110, Center, c)
	why := "Время вышло"
	switch {
	case s.mode == sim.DuelSurvival && (d.Out(0) || d.Out(1)):
		loser := 0
		if d.Out(1) && !d.Out(0) {
			loser = 1
		}
		why = duelNames[loser] + " остался без жизней"
		if d.Out(0) && d.Out(1) {
			why = "Оба остались без жизней"
		}
	case !d.TimedOut():
		why = "Все " + sim.Worlds[s.lv.Track.World].ItemsPl + " разобраны"
	}
	score := "Счёт раунда " + itoa(d.D[0].Pts) + " : " + itoa(d.D[1].Pts)
	if s.mode == sim.DuelSurvival {
		score = "Жизни " + itoa(survLives(d, 0)) + " : " + itoa(survLives(d, 1))
	}
	drawTextShadow(dst, why+"   ·   "+score, 22, false, ScreenW/2, ScreenH/2-24, Center, colText)
	// Победы в раундах кружками.
	for i := 0; i < 2; i++ {
		x := ScreenW/2 + float64(2*i-1)*130
		drawTextShadow(dst, duelNames[i], 22, true, x, ScreenH/2+24, Center, duelColors[i])
		drawWinPips(dst, x, ScreenH/2+78, s.m.Wins[i], duelColors[i], 14)
	}
	drawTextShadow(dst, ":", 40, true, ScreenW/2, ScreenH/2+40, Center, colDim)
	if s.endT > 40 {
		next := "Дальше — раунд " + itoa(s.m.Round+1)
		if s.m.Over {
			next = "Дальше — итоги матча"
		}
		drawTextShadow(dst, next, 18, false, ScreenW/2, ScreenH/2+120, Center, colDim)
	}
}

func survLives(d *sim.Duel, i int) int {
	if d.Out(i) {
		return 0
	}
	return d.D[i].Lives
}

// drawWinPips — кружки побед в раундах матча.
func drawWinPips(dst *ebiten.Image, cx, cy float64, wins int, c rgbaT, r float64) {
	n := sim.DuelWinsNeeded
	for k := 0; k < n; k++ {
		x := cx + (float64(k)-float64(n-1)/2)*r*2.6
		if k < wins {
			fillCircle(dst, x, cy, r, c)
			fillCircle(dst, x, cy, r*0.45, rgba(255, 255, 255, 200))
		} else {
			strokeCircle(dst, x, cy, r, 2, withAlpha(c, 0.6))
		}
	}
}

// drawCrown — корона лидера.
func drawCrown(dst *ebiten.Image, x, y, s float64) {
	fillPoly(dst, []float64{
		x - 10*s, y + 6*s, x - 10*s, y - 5*s, x - 5*s, y, x, y - 8*s,
		x + 5*s, y, x + 10*s, y - 5*s, x + 10*s, y + 6*s,
	}, colGold)
}

func (s *duelScene) drawHUD(a *App, dst *ebiten.Image) {
	st := s.d.S
	roundRect(dst, FieldX-3, 8, FieldW+6, 68, 12, rgba(26, 21, 30, 255))
	ab := s.lv.Rules().Ability
	abc := abilityColor(ab)
	lead := s.d.Leader()
	captions := [...]string{"на руках", "в банке", "предметов"}
	for i := 0; i < 2; i++ {
		p := &s.d.D[i]
		c := duelColors[i]
		// Левый игрок слева направо, правый — зеркально.
		side := func(x float64) float64 {
			if i == 0 {
				return x
			}
			return ScreenW - x
		}
		align := Left
		if i == 1 {
			align = Right
		}
		nw := textWidth(duelNames[i], 16, true)
		drawText(dst, duelNames[i], 16, true, side(FieldX+14), 12, align, c)
		drawWinPips(dst, side(FieldX+14+nw+22), 21, s.m.Wins[i], c, 6)
		if i == lead && !s.d.Over {
			drawCrown(dst, side(FieldX+14+nw+62), 21, 0.9)
		}
		hearts := s.lv.Lives
		if s.mode == sim.DuelSurvival {
			hearts = max(hearts, p.Lives)
		}
		for k := 0; k < hearts; k++ {
			hc := rgba(255, 80, 100, 255)
			if k >= p.Lives || p.Down > 0 {
				hc = rgba(70, 50, 60, 255)
			}
			drawHeart(dst, side(FieldX+26+float64(k)*26), 52, 0.85, hc)
		}
		drawText(dst, itoa(p.Pts), 32, true, side(235), 14, align, colText)
		drawText(dst, captions[s.mode], 12, false, side(235), 54, align, colDim)
		if s.mode == sim.DuelBank {
			drawItem(dst, s.lv.Track.World, side(345), 38, 0.55, float64(a.Frame), 0)
			drawText(dst, itoa(p.Carry), 24, true, side(365), 24, align, c)
			drawText(dst, "в мешке", 12, false, side(330), 54, align, colDim)
		}
		if !a.Pointer.touchMode {
			chip(dst, side(440), 42, "", "", p.P.DashCD, sim.DashCooldown, rgba(255, 190, 90, 255))
			chip(dst, side(495), 42, "", "", p.P.AbilityCD, sim.Abilities[ab].Cooldown, abc)
		}
	}
	// Посередине — время, раунд и сколько предметов осталось.
	left := st.TimeLeft()
	tc := colText
	if left < 10*sim.TPS && (a.Frame/15)%2 == 0 {
		tc = colRed
	}
	drawText(dst, fmtTime(left), 30, true, ScreenW/2, 12, Center, tc)
	msg := "раунд " + itoa(s.m.Round)
	if s.mode != sim.DuelSurvival {
		msg += " · на поле " + itoa(s.d.Left())
	}
	drawText(dst, msg, 15, false, ScreenW/2, 50, Center, colDim)
	if a.Pointer.touchMode {
		fillRect(dst, ScreenW/2-80, 22, 5, 16, withAlpha(colDim, 0.6))
		fillRect(dst, ScreenW/2-71, 22, 5, 16, withAlpha(colDim, 0.6))
	}
}

// --- Итоги дуэли -----------------------------------------------------------------

type duelResultScene struct {
	track sim.Track
	mode  sim.DuelMode
	m     *sim.DuelMatch
	sel   int
	nav   nav
	hits  hits
	t     int
}

var duelResultItems = []string{"Реванш", "Другой уровень", "Другая дуэль", "Выйти в меню"}

func newDuelResultScene(a *App, s *duelScene) *duelResultScene {
	return &duelResultScene{track: s.track, mode: s.mode, m: s.m}
}

func (r *duelResultScene) Update(a *App) {
	r.t++
	n := len(duelResultItems)
	if r.nav.dir(navUp) {
		r.sel = (r.sel + n - 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navDown) {
		r.sel = (r.sel + 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newDuelSetupScene(a))
		return
	}
	activate := okPressed()
	if id, ok := r.hits.tapped(a); ok {
		r.sel, activate = id, true
	}
	if !activate || r.t < 20 {
		return
	}
	a.Sound.Play(sfx.Select)
	switch r.sel {
	case 0:
		a.Goto(newDuelScene(a, r.track, r.mode))
	case 1:
		a.Goto(newDuelSetupScene(a))
	case 2:
		a.Goto(newDuelModeScene(a))
	default:
		a.Goto(newMenuScene())
	}
}

func (r *duelResultScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	m := r.m
	drawText(dst, "Дуэль «"+sim.DuelModes[r.mode].Name+"» · "+sim.Worlds[r.track.World].Name, 20, false, ScreenW/2, 20, Center, colDim)
	title, tc := "Ничья!", colText
	if m.Winner >= 0 {
		title, tc = "Победил "+duelNames[m.Winner]+"!", duelColors[m.Winner]
		glow(dst, ScreenW/2, 90, 320, tc, 0.25)
	}
	drawTextShadow(dst, title, 52, true, ScreenW/2, 50, Center, tc)
	drawText(dst, "Раунды  "+itoa(m.Wins[0])+" : "+itoa(m.Wins[1])+"   ·   Счёт матчей за сессию  "+itoa(a.duelWins[0])+" : "+itoa(a.duelWins[1]), 20, true, ScreenW/2, 118, Center, colGold)

	steal := "Выбил предметов"
	if r.mode == sim.DuelSurvival {
		steal = "Ранил рывком"
	}
	for i := 0; i < 2; i++ {
		x := 140 + float64(i)*520
		drawPanel(dst, x, 160, 480, 300)
		roundRect(dst, x, 160, 480, 6, 3, duelColors[i])
		drawText(dst, duelNames[i], 28, true, x+24, 178, Left, duelColors[i])
		if i == m.Winner {
			drawCrown(dst, x+textWidth(duelNames[i], 28, true)+50, 196, 1.4)
			drawText(dst, "победа", 18, true, x+456, 186, Right, colGold)
		}
		rows := [][2]string{
			{"Подобрано " + sim.Worlds[r.track.World].ItemsGen, itoa(m.Got[i])},
			{steal, itoa(m.Steals[i])},
			{"Выбит опасностями", timesRu(m.Knocks[i])},
		}
		for k, row := range rows {
			y := 232 + float64(k)*36
			drawText(dst, row[0], 20, false, x+24, y, Left, colDim)
			drawText(dst, row[1], 20, true, x+456, y, Right, colText)
		}
		drawText(dst, "Раунды", 22, false, x+24, 398, Left, colDim)
		drawWinPips(dst, x+150, 412, m.Wins[i], duelColors[i], 11)
		drawText(dst, itoa(m.Wins[i]), 36, true, x+456, 388, Right, colText)
	}
	// История раундов.
	hx := ScreenW/2 - float64(len(m.History)-1)*22
	for k, w := range m.History {
		c := colDim
		if w >= 0 {
			c = duelColors[w]
		}
		fillCircle(dst, hx+float64(k)*44, 480, 12, c)
		drawText(dst, itoa(k+1), 14, true, hx+float64(k)*44, 472, Center, rgba(20, 16, 26, 255))
	}

	r.hits.reset()
	for i, it := range duelResultItems {
		y := 506 + float64(i)*48
		r.hits.add(rect{ScreenW/2 - 160, y, 320, 44}, i)
		c := colDim
		if i == r.sel {
			c = colText
			roundRect(dst, ScreenW/2-160, y, 320, 42, 10, rgba(60, 46, 96, 240))
		}
		drawText(dst, it, 22, i == r.sel, ScreenW/2, y+7, Center, c)
	}
}

// timesRu — «ни разу», «1 раз», «2 раза», «5 раз».
func timesRu(n int) string {
	if n == 0 {
		return "ни разу"
	}
	w := "раз"
	if d := n % 10; (d >= 2 && d <= 4) && (n%100 < 12 || n%100 > 14) {
		w = "раза"
	}
	return itoa(n) + " " + w
}
