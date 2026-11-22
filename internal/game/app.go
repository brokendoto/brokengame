// Package game — всё, что связано с экраном: сцены, отрисовка, ввод, звук.
// Правила игры живут в пакете sim и ничего не знают об Ebitengine.
package game

import (
	"image/color"
	"math"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/save"
	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// Scene — экран игры (меню, уровень, итоги...).
type Scene interface {
	Update(a *App)
	Draw(a *App, dst *ebiten.Image)
}

// App — корневой объект, реализует ebiten.Game.
type App struct {
	Prof    *save.Profile
	Records save.Leaderboard
	Rec     *save.Records
	Sound   *sfx.Bank
	Frame   int
	Pointer *pointer

	scene    Scene
	next     Scene
	fade     float64
	quit     bool
	toast    string
	toastT   int
	bgDust   []dust
	Headless bool // без окна (для скриншотов и тестов)
}

type dust struct{ x, y, vx, vy, r, phase float64 }

// New создаёт игру и загружает сохранения.
func New() *App {
	dir := save.Dir()
	a := &App{
		Prof:    save.LoadProfile(dir),
		Rec:     save.LoadRecords(dir),
		Sound:   sfx.NewBank(),
		Pointer: newPointer(),
	}
	a.Records = a.Rec
	a.Sound.Muted = a.Prof.Muted
	r := sim.NewRNG(uint64(time.Now().UnixNano()))
	for i := 0; i < 70; i++ {
		a.bgDust = append(a.bgDust, dust{
			x: r.Float() * ScreenW, y: r.Float() * ScreenH,
			vx: (r.Float() - 0.5) * 0.3, vy: -0.1 - r.Float()*0.35,
			r: 1 + r.Float()*2.5, phase: r.Float() * 6.28,
		})
	}
	if a.Prof.Name == "" {
		a.scene = newNameScene(true)
	} else {
		a.scene = newMenuScene()
	}
	return a
}

// Goto плавно переключает сцену.
func (a *App) Goto(s Scene) {
	a.next = s
}

// Toast показывает короткое сообщение внизу экрана.
func (a *App) Toast(s string) { a.toast, a.toastT = s, 150 }

// Update вызывается Ebitengine 60 раз в секунду.
func (a *App) Update() error {
	a.Frame++
	a.Pointer.update()
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) ||
		(inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		a.SetFullscreen(!ebiten.IsFullscreen())
	}
	if _, typing := a.scene.(*nameScene); !typing && inpututil.IsKeyJustPressed(ebiten.KeyM) {
		a.SetMuted(!a.Sound.Muted)
	}
	for i := range a.bgDust {
		d := &a.bgDust[i]
		d.x += d.vx
		d.y += d.vy
		if d.y < -10 {
			d.y, d.x = ScreenH+10, math.Mod(d.x*7.31, ScreenW)
		}
	}
	if a.toastT > 0 {
		a.toastT--
	}
	if a.next != nil {
		a.fade += 0.12
		if a.fade >= 1 {
			a.scene, a.next = a.next, nil
		}
		return nil
	}
	if a.fade > 0 {
		a.fade -= 0.12
	}
	a.scene.Update(a)
	if a.quit {
		return ebiten.Termination
	}
	return nil
}

// Draw рисует кадр.
func (a *App) Draw(screen *ebiten.Image) {
	a.scene.Draw(a, screen)
	if a.toastT > 0 {
		al := math.Min(1, float64(a.toastT)/20)
		w := textWidth(a.toast, 20, false) + 40
		roundRect(screen, (ScreenW-w)/2, ScreenH-64, w, 40, 12, withAlpha(rgba(20, 16, 30, 230), al))
		drawText(screen, a.toast, 20, false, ScreenW/2, ScreenH-56, Center, withAlpha(rgba(240, 230, 255, 255), al))
	}
	if a.fade > 0 {
		fillRect(screen, 0, 0, ScreenW, ScreenH, withAlpha(rgba(8, 6, 12, 255), a.fade))
	}
}

// Layout задаёт логическое разрешение.
func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }

// Quit завершает игру.
func (a *App) Quit() { a.quit = true }

// CanQuit — есть ли смысл в пункте «Выход» (в браузере его нет).
func CanQuit() bool { return runtime.GOOS != "js" && !IsMobile() }

// SetFullscreen переключает полноэкранный режим и запоминает выбор.
func (a *App) SetFullscreen(on bool) {
	ebiten.SetFullscreen(on)
	a.Prof.Fullscreen = on
	_ = a.Prof.Save()
}

// SetMuted включает/выключает звук.
func (a *App) SetMuted(m bool) {
	a.Sound.Muted = m
	a.Prof.Muted = m
	_ = a.Prof.Save()
	if m {
		a.Toast("Звук выключен (M)")
	} else {
		a.Toast("Звук включён (M)")
	}
}

// Палитра интерфейса.
var (
	colText   = rgba(240, 232, 220, 255)
	colDim    = rgba(160, 150, 170, 255)
	colAccent = rgba(120, 230, 255, 255)
	colGold   = rgba(255, 205, 80, 255)
	colSilver = rgba(205, 215, 230, 255)
	colBronze = rgba(220, 140, 80, 255)
	colRed    = rgba(255, 90, 90, 255)
	colGreen  = rgba(120, 240, 150, 255)
	colPanel  = rgba(24, 20, 32, 235)
)

func medalColor(m sim.Medal) color.RGBA {
	switch m {
	case sim.Gold:
		return colGold
	case sim.Silver:
		return colSilver
	case sim.Bronze:
		return colBronze
	}
	return rgba(70, 64, 80, 255)
}

// drawBackground — общий фон меню: тёмная пещера и парящая кристальная пыль.
func (a *App) drawBackground(dst *ebiten.Image) {
	dst.Fill(rgba(14, 11, 20, 255))
	for y := 0; y < ScreenH; y += 8 {
		t := float64(y) / ScreenH
		fillRect(dst, 0, float64(y), ScreenW, 8, lerpColor(rgba(22, 16, 34, 255), rgba(10, 18, 26, 255), t))
	}
	for _, d := range a.bgDust {
		tw := 0.5 + 0.5*math.Sin(float64(a.Frame)*0.04+d.phase)
		c := rgba(110, 220, 255, 255)
		if int(d.phase*10)%3 == 0 {
			c = rgba(220, 120, 255, 255)
		}
		glow(dst, d.x, d.y, d.r*6, c, 0.25*tw)
		fillCircle(dst, d.x, d.y, d.r*0.6, withAlpha(c, 0.6*tw+0.2))
	}
}

// drawMedal рисует значок медали.
func drawMedal(dst *ebiten.Image, x, y, r float64, m sim.Medal) {
	c := medalColor(m)
	if m == sim.NoMedal {
		strokeCircle(dst, x, y, r, 2, c)
		return
	}
	fillPoly(dst, []float64{x - r*0.7, y - r*1.9, x - r*0.1, y - r*0.6, x - r*0.6, y - r*0.4, x - r*1.2, y - r*1.6}, rgba(200, 60, 70, 255))
	fillPoly(dst, []float64{x + r*0.7, y - r*1.9, x + r*0.1, y - r*0.6, x + r*0.6, y - r*0.4, x + r*1.2, y - r*1.6}, rgba(60, 90, 200, 255))
	fillCircle(dst, x, y, r, c)
	fillCircle(dst, x, y, r*0.7, lerpColor(c, rgba(255, 255, 255, 255), 0.35))
	fillCircle(dst, x, y, r*0.45, c)
}

// drawPanel — полупрозрачная панель.
func drawPanel(dst *ebiten.Image, x, y, w, h float64) {
	roundRect(dst, x, y, w, h, 16, colPanel)
	roundRectStroke(dst, x, y, w, h, 16, 2, rgba(90, 80, 120, 200))
}

// keyHint рисует подсказку управления внизу экрана.
func keyHint(dst *ebiten.Image, s string) {
	drawText(dst, s, 16, false, ScreenW/2, ScreenH-30, Center, colDim)
}

// today — дата для испытания дня (по UTC, чтобы у всех игроков совпадала).
func today() string { return time.Now().UTC().Format("2006-01-02") }
