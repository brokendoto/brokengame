package game

import (
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/achieve"
	"github.com/brokendoto/brokengame/internal/replay"
	"github.com/brokendoto/brokengame/internal/save"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// marathonRun — забег марафона: этапы идут подряд, жизни переносятся,
// очки складываются.
type marathonRun struct {
	week    string
	stage   int // текущий этап, с 1
	lives   int
	score   int
	ticks   int
	done    int      // пройдено этапов
	recs    []string // записи этапов (для проверки результата)
	usedBot bool
}

// replaySep разделяет записи этапов: в base64 (URL) такого символа нет.
const replaySep = "~"

func newMarathonRun() *marathonRun {
	return &marathonRun{week: sim.MarathonWeek(time.Now()), stage: 1, lives: sim.MarathonLives}
}

func newMarathonPlay(a *App, m *marathonRun) *playScene {
	lv := sim.MarathonLevel(m.week, m.stage)
	s := &playScene{
		track:     lv.Track,
		lv:        lv,
		st:        sim.NewMarathonState(lv, m.lives),
		view:      newFieldView(lv),
		fx:        newFX(),
		countdown: countdownTicks,
		mara:      m,
		autopilot: m.usedBot,
	}
	if m.stage > 1 {
		s.countdown = 2 * sim.TPS
	}
	return s
}

// marathonScene — пауза между этапами или итог марафона.
type marathonScene struct {
	m        *marathonRun
	over     bool
	stageRes sim.Result
	reason   string
	healed   bool
	name     string
	rank     int
	prevBest save.Entry
	hadPrev  bool
	top      []save.Entry
	t        int
	sel      int
	nav      nav
	items    []string
	hits     hits
}

func newMarathonScene(a *App, p *playScene) *marathonScene {
	m := p.mara
	m.usedBot = m.usedBot || p.usedBot
	m.recs = append(m.recs, replay.Encode(p.rec))
	r := &marathonScene{m: m, stageRes: p.st.Result(), reason: p.st.LoseReason}
	ghost := 0
	a.Grant(achieve.CheckRun(achieve.Run{
		Track: p.track, Result: r.stageRes, ParTicks: p.lv.ParTicks,
		GoldenTotal: len(p.lv.Golden), GoldenTaken: p.goldenGot,
		CursedTotal: len(p.lv.Cursed), CursedTaken: p.cursedGot,
		HoleHit: p.holeHit, GhostScore: ghost, UsedBot: m.usedBot,
		Medals: a.Prof.Medals,
	})...)
	if r.stageRes.Won {
		m.done++
		m.score += r.stageRes.Total
		m.ticks += r.stageRes.Ticks
		next := sim.MarathonCarry(p.st.Lives, m.done)
		r.healed = next > p.st.Lives
		m.lives = next
		if !m.usedBot {
			a.Grant(achieve.CheckMarathon(m.done)...)
		}
		r.items = []string{"Следующий этап", "Завершить забег"}
		return r
	}
	r.finish(a)
	return r
}

// finish подводит итог марафона и записывает результат в таблицу недели.
func (r *marathonScene) finish(a *App) {
	m := r.m
	r.over = true
	key := sim.MarathonTrack(m.week, 1).Key()
	r.name = a.Prof.Name
	if m.usedBot {
		r.name = BotName
	}
	r.prevBest, r.hadPrev = a.Rec.PersonalBest(key, r.name)
	if m.done > 0 {
		r.rank = a.Records.Submit(key, save.Entry{
			Name:   r.name,
			Score:  m.score,
			Ticks:  m.ticks,
			Stages: m.done,
			Replay: strings.Join(m.recs, replaySep),
		})
	}
	r.top = a.Records.Top(key)
	r.items = []string{"Новый забег", "Выйти в меню"}
	r.sel, r.t = 0, 0
}

func (r *marathonScene) Update(a *App) {
	r.t++
	if r.nav.dir(navUp) {
		r.sel = (r.sel + len(r.items) - 1) % len(r.items)
		a.Sound.Play(sfx.Menu)
	}
	if r.nav.dir(navDown) {
		r.sel = (r.sel + 1) % len(r.items)
		a.Sound.Play(sfx.Menu)
	}
	activate := okPressed()
	if id, ok := r.hits.tapped(a); ok {
		r.sel, activate = id, true
	}
	if backPressed() {
		if r.over {
			a.Goto(newMenuScene())
			return
		}
		r.sel, activate = 1, true
	}
	if !activate || r.t < 20 {
		return
	}
	a.Sound.Play(sfx.Select)
	switch r.items[r.sel] {
	case "Следующий этап":
		r.m.stage++
		a.Goto(newMarathonPlay(a, r.m))
	case "Завершить забег":
		r.finish(a)
	case "Новый забег":
		a.Goto(newMarathonPlay(a, newMarathonRun()))
	default:
		a.Goto(newMenuScene())
	}
}

func (r *marathonScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	m := r.m
	drawText(dst, "Марафон недели "+weekLabel(m.week), 20, false, ScreenW/2, 28, Center, colDim)
	drawPanel(dst, 60, 70, 640, 590)
	cx := 380.0
	if !r.over {
		drawText(dst, "Этап "+itoa(m.done)+" пройден!", 44, true, cx, 92, Center, colGreen)
		drawMedal(dst, cx, 196, 24, r.stageRes.Medal)
		drawText(dst, "+"+fmtScore(r.stageRes.Total)+" за этап", 24, true, cx, 230, Center, colGold)
		drawText(dst, "Всего очков", 20, false, cx, 290, Center, colDim)
		k := clamp(float64(r.t)/40, 0, 1)
		drawText(dst, fmtScore(m.score-r.stageRes.Total+int(float64(r.stageRes.Total)*k)), 48, true, cx, 314, Center, colText)
		for i := 0; i < sim.MarathonLives; i++ {
			c := rgba(255, 80, 100, 255)
			if i >= m.lives {
				c = rgba(70, 50, 60, 255)
			}
			drawHeart(dst, cx-30+float64(i)*30, 400, 1.2, c)
		}
		if r.healed {
			drawText(dst, "Каждые "+itoa(sim.MarathonHealEvery)+" этапов — жизнь в подарок!", 18, false, cx, 424, Center, rgba(255, 140, 160, 255))
		} else {
			left := sim.MarathonHealEvery - m.done%sim.MarathonHealEvery
			drawText(dst, "До подарочной жизни: "+itoa(left)+" "+stagesRu(left), 17, false, cx, 424, Center, colDim)
		}
		next := sim.MarathonTrack(m.week, m.stage+1)
		drawText(dst, "Дальше: "+sim.Worlds[next.World].Name+" · уровень "+itoa(next.Level)+" · "+sim.Difficulties[next.Diff].Name,
			19, false, cx, 470, Center, colAccent)
		drawItem(dst, next.World, cx, 530, 1.1, float64(a.Frame), 0)
	} else {
		drawText(dst, "Марафон окончен", 44, true, cx, 92, Center, colGold)
		if r.reason != "" {
			drawText(dst, r.reason, 22, false, cx, 150, Center, colRed)
		}
		drawText(dst, "Пройдено этапов", 20, false, cx, 200, Center, colDim)
		drawText(dst, itoa(m.done), 64, true, cx, 224, Center, colText)
		drawText(dst, "Очки", 20, false, cx, 320, Center, colDim)
		drawText(dst, fmtScore(m.score), 48, true, cx, 344, Center, colGold)
		drawText(dst, "Время в пути: "+fmtTime(m.ticks), 18, false, cx, 410, Center, colDim)
		msg, mc := "", colDim
		switch {
		case m.done == 0:
			msg = "Первый этап не пройден — попробуй ещё раз"
		case r.hadPrev && m.score > r.prevBest.Score:
			msg, mc = "Новый рекорд недели! +"+fmtScore(m.score-r.prevBest.Score), colGold
		case !r.hadPrev:
			msg, mc = "Первый результат этой недели", colAccent
		default:
			msg = "До личного рекорда не хватило " + fmtScore(r.prevBest.Score-m.score)
		}
		pulse := 1.0
		if mc == colGold {
			pulse = 0.75 + 0.25*math.Sin(float64(r.t)*0.15)
		}
		drawText(dst, msg, 22, true, cx, 460, Center, withAlpha(mc, pulse))
	}

	r.hits.reset()
	for i, it := range r.items {
		y := 594 + float64(i)*30
		r.hits.add(rect{100, y - 4, 560, 30}, i)
		c := colDim
		if i == r.sel {
			c = colAccent
			drawText(dst, "›", 22, true, 250, y-2, Center, colAccent)
		}
		drawText(dst, it, 20, i == r.sel, cx, y, Center, c)
	}

	drawPanel(dst, 730, 70, 490, 590)
	if r.over {
		drawText(dst, "Рекорды недели", 26, true, 975, 90, Center, colText)
		drawLeaderboard(dst, r.top, 750, 140, 450, r.name, r.rank)
		if r.rank > 0 {
			drawText(dst, "Твоё место: "+itoa(r.rank), 22, true, 975, 600, Center, colGold)
		}
	} else {
		drawText(dst, "Как устроен марафон", 26, true, 975, 90, Center, colText)
		rules := []string{
			"Уровни идут подряд по всем",
			"пяти мирам и постепенно",
			"становятся сложнее.",
			"",
			"Жизни переносятся на",
			"следующий этап, а каждые " + itoa(sim.MarathonHealEvery),
			"этапов одна возвращается.",
			"",
			"Очки за этапы складываются.",
			"Уровни одинаковы у всех",
			"игроков всю неделю, так что",
			"рекорды честно сравнимы.",
		}
		for i, s := range rules {
			drawText(dst, s, 19, false, 760, 150+float64(i)*30, Left, colDim)
		}
	}
	if !a.Pointer.touchMode {
		keyHint(dst, "Enter — выбрать   ·   ↑↓ — пункт")
	}
}

// weekLabel — «47 (2026)» из 2026-W47.
func weekLabel(week string) string {
	if y, w, ok := strings.Cut(week, "-W"); ok {
		return strings.TrimLeft(w, "0") + " · " + y
	}
	return week
}

func stagesRu(n int) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return "этап"
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return "этапа"
	}
	return "этапов"
}
