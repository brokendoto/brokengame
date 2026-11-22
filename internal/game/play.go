package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/achieve"
	"github.com/brokendoto/brokengame/internal/controls"
	"github.com/brokendoto/brokengame/internal/replay"
	"github.com/brokendoto/brokengame/internal/save"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

const countdownTicks = 3 * sim.TPS

// BotName — имя, под которым записываются забеги с автопилотом.
const BotName = "Бот"

// playScene — сам уровень.
type playScene struct {
	track sim.Track
	lv    *sim.Level
	st    *sim.State
	view  *fieldView
	fx    *fxSystem
	rec   []sim.Input

	ghost      *sim.State
	ghostIn    []sim.Input
	ghostEntry save.Entry
	ghostDone  int

	countdown int
	endT      int
	paused    bool
	pauseSel  int
	nav       nav
	trail     [][2]float64
	usedBot   bool
	autopilot bool // F9: бот ведёт шахтёра (для демонстрации алгоритма)
	touch     touchControls
	hits      hits
	banner    string
	bannerT   int

	// Счётчики для достижений.
	goldenGot, cursedGot int
	holeHit              bool
}

func newPlayScene(a *App, t sim.Track) *playScene {
	lv := sim.Generate(t)
	s := &playScene{
		track:     t,
		lv:        lv,
		st:        sim.NewState(lv),
		view:      newFieldView(lv),
		fx:        newFX(),
		countdown: countdownTicks,
	}
	if a.Prof.ShowGhost {
		if e, ok := a.Rec.Ghost(t.Key()); ok {
			if in, err := replay.Decode(e.Replay); err == nil {
				// Перед тем как показывать призрака, проверяем запись:
				// повтор обязан дать ровно тот же счёт, что в таблице.
				if r := sim.Run(lv, in); r.Won && r.Total == e.Score {
					s.ghost, s.ghostIn, s.ghostEntry = sim.NewState(lv), in, e
				}
			}
		}
	}
	return s
}

func (s *playScene) Update(a *App) {
	s.fx.update()
	touchIn := s.touch.update(a.Pointer)
	if s.bannerT > 0 {
		s.bannerT--
	}
	if s.paused {
		s.updatePause(a)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		s.autopilot = !s.autopilot
		if s.autopilot {
			a.Toast("Автопилот включён: результат запишется на имя «Бот»")
			a.Grant(achieve.Robot)
		}
	}
	if s.countdown > 0 {
		if s.countdown%sim.TPS == 0 {
			a.Sound.Play(sfx.Tick)
		}
		s.countdown--
		if s.countdown == 0 {
			a.Sound.Play(sfx.Go)
		}
		if pausePressed() || s.touch.pauseTap {
			s.paused = true
		}
		return
	}
	if s.st.Status != sim.Playing {
		s.endT++
		if s.endT > 80 || (s.endT > 20 && (okPressed() || a.Pointer.tapped)) {
			a.Goto(newResultScene(a, s))
		}
		if retryPressed() {
			a.Goto(newPlayScene(a, s.track))
		}
		return
	}
	if retryPressed() {
		a.Goto(newPlayScene(a, s.track))
		return
	}
	if pausePressed() || s.touch.pauseTap {
		s.paused, s.pauseSel = true, 0
		return
	}

	in := readPlayInput() | touchIn
	if s.autopilot {
		in = sim.Autopilot(s.st)
		s.usedBot = true
	}
	s.rec = append(s.rec, in)
	s.st.Step(in)
	if s.ghost != nil && s.ghost.Status == sim.Playing {
		gi := sim.Input(0)
		if s.st.Tick-1 < len(s.ghostIn) {
			gi = s.ghostIn[s.st.Tick-1]
		}
		s.ghost.Step(gi)
		if s.ghost.Status == sim.Won {
			s.ghostDone = s.ghost.FinishTick
		}
	}
	if s.st.P.DashT > 0 {
		s.trail = append(s.trail, [2]float64{s.st.P.X, s.st.P.Y})
		if len(s.trail) > 6 {
			s.trail = s.trail[1:]
		}
	} else if len(s.trail) > 0 {
		s.trail = s.trail[1:]
	}
	s.handleEvents(a)
}

func (s *playScene) handleEvents(a *App) {
	for _, e := range s.st.Events {
		switch e.Kind {
		case sim.EvGolden:
			s.goldenGot++
		case sim.EvCursed:
			s.cursedGot++
		case sim.EvHole:
			s.holeHit = true
			if !s.usedBot {
				a.Grant(achieve.EventHoriz)
			}
		}
	}
	eventFX(a, s.fx, s.track.World, s.st.Events, false, func(t string) { s.banner, s.bannerT = t, 120 })
}

// eventFX превращает события симуляции в звук, частицы и надписи.
// duel — подписи очков цветом игрока и события дуэли.
func eventFX(a *App, fx *fxSystem, w sim.World, events []sim.Event, duel bool, banner func(string)) {
	for _, e := range events {
		x, y := px(e.X), py(e.Y)
		switch e.Kind {
		case sim.EvPickup:
			a.Sound.PlayPickup(e.N)
			fx.burst(x, y, 16, themes[w].itemGlow, 5, 3, true)
			pts := int(100 * (1 + 0.25*float64(min(e.N, 9)-1)))
			c := colText
			if e.N >= 3 {
				c = colGold
			}
			if duel {
				c = duelColors[e.P]
			}
			fx.text("+"+itoa(pts), x, y-30, c, 20)
		case sim.EvExitOpen:
			a.Sound.Play(sfx.ExitOpen)
			fx.burst(x, y, 40, exitGlow, 7, 4, true)
			banner("Все " + sim.Worlds[w].ItemsPl + " собраны — к выходу!")
		case sim.EvHit:
			a.Sound.Play(sfx.Hit)
			fx.shake, fx.flash = 14, 1
			fx.burst(x, y, 20, colRed, 6, 4, false)
		case sim.EvBlock:
			a.Sound.Play(sfx.Block)
			fx.burst(x, y, 14, colAccent, 6, 3, true)
		case sim.EvDash:
			a.Sound.Play(sfx.Dash)
			fx.debris(x, y+10, 6, themes[w].speck)
		case sim.EvAbility:
			ab := sim.Ability(e.N)
			c := abilityColor(ab)
			switch ab {
			case sim.AbBlink:
				a.Sound.Play(sfx.Dash)
				fx.burst(x, y, 24, c, 5, 3, true)
			case sim.AbPulse:
				a.Sound.Play(sfx.Crash)
				fx.shake = math.Max(fx.shake, 5)
				for i := 0; i < 3; i++ {
					fx.burst(x, y, 24, c, 8+float64(i)*3, 3, true)
				}
			default:
				a.Sound.Play(sfx.Shield)
				fx.burst(x, y, 20, c, 4, 3, true)
			}
		case sim.EvBlinkOut:
			fx.burst(x, y, 14, themes[w].eyeSpecial, 4, 3, true)
		case sim.EvBlinkIn:
			if e.N == 1 {
				a.Sound.Play(sfx.Warn)
			} else {
				fx.burst(x, y, 14, themes[w].eyeSpecial, 4, 3, true)
			}
		case sim.EvCursed:
			a.Sound.Play(sfx.Hit)
			fx.burst(x, y, 18, rgba(255, 60, 140, 255), 5, 3, true)
			fx.text("−5 с", x, y-30, colRed, 24)
			fx.shake = math.Max(fx.shake, 5)
		case sim.EvGolden:
			a.Sound.PlayPickup(e.N + 2)
			fx.burst(x, y, 24, colGold, 6, 3, true)
			pts := int(sim.GoldenScore * (1 + 0.25*float64(min(e.N, 9)-1)))
			fx.text("+"+itoa(pts), x, y-30, colGold, 24)
		case sim.EvVanish:
			fx.burst(x, y, 10, withAlpha(colGold, 0.6), 3, 2, true)
		case sim.EvHole:
			fx.burst(x, y, 30, rgba(170, 90, 255, 255), 6, 4, true)
		case sim.EvRockWarn, sim.EvCaveWarn:
			a.Sound.Play(sfx.Warn)
		case sim.EvRockBreak:
			a.Sound.Play(sfx.Crash)
			fx.debris(x, y, 14, rgba(138, 128, 120, 255))
			fx.shake = math.Max(fx.shake, 4)
		case sim.EvCaveFall:
			a.Sound.Play(sfx.Crash)
			fx.shake = math.Max(fx.shake, 9)
		case sim.EvHeal:
			a.Sound.Play(sfx.Heal)
			fx.burst(x, y, 16, rgba(255, 110, 130, 255), 4, 3, true)
		case sim.EvWin:
			a.Sound.Play(sfx.Win)
			fx.burst(x, y, 60, exitGlow, 9, 4, true)
		case sim.EvLose:
			a.Sound.Play(sfx.Lose)
		case sim.EvKnockout:
			a.Sound.Play(sfx.Lose)
			fx.burst(x, y, 40, duelColors[e.N], 7, 4, true)
			banner("Игрок " + itoa(e.N+1) + " выбит! −" + itoa(sim.DuelKnockPenalty))
		case sim.EvRespawn:
			a.Sound.Play(sfx.Heal)
			fx.burst(x, y, 20, duelColors[e.N], 4, 3, true)
		case sim.EvStun:
			a.Sound.Play(sfx.Block)
			fx.shake = math.Max(fx.shake, 6)
			fx.burst(x, y, 18, rgba(255, 240, 120, 255), 5, 3, true)
			fx.text("Оглушён!", x, y-34, duelColors[e.N], 20)
		}
	}
}

var pauseItems = []string{"Продолжить", "Начать заново", "Выйти в меню"}

func (s *playScene) updatePause(a *App) {
	if s.nav.dir(navUp) {
		s.pauseSel = (s.pauseSel + len(pauseItems) - 1) % len(pauseItems)
		a.Sound.Play(sfx.Menu)
	}
	if s.nav.dir(navDown) {
		s.pauseSel = (s.pauseSel + 1) % len(pauseItems)
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
	if activate {
		a.Sound.Play(sfx.Select)
		switch s.pauseSel {
		case 0:
			s.paused = false
		case 1:
			a.Goto(newPlayScene(a, s.track))
		case 2:
			a.Goto(newMenuScene())
		}
	}
}

func (s *playScene) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(themes[s.track.World].bg)
	var ghost *sim.State
	if s.ghost != nil && s.ghost.Status == sim.Playing {
		ghost = s.ghost
	}
	s.view.drawWorld(dst, s.st, ghost, a.Frame, s.fx, s.trail, s.ghostEntry.Name)
	s.drawHUD(a, dst)
	if a.Pointer.touchMode && s.st.Status == sim.Playing && !s.paused {
		s.touch.draw(dst, s.st.P.DashCD, s.st.P.AbilityCD, s.lv.Rules().Ability)
		// Значок паузы: нажатие на верхнюю панель ставит игру на паузу.
		fillRect(dst, 580, 18, 6, 20, withAlpha(colDim, 0.7))
		fillRect(dst, 591, 18, 6, 20, withAlpha(colDim, 0.7))
	}

	if s.countdown > 0 {
		n := s.countdown/sim.TPS + 1
		k := float64(s.countdown%sim.TPS) / sim.TPS
		size := int(80 + 40*k)
		drawTextShadow(dst, itoa(n), size, true, ScreenW/2, ScreenH/2-float64(size)/2, Center, withAlpha(colText, 0.4+0.6*k))
		drawTextShadow(dst, s.track.Title(), 22, false, ScreenW/2, ScreenH/2+70, Center, colText)
		w := s.track.World
		goal := "Собери все " + sim.Worlds[w].ItemsPl + " (" + itoa(len(s.lv.Items)) + ") и доберись до выхода"
		drawTextShadow(dst, goal, 18, false, ScreenW/2, ScreenH/2+104, Center, colDim)
		ab := sim.Abilities[s.lv.Rules().Ability]
		key := hintKey(controls.Solo, controls.Ability)
		if a.Pointer.touchMode {
			key = "кнопка"
		}
		drawTextShadow(dst, sim.Worlds[w].Hero+": «"+ab.Name+"» ("+key+") — "+ab.Hint, 16, false, ScreenW/2, ScreenH/2+130, Center, abilityColor(s.lv.Rules().Ability))
		if s.ghost != nil {
			drawTextShadow(dst, "Соперник-призрак: "+s.ghostEntry.Name+" · "+fmtScore(s.ghostEntry.Score)+" очков", 18, false, ScreenW/2, ScreenH/2+160, Center, rgba(170, 210, 255, 255))
		}
	} else if s.st.Tick < 40 {
		drawTextShadow(dst, "ВПЕРЁД!", 72, true, ScreenW/2, ScreenH/2-50, Center, withAlpha(colGold, 1-float64(s.st.Tick)/40))
	}
	if s.bannerT > 0 {
		al := math.Min(1, float64(s.bannerT)/20)
		drawTextShadow(dst, s.banner, 28, true, ScreenW/2, FieldY+20, Center, withAlpha(exitGlow, al))
	}
	if s.st.Status == sim.Won {
		drawTextShadow(dst, "ВЫХОД!", 80, true, ScreenW/2, ScreenH/2-60, Center, colGold)
	} else if s.st.Status == sim.Lost {
		fillRect(dst, 0, 0, ScreenW, ScreenH, withAlpha(rgba(30, 0, 0, 255), math.Min(0.5, float64(s.endT)/60)))
		drawTextShadow(dst, s.st.LoseReason, 54, true, ScreenW/2, ScreenH/2-40, Center, colRed)
	}
	if s.paused {
		fillRect(dst, 0, 0, ScreenW, ScreenH, rgba(0, 0, 0, 160))
		drawPanel(dst, ScreenW/2-200, 220, 400, 260)
		drawText(dst, "Пауза", 36, true, ScreenW/2, 240, Center, colText)
		s.hits.reset()
		for i, it := range pauseItems {
			s.hits.add(rect{ScreenW/2 - 160, 305 + float64(i)*52, 320, 48}, i)
			c := colDim
			if i == s.pauseSel {
				c = colAccent
				roundRect(dst, ScreenW/2-160, 305+float64(i)*52, 320, 44, 10, rgba(60, 50, 90, 255))
			}
			drawText(dst, it, 24, i == s.pauseSel, ScreenW/2, 312+float64(i)*52, Center, c)
		}
	}
}

func (s *playScene) drawHUD(a *App, dst *ebiten.Image) {
	st := s.st
	roundRect(dst, FieldX-3, 8, FieldW+6, 68, 12, rgba(26, 21, 30, 255))

	// Слева: уровень и жизни.
	title := themes[s.track.World].short + " · ур. " + itoa(s.track.Level)
	if s.track.Daily != "" {
		title = "День · " + themes[s.track.World].short
	}
	drawText(dst, title, 16, false, FieldX+14, 14, Left, colDim)
	for i := 0; i < s.lv.Lives; i++ {
		c := rgba(255, 80, 100, 255)
		if i >= st.Lives {
			c = rgba(70, 50, 60, 255)
		}
		drawHeart(dst, FieldX+26+float64(i)*30, 52, 1, c)
	}

	// Кристаллы.
	x := FieldX + 190.0
	drawItem(dst, s.track.World, x, 42, 0.7, float64(a.Frame), 0)
	cc := colText
	if st.ExitOpen {
		cc = exitGlow
	}
	drawText(dst, itoa(st.Collected)+" / "+itoa(len(st.Items)), 30, true, x+24, 24, Left, cc)

	// Время.
	left := st.TimeLeft()
	tc := colText
	if left < 10*sim.TPS && (a.Frame/15)%2 == 0 {
		tc = colRed
	}
	drawText(dst, "время", 14, false, 470, 14, Left, colDim)
	drawText(dst, fmtTime(left), 32, true, 470, 30, Left, tc)
	gold := s.lv.GoldTicks() - st.Tick
	if gold > 0 {
		drawText(dst, "на золото", 14, false, 606, 14, Left, colDim)
		drawMedal(dst, 616, 56, 6, sim.Gold)
		drawText(dst, fmtTime(gold), 16, false, 628, 47, Left, colGold)
	}

	// Серия.
	cx := 720.0
	drawText(dst, "серия", 14, false, cx, 14, Left, colDim)
	mul := st.Multiplier()
	mc := colText
	if st.Combo >= 3 {
		mc = colGold
	}
	drawText(dst, "×"+fmtMul(mul), 30, true, cx, 30, Left, mc)
	if st.ComboTimer > 0 {
		w := 70 * float64(st.ComboTimer) / sim.ComboWindow
		roundRect(dst, cx+78, 46, 70, 8, 4, rgba(60, 50, 70, 255))
		roundRect(dst, cx+78, 46, w, 8, 4, mc)
	}

	// Очки за кристаллы.
	drawText(dst, "очки", 14, false, 900, 14, Left, colDim)
	drawText(dst, fmtScore(st.ItemScore), 30, true, 900, 30, Left, colText)

	// Способности.
	dashKey, abKey := hintKey(controls.Solo, controls.Dash), hintKey(controls.Solo, controls.Ability)
	if a.Pointer.touchMode {
		dashKey, abKey = "", ""
	}
	ab := s.lv.Rules().Ability
	chip(dst, 1040, 42, "Рывок", dashKey, st.P.DashCD, sim.DashCooldown, rgba(255, 190, 90, 255))
	chip(dst, 1150, 42, sim.Abilities[ab].Name, abKey, st.P.AbilityCD, sim.Abilities[ab].Cooldown, abilityColor(ab))

	// Призрак.
	if s.ghost != nil {
		var txt string
		if s.ghostDone > 0 {
			txt = "Призрак " + s.ghostEntry.Name + " финишировал: " + fmtTime(s.ghostDone)
		} else {
			txt = "Призрак " + s.ghostEntry.Name + ": " + itoa(s.ghost.Collected) + "/" + itoa(len(s.ghost.Items))
			diff := st.Collected - s.ghost.Collected
			if diff > 0 {
				txt += "  ·  ты впереди на " + itoa(diff)
			} else if diff < 0 {
				txt += "  ·  отстаёшь на " + itoa(-diff)
			}
		}
		drawTextShadow(dst, txt, 15, false, FieldX+FieldW-10, FieldY+FieldH-26, Right, rgba(170, 210, 255, 230))
	}
}

func fmtMul(m float64) string {
	t := int(math.Round(m * 100))
	return itoa(t/100) + "," + pad2(t%100)
}

// chip — значок способности с перезарядкой.
func chip(dst *ebiten.Image, x, y float64, name, key string, cd, max int, c rgbaT) {
	r := 22.0
	fillCircle(dst, x, y, r, rgba(40, 34, 48, 255))
	if cd > 0 {
		k := 1 - float64(cd)/float64(max)
		var pts []float64
		pts = append(pts, x, y)
		for i := 0; i <= 24; i++ {
			ang := -math.Pi/2 + k*2*math.Pi*float64(i)/24
			pts = append(pts, x+math.Cos(ang)*r, y+math.Sin(ang)*r)
		}
		fillPoly(dst, pts, withAlpha(c, 0.35))
		drawText(dst, itoa(cd/sim.TPS+1), 18, true, x, y-11, Center, colDim)
	} else {
		fillCircle(dst, x, y, r, withAlpha(c, 0.85))
		size := 14
		for size > 9 && textWidth(key, size, true) > 2*r-6 {
			size--
		}
		drawText(dst, key, size, true, x, y-float64(size)*0.62, Center, rgba(20, 16, 24, 255))
	}
	drawText(dst, name, 13, false, x+r+6, y-8, Left, colDim)
}
