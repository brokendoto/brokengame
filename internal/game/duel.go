package game

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/controls"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// Дуэль вдвоём на одном экране (или одном планшете). Правила — в sim/duel.go.

var duelNames = [2]string{"Игрок 1", "Игрок 2"}

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
			a.Goto(newMenuScene())
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
		a.Goto(newMenuScene())
		return
	}
	if start {
		a.Sound.Play(sfx.Select)
		a.duelWorld, a.duelLevel = d.world, d.level
		a.Goto(newDuelScene(a, sim.Track{World: d.world, Level: d.level, Diff: a.Prof.Diff}))
	}
}

func (d *duelSetupScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	d.hits.reset()
	t := float64(a.Frame)
	drawTextShadow(dst, "Дуэль вдвоём", 40, true, ScreenW/2, 18, Center, colText)
	score := "Счёт матчей: " + itoa(a.duelWins[0]) + " : " + itoa(a.duelWins[1])
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
	rules := []string{
		"Предметы общие: кто первым взял, тому и очки. Серии работают как обычно.",
		"Рывок в соперника оглушает его. «Стоп-время» и «Импульс» тоже бьют по сопернику.",
		"Выбитый игрок теряет " + itoa(sim.DuelKnockPenalty) + " очков и через 3 с возвращается на старт.",
		"Кто первым выйдет через открытый выход, получит +" + itoa(sim.DuelExitBonus) + ". Побеждает тот, у кого больше очков.",
	}
	for k, line := range rules {
		drawText(dst, line, 17, false, ScreenW/2, 452+float64(k)*26, Center, colDim)
	}
	roundRect(dst, ScreenW/2-140, 572, 280, 56, 14, rgba(80, 60, 150, 250))
	glow(dst, ScreenW/2, 600, 200, rgba(120, 90, 255, 255), 0.2)
	drawText(dst, "В бой!", 28, true, ScreenW/2, 582, Center, colText)
	d.hits.add(rect{ScreenW/2 - 150, 566, 300, 68}, hitDuelStart)
	if a.Pointer.touchMode {
		drawBackButton(dst, &d.hits, hitDuelBack)
	} else {
		keyHint(dst, "↑↓ — мир   ·   ←→ — уровень   ·   Tab — сложность   ·   Enter — начать   ·   Esc — назад")
	}
}

// --- Сама дуэль ------------------------------------------------------------------

type duelScene struct {
	track     sim.Track
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

func newDuelScene(a *App, t sim.Track) *duelScene {
	lv := sim.Generate(t)
	d := sim.NewDuel(lv)
	v := newFieldView(lv)
	v.duel = d
	return &duelScene{
		track: t, lv: lv, d: d, view: v, fx: newFX(),
		countdown: countdownTicks,
		touch:     [2]touchControls{duelTouch(0), duelTouch(1)},
	}
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
		if s.endT > 100 || (s.endT > 30 && (okPressed() || a.Pointer.tapped)) {
			a.Goto(newDuelResultScene(a, s))
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
	eventFX(a, s.fx, s.track.World, s.d.S.Events, true, func(t string) { s.banner, s.bannerT = t, 120 })
	if s.d.Over {
		a.duelWins[0] += b2i(s.d.Winner == 0)
		a.duelWins[1] += b2i(s.d.Winner == 1)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

var duelPauseItems = []string{"Продолжить", "Начать заново", "Выбрать уровень", "Выйти в меню"}

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
		a.Goto(newDuelScene(a, s.track))
	case 2:
		a.Goto(newDuelSetupScene(a))
	default:
		a.Goto(newMenuScene())
	}
}

func (s *duelScene) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(themes[s.track.World].bg)
	s.view.drawWorld(dst, s.d.S, nil, a.Frame, s.fx, nil, "")
	s.drawHUD(a, dst)
	st := s.d.S
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
		drawTextShadow(dst, itoa(n), size, true, ScreenW/2, ScreenH/2-float64(size)/2, Center, withAlpha(colText, 0.4+0.6*k))
		drawTextShadow(dst, "Дуэль · "+s.track.Title(), 22, false, ScreenW/2, ScreenH/2+70, Center, colText)
		drawTextShadow(dst, "Собирай "+sim.Worlds[s.track.World].ItemsPl+" быстрее соперника и первым беги к выходу", 18, false, ScreenW/2, ScreenH/2+104, Center, colDim)
		drawTextShadow(dst, "Рывок в соперника его оглушает", 16, false, ScreenW/2, ScreenH/2+130, Center, colAccent)
	} else if st.Tick < 40 {
		drawTextShadow(dst, "В БОЙ!", 72, true, ScreenW/2, ScreenH/2-50, Center, withAlpha(colGold, 1-float64(st.Tick)/40))
	}
	if s.bannerT > 0 {
		al := math.Min(1, float64(s.bannerT)/20)
		drawTextShadow(dst, s.banner, 28, true, ScreenW/2, FieldY+20, Center, withAlpha(exitGlow, al))
	}
	if s.d.Over {
		fillRect(dst, 0, 0, ScreenW, ScreenH, withAlpha(rgba(0, 0, 0, 255), math.Min(0.45, float64(s.endT)/60)))
		txt, c := "Ничья!", colText
		if s.d.Winner >= 0 {
			txt, c = "Победил "+duelNames[s.d.Winner]+"!", duelColors[s.d.Winner]
		}
		drawTextShadow(dst, txt, 64, true, ScreenW/2, ScreenH/2-60, Center, c)
		sub := "Время вышло"
		if s.d.Exited >= 0 {
			sub = duelNames[s.d.Exited] + " первым добрался до выхода"
		}
		drawTextShadow(dst, sub, 22, false, ScreenW/2, ScreenH/2+20, Center, colText)
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

func (s *duelScene) drawHUD(a *App, dst *ebiten.Image) {
	st := s.d.S
	roundRect(dst, FieldX-3, 8, FieldW+6, 68, 12, rgba(26, 21, 30, 255))
	ab := s.lv.Rules().Ability
	abc := abilityColor(ab)
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
		drawText(dst, duelNames[i], 16, true, side(FieldX+14), 12, align, c)
		for k := 0; k < s.lv.Lives; k++ {
			hc := rgba(255, 80, 100, 255)
			if k >= p.Lives || p.Down > 0 {
				hc = rgba(70, 50, 60, 255)
			}
			drawHeart(dst, side(FieldX+26+float64(k)*28), 52, 0.9, hc)
		}
		drawText(dst, fmtScore(p.ItemScore), 32, true, side(215), 24, align, colText)
		drawItem(dst, s.track.World, side(345), 42, 0.55, float64(a.Frame), 0)
		drawText(dst, itoa(p.Got), 24, true, side(365), 28, align, c)
		if !a.Pointer.touchMode {
			chip(dst, side(440), 42, "", "", p.P.DashCD, sim.DashCooldown, rgba(255, 190, 90, 255))
			chip(dst, side(495), 42, "", "", p.P.AbilityCD, sim.Abilities[ab].Cooldown, abc)
		}
	}
	// Посередине — время и сколько предметов осталось.
	left := st.TimeLeft()
	tc := colText
	if left < 10*sim.TPS && (a.Frame/15)%2 == 0 {
		tc = colRed
	}
	drawText(dst, fmtTime(left), 30, true, ScreenW/2, 14, Center, tc)
	rest := len(st.Items) - st.Collected
	msg := "осталось " + itoa(rest)
	mc := colDim
	if st.ExitOpen {
		msg, mc = "выход открыт!", exitGlow
	}
	drawText(dst, msg, 15, false, ScreenW/2, 50, Center, mc)
	if a.Pointer.touchMode {
		fillRect(dst, ScreenW/2-80, 22, 5, 16, withAlpha(colDim, 0.6))
		fillRect(dst, ScreenW/2-71, 22, 5, 16, withAlpha(colDim, 0.6))
	}
}

// --- Итоги дуэли -----------------------------------------------------------------

type duelResultScene struct {
	track sim.Track
	d     *sim.Duel
	sel   int
	nav   nav
	hits  hits
	t     int
}

var duelResultItems = []string{"Реванш", "Другой уровень", "Выйти в меню"}

func newDuelResultScene(a *App, s *duelScene) *duelResultScene {
	if s.d.Winner >= 0 {
		a.Sound.Play(sfx.Win)
	}
	return &duelResultScene{track: s.track, d: s.d}
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
		a.Goto(newDuelScene(a, r.track))
	case 1:
		a.Goto(newDuelSetupScene(a))
	default:
		a.Goto(newMenuScene())
	}
}

func (r *duelResultScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawText(dst, "Дуэль · "+r.track.Title(), 20, false, ScreenW/2, 24, Center, colDim)
	title, tc := "Ничья!", colText
	if r.d.Winner >= 0 {
		title, tc = "Победил "+duelNames[r.d.Winner]+"!", duelColors[r.d.Winner]
		glow(dst, ScreenW/2, 100, 320, tc, 0.25)
	}
	drawTextShadow(dst, title, 52, true, ScreenW/2, 60, Center, tc)
	drawText(dst, "Счёт матчей  "+itoa(a.duelWins[0])+" : "+itoa(a.duelWins[1]), 22, true, ScreenW/2, 130, Center, colGold)

	for i := 0; i < 2; i++ {
		p := &r.d.D[i]
		x := 140 + float64(i)*520
		drawPanel(dst, x, 180, 480, 300)
		roundRect(dst, x, 180, 480, 6, 3, duelColors[i])
		drawText(dst, duelNames[i], 28, true, x+24, 198, Left, duelColors[i])
		if i == r.d.Winner {
			drawItem(dst, sim.Space, x+textWidth(duelNames[i], 28, true)+46, 216, 0.8, float64(a.Frame), 0)
			drawText(dst, "победа", 18, true, x+456, 206, Right, colGold)
		}
		rows := [][2]string{
			{"Собрано " + sim.Worlds[r.track.World].ItemsGen, itoa(p.Got) + " из " + itoa(len(r.d.S.Items))},
			{"Лучшая серия", "×" + itoa(p.BestCombo)},
			{"Выбит", timesRu(p.Knocks)},
		}
		if r.d.Exited == i {
			rows = append(rows, [2]string{"Первым у выхода", "+" + itoa(sim.DuelExitBonus)})
		}
		for k, row := range rows {
			y := 250 + float64(k)*36
			drawText(dst, row[0], 20, false, x+24, y, Left, colDim)
			drawText(dst, row[1], 20, true, x+456, y, Right, colText)
		}
		drawText(dst, "Очки", 22, false, x+24, 420, Left, colDim)
		drawText(dst, fmtScore(p.ItemScore), 36, true, x+456, 410, Right, colText)
	}

	r.hits.reset()
	for i, it := range duelResultItems {
		y := 510 + float64(i)*52
		r.hits.add(rect{ScreenW/2 - 160, y, 320, 48}, i)
		c := colDim
		if i == r.sel {
			c = colText
			roundRect(dst, ScreenW/2-160, y, 320, 44, 10, rgba(60, 46, 96, 240))
		}
		drawText(dst, it, 24, i == r.sel, ScreenW/2, y+7, Center, c)
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
