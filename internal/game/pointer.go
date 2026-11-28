package game

import (
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Касания и мышь. На телефоне это единственный способ управления, на компьютере
// мышью тоже можно нажимать пункты меню.

// IsMobile — запущена ли игра на телефоне или планшете.
func IsMobile() bool { return runtime.GOOS == "android" || runtime.GOOS == "ios" }

// touchFirst — управление в первую очередь пальцами: телефон или веб-версия
// на сенсорном экране. Значение не меняется, поэтому считаем один раз.
var touchFirst = touchScreen()

type touchState struct {
	sx, sy   float64 // где палец коснулся экрана
	x, y     float64 // где он сейчас
	frames   int
	maxMoved float64
}

// pointer собирает касания и клики за кадр.
type pointer struct {
	touches   map[ebiten.TouchID]*touchState
	ids       []ebiten.TouchID
	tapped    bool
	tapX      float64
	tapY      float64
	touchMode bool // показывать экранные кнопки
}

func newPointer() *pointer {
	return &pointer{touches: map[ebiten.TouchID]*touchState{}, touchMode: touchFirst}
}

func (p *pointer) update() {
	p.tapped = false
	p.ids = inpututil.AppendJustPressedTouchIDs(p.ids[:0])
	for _, id := range p.ids {
		x, y := ebiten.TouchPosition(id)
		p.touches[id] = &touchState{sx: float64(x), sy: float64(y), x: float64(x), y: float64(y)}
		p.touchMode = true
	}
	for id, t := range p.touches {
		if inpututil.IsTouchJustReleased(id) {
			if t.maxMoved < 24 && t.frames < 45 {
				p.tapped, p.tapX, p.tapY = true, t.sx, t.sy
			}
			delete(p.touches, id)
			continue
		}
		x, y := ebiten.TouchPosition(id)
		t.x, t.y = float64(x), float64(y)
		t.frames++
		t.maxMoved = math.Max(t.maxMoved, math.Hypot(t.x-t.sx, t.y-t.sy))
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		p.tapped, p.tapX, p.tapY = true, float64(x), float64(y)
	}
	if x, y, touch, ok := lostTap(); ok && !p.tapped {
		p.tapped, p.tapX, p.tapY = true, x, y
		if touch {
			p.touchMode = true
		}
	}
	// Клавиатура или геймпад — прячем экранные кнопки (кроме телефонов).
	if !touchFirst && (len(inpututil.AppendJustPressedKeys(nil)) > 0) {
		p.touchMode = false
	}
}

// rect — прямоугольная зона нажатия.
type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// hits — зоны нажатия, которые сцена регистрирует при отрисовке
// и проверяет в следующем Update.
type hits struct {
	list []hit
}

type hit struct {
	r  rect
	id int
}

func (h *hits) reset()             { h.list = h.list[:0] }
func (h *hits) add(r rect, id int) { h.list = append(h.list, hit{r, id}) }

// tapped возвращает id зоны, по которой был тап в этом кадре.
func (h *hits) tapped(a *App) (int, bool) {
	if !a.Pointer.tapped {
		return 0, false
	}
	for i := len(h.list) - 1; i >= 0; i-- {
		if h.list[i].r.has(a.Pointer.tapX, a.Pointer.tapY) {
			return h.list[i].id, true
		}
	}
	return 0, false
}

// hover возвращает id зоны под курсором мыши (на компьютере).
func (h *hits) hover() (int, bool) {
	if touchFirst {
		return 0, false
	}
	x, y := ebiten.CursorPosition()
	for i := len(h.list) - 1; i >= 0; i-- {
		if h.list[i].r.has(float64(x), float64(y)) {
			return h.list[i].id, true
		}
	}
	return 0, false
}

// --- экранные кнопки в игре ------------------------------------------------

const (
	stickR      = 90.0
	dashBtnX    = 1172.0
	dashBtnY    = 610.0
	dashBtnR    = 62.0
	shieldBtnX  = 1040.0
	shieldBtnY  = 652.0
	shieldBtnR  = 46.0
	stickZoneW  = ScreenW * 0.5
	hudBottom   = 80.0
	stickDeadPx = 14.0
)

// touchControls — плавающий джойстик слева и кнопки справа.
type touchControls struct {
	stickID  ebiten.TouchID
	stickOn  bool
	ox, oy   float64
	kx, ky   float64
	dashOn   bool
	shieldOn bool
	pauseTap bool

	// Раскладка: зона джойстика по x и места кнопок. Нулевая — одиночная игра.
	x0, x1       float64
	dashX, dashY float64
	abX, abY     float64
	label        string // подпись над джойстиком (в дуэли — «Игрок 1»)
	tint         rgbaT
}

// duelTouch — раскладка для дуэли на планшете: каждому игроку половина экрана.
func duelTouch(i int) touchControls {
	if i == 0 {
		return touchControls{x0: 0, x1: ScreenW / 2, dashX: 560, dashY: 610, abX: 440, abY: 652, label: "Игрок 1", tint: duelColors[0]}
	}
	return touchControls{x0: ScreenW / 2, x1: ScreenW, dashX: dashBtnX, dashY: dashBtnY, abX: shieldBtnX, abY: shieldBtnY, label: "Игрок 2", tint: duelColors[1]}
}

func (c *touchControls) layout() {
	if c.x1 == 0 {
		c.x0, c.x1 = 0, stickZoneW
		c.dashX, c.dashY, c.abX, c.abY = dashBtnX, dashBtnY, shieldBtnX, shieldBtnY
	}
}

func (c *touchControls) update(p *pointer) sim.Input {
	c.layout()
	c.dashOn, c.shieldOn, c.pauseTap = false, false, false
	if c.stickOn {
		if _, ok := p.touches[c.stickID]; !ok {
			c.stickOn = false
		}
	}
	for id, t := range p.touches {
		switch {
		case c.stickOn && id == c.stickID:
			c.kx, c.ky = t.x, t.y
		case math.Hypot(t.x-c.dashX, t.y-c.dashY) < dashBtnR*1.25:
			c.dashOn = true
		case math.Hypot(t.x-c.abX, t.y-c.abY) < shieldBtnR*1.3:
			c.shieldOn = true
		case !c.stickOn && t.sx >= c.x0 && t.sx < c.x1 && t.sy > hudBottom:
			c.stickID, c.stickOn = id, true
			c.ox, c.oy, c.kx, c.ky = t.sx, t.sy, t.x, t.y
		}
	}
	if p.tapped && p.tapY < hudBottom {
		c.pauseTap = true
	}
	var in sim.Input
	if c.stickOn {
		dx, dy := c.kx-c.ox, c.ky-c.oy
		d := math.Hypot(dx, dy)
		// Джойстик «тянется» за пальцем, если тот ушёл далеко.
		if d > stickR {
			c.ox += dx / d * (d - stickR)
			c.oy += dy / d * (d - stickR)
		}
		if d > stickDeadPx {
			nx, ny := dx/d, dy/d
			// 8 направлений: компонента больше sin(22.5°).
			if nx > 0.38 {
				in |= sim.InRight
			} else if nx < -0.38 {
				in |= sim.InLeft
			}
			if ny > 0.38 {
				in |= sim.InDown
			} else if ny < -0.38 {
				in |= sim.InUp
			}
		}
	}
	if c.dashOn {
		in |= sim.InDash
	}
	if c.shieldOn {
		in |= sim.InAbility
	}
	return in
}

func (c *touchControls) draw(dst *ebiten.Image, dashCD, abCD int, ab sim.Ability) {
	c.layout()
	white := rgba(255, 255, 255, 255)
	if c.stickOn {
		strokeCircle(dst, c.ox, c.oy, stickR, 3, withAlpha(white, 0.35))
		fillCircle(dst, c.ox, c.oy, stickR, withAlpha(rgba(0, 0, 0, 255), 0.18))
		kx, ky := c.kx, c.ky
		if d := math.Hypot(kx-c.ox, ky-c.oy); d > stickR {
			kx, ky = c.ox+(kx-c.ox)/d*stickR, c.oy+(ky-c.oy)/d*stickR
		}
		fillCircle(dst, kx, ky, 38, withAlpha(white, 0.45))
	} else {
		strokeCircle(dst, c.x0+170, 590, stickR, 3, withAlpha(white, 0.15))
		drawText(dst, "двигай пальцем", 16, false, c.x0+170, 580, Center, withAlpha(white, 0.3))
		if c.label != "" {
			drawText(dst, c.label, 18, true, c.x0+170, 470, Center, withAlpha(c.tint, 0.6))
		}
	}
	btn := func(x, y, r float64, label string, on bool, cd int, c rgbaT) {
		a := 0.35
		if on {
			a = 0.7
		}
		if cd > 0 {
			a = 0.15
		}
		fillCircle(dst, x, y, r, withAlpha(c, a))
		strokeCircle(dst, x, y, r, 3, withAlpha(white, 0.5))
		size := 18
		if len([]rune(label)) > 6 {
			size = 14
		}
		drawText(dst, label, size, true, x, y-float64(size)*0.6, Center, withAlpha(white, 0.9))
	}
	btn(c.dashX, c.dashY, dashBtnR, "Рывок", c.dashOn, dashCD, rgba(255, 170, 60, 255))
	btn(c.abX, c.abY, shieldBtnR, sim.Abilities[ab].Name, c.shieldOn, abCD, abilityColor(ab))
}

// drawBackButton рисует кнопку «Назад» в углу и регистрирует зону нажатия.
func drawBackButton(dst *ebiten.Image, h *hits, id int) {
	const y = ScreenH - 66
	roundRect(dst, 16, y, 130, 50, 12, rgba(40, 32, 60, 230))
	roundRectStroke(dst, 16, y, 130, 50, 12, 2, rgba(90, 80, 120, 200))
	drawText(dst, "‹ Назад", 20, true, 81, y+12, Center, colText)
	h.add(rect{4, y - 12, 160, 76}, id)
}
