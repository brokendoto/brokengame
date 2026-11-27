package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Герои нарисованы кодом в стиле «чиби»: большая голова, маленькое тело,
// ножки шагают при ходьбе, взгляд следит за направлением. Своя одежда и
// шляпа у каждого мира. Никаких чужих спрайтов: всё из кругов и
// многоугольников.

// heroBuf — холст под одного героя. Героя рисуем сюда, а потом целиком
// переносим на поле: так у полупрозрачного призрака не просвечивают
// детали друг сквозь друга.
var heroBuf *ebiten.Image

// heroSkin — выбранный игроком герой (save.Profile.Skin): 0 — герой своего мира.
var heroSkin int

const heroBufW, heroBufH = 96, 96

// heroScale — во сколько раз герой крупнее чертежа (рисуем в масштабе
// клетки 48 пикселей).
const heroScale = 1.2

type heroLook struct {
	skin, outfit, outfit2, outline rgbaT
}

var heroLooks = [sim.WorldCount]heroLook{
	sim.Mine:   {rgba(255, 214, 170, 255), rgba(70, 96, 170, 255), rgba(232, 138, 42, 255), rgba(36, 24, 20, 255)},
	sim.Forest: {rgba(214, 120, 50, 255), rgba(214, 120, 50, 255), rgba(250, 222, 182, 255), rgba(60, 30, 12, 255)},
	sim.Temple: {rgba(250, 210, 170, 255), rgba(44, 52, 108, 255), rgba(200, 60, 60, 255), rgba(12, 12, 24, 255)},
	sim.Front:  {rgba(240, 196, 150, 255), rgba(104, 116, 66, 255), rgba(120, 82, 50, 255), rgba(22, 24, 14, 255)},
	sim.Space:  {rgba(255, 220, 190, 255), rgba(238, 238, 244, 255), rgba(230, 120, 40, 255), rgba(50, 54, 66, 255)},
}

func drawHero(dst *ebiten.Image, w sim.World, p sim.Player, x, y, t, alpha float64, ghost bool) {
	if heroSkin > 0 && heroSkin <= int(sim.WorldCount) {
		w = sim.World(heroSkin - 1)
	}
	if heroBuf == nil {
		heroBuf = ebiten.NewImage(heroBufW, heroBufH)
	}
	heroBuf.Clear()
	cx, cy := float64(heroBufW)/2, float64(heroBufH)/2+6
	// Тень на полу.
	fillPoly(dst, ellipse(x+1, y+18, 15, 5, 0, 16), withAlpha(rgba(0, 0, 0, 255), 0.3*alpha))
	drawChibi(heroBuf, w, p, cx, cy, t, ghost)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-cx, -cy)
	op.GeoM.Scale(heroScale, heroScale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(float32(alpha))
	if ghost {
		op.ColorScale.Scale(0.75, 0.9, 1.25, 1)
	}
	dst.DrawImage(heroBuf, op)
	if p.ShieldT > 0 && !ghost {
		a := 0.6
		if p.ShieldT < 60 && (p.ShieldT/6)%2 == 0 {
			a = 0.2
		}
		strokeCircle(dst, x, y, CellPx*0.32+11+math.Sin(t*0.3)*1.5, 3, withAlpha(colAccent, a))
	}
}

// drawChibi рисует героя мира w с центром тела в (x, y).
func drawChibi(dst *ebiten.Image, w sim.World, p sim.Player, x, y, t float64, ghost bool) {
	lk := heroLooks[w]
	ol := lk.outline
	dir := 1.0 // куда повёрнут: 1 вправо, -1 влево
	if p.FX < -0.1 {
		dir = -1
	}
	back := p.FY < -0.5 && math.Abs(p.FX) < 0.5 // смотрит вверх: видна спина
	ph := p.WalkDistance * 9
	step := 0.0
	if p.Moving {
		step = math.Sin(ph)
		y -= math.Abs(math.Cos(ph)) * 1.6
	}
	if p.DashT > 0 {
		step = 0
	}
	// blob — фигура с тёмной обводкой.
	blob := func(cx, cy, r float64, c rgbaT) {
		fillCircle(dst, cx, cy, r+2, ol)
		fillCircle(dst, cx, cy, r, c)
	}
	oval := func(cx, cy, rx, ry float64, c rgbaT) {
		fillPoly(dst, ellipse(cx, cy, rx+2, ry+2, 0, 20), ol)
		fillPoly(dst, ellipse(cx, cy, rx, ry, 0, 20), c)
	}

	// То, что за спиной, рисуем первым.
	switch w {
	case sim.Forest: // пушистый хвост
		sw := math.Sin(t*0.12) * 2
		tx := x - dir*13
		oval(tx, y-2+sw*0.3, 7, 11, rgba(180, 96, 38, 255))
		oval(tx-dir*3, y-12+sw, 6, 7, rgba(200, 112, 46, 255))
		fillPoly(dst, ellipse(tx-dir*2, y-12+sw, 3, 4, 0, 12), rgba(240, 190, 120, 255))
	case sim.Temple: // катана за спиной
		line(dst, x-dir*10, y+8, x+dir*8, y-20, 5, ol)
		line(dst, x-dir*10, y+8, x+dir*8, y-20, 3, rgba(190, 196, 210, 255))
		line(dst, x-dir*6, y+2, x-dir*10, y+8, 4, rgba(140, 40, 40, 255))
	case sim.Front: // шарф развевается
		f := math.Sin(t*0.3) * 2
		fillPoly(dst, []float64{x - dir*2, y - 4, x - dir*16, y - 6 + f, x - dir*18, y - 1 + f, x - dir*3, y + 1}, rgba(210, 40, 40, 255))
	case sim.Space: // ранец
		fillRect(dst, x-dir*12-6, y-6, 12, 16, ol)
		roundRect(dst, x-dir*12-5, y-5, 10, 14, 3, rgba(170, 170, 184, 255))
		fillCircle(dst, x-dir*12, y+1, 2, rgba(90, 220, 255, 255))
	}

	// Ножки.
	for i, s := range []float64{-1, 1} {
		lift := step * s
		fx := x + s*4 + dir*lift*2.5
		fy := y + 12 - math.Max(0, lift)*2.5
		c := rgba(60, 44, 36, 255)
		if w == sim.Space {
			c = rgba(200, 200, 210, 255)
		} else if w == sim.Forest {
			c = rgba(150, 80, 30, 255)
		}
		_ = i
		oval(fx, fy, 3.6, 2.6, c)
	}
	// Тело.
	fillCircle(dst, x, y+4, 10, ol)
	roundRect(dst, x-8, y-4, 16, 16, 6, lk.outfit)
	switch w {
	case sim.Mine: // оранжевая рубашка и синий комбинезон с лямками
		roundRect(dst, x-8, y-4, 16, 7, 4, lk.outfit2)
		line(dst, x-4.5, y-4, x-4, y+3, 2.5, lk.outfit)
		line(dst, x+4.5, y-4, x+4, y+3, 2.5, lk.outfit)
		fillCircle(dst, x, y+6, 1.6, rgba(250, 210, 60, 255))
	case sim.Forest: // светлое брюшко
		fillPoly(dst, ellipse(x+dir*1.5, y+5, 5, 6, 0, 14), lk.outfit2)
	case sim.Temple: // запах кимоно и пояс
		line(dst, x-dir*5, y-4, x+dir*3, y+5, 2.5, rgba(220, 220, 230, 255))
		fillRect(dst, x-8, y+5, 16, 3, lk.outfit2)
	case sim.Front: // куртка с ремнём
		fillRect(dst, x-8, y+4, 16, 2.5, rgba(60, 40, 24, 255))
		fillRect(dst, x-1.5, y+3.5, 3, 3.5, rgba(220, 190, 80, 255))
	case sim.Space: // нашивка
		fillCircle(dst, x+dir*3, y+2, 2.6, lk.outfit2)
	}
	// Ручки качаются в такт шагам.
	for _, s := range []float64{-1, 1} {
		sw := step * s * 2.5
		blob(x+s*9+dir*sw*0.5, y+3-math.Abs(sw)*0.3, 3, lk.outfit)
	}

	// Голова.
	hx, hy := x+dir*1, y-11
	if w == sim.Space {
		// Шлем-аквариум: стекло, внутри лицо.
		blob(hx, hy, 12, rgba(220, 224, 236, 255))
		fillCircle(dst, hx+dir*1.5, hy+0.5, 9, rgba(30, 60, 120, 255))
		if !back {
			fillCircle(dst, hx+dir*2.5, hy+1.5, 6.5, lk.skin)
			eyes(dst, hx+dir*4, hy+1, dir, 2.4)
		}
		fillPoly(dst, ellipse(hx-dir*3, hy-4, 3.5, 2, -dir*0.6, 10), rgba(220, 240, 255, 200))
		fillCircle(dst, hx, hy-13, 1.8, ol)
		line(dst, hx, hy-11, hx, hy-14, 1.5, ol)
		return
	}
	blob(hx, hy, 10, lk.skin)
	if !back {
		eyes(dst, hx+dir*3, hy+1, dir, 3.2)
		fillCircle(dst, hx+dir*1-4.5, hy+5, 2, rgba(255, 140, 140, 150))
		fillCircle(dst, hx+dir*1+5.5, hy+5, 2, rgba(255, 140, 140, 150))
	}
	switch w {
	case sim.Mine: // каска с фонарём
		fillPoly(dst, halfDome(hx, hy-2, 12.5), ol)
		fillPoly(dst, halfDome(hx, hy-2, 10.8), rgba(250, 210, 60, 255))
		fillRect(dst, hx-12.5, hy-3, 25, 3.2, ol)
		fillRect(dst, hx-11.5, hy-2.5, 23, 2, rgba(220, 170, 40, 255))
		fillCircle(dst, hx-3, hy-8, 2.2, rgba(255, 245, 190, 255))
		if !back {
			fillCircle(dst, hx+dir*8, hy-7, 4, ol)
			fillCircle(dst, hx+dir*8, hy-7, 2.8, rgba(255, 255, 220, 255))
		}
	case sim.Forest: // ушки с кисточками и мордочка
		for _, s := range []float64{-1, 1} {
			ex := hx + s*6
			fillPoly(dst, []float64{ex - 5, hy - 5, ex + s*1, hy - 17, ex + 4, hy - 5}, ol)
			fillPoly(dst, []float64{ex - 3, hy - 6, ex + s*1, hy - 14, ex + 2.5, hy - 6}, lk.skin)
			line(dst, ex+s*1, hy-14, ex+s*1, hy-19, 2, rgba(110, 50, 20, 255))
		}
		if !back {
			fillPoly(dst, ellipse(hx+dir*5, hy+4, 4.5, 3.5, 0, 12), lk.outfit2)
			fillCircle(dst, hx+dir*8, hy+3, 1.6, ol)
		}
	case sim.Temple: // широкая соломенная шляпа-каса
		fillPoly(dst, []float64{hx - 17, hy - 2, hx, hy - 15, hx + 17, hy - 2, hx + 15, hy + 0.5, hx - 15, hy + 0.5}, ol)
		fillPoly(dst, []float64{hx - 14.5, hy - 2, hx, hy - 12.5, hx + 14.5, hy - 2}, rgba(214, 182, 110, 255))
		line(dst, hx-7, hy-6, hx+7, hy-6, 1.2, rgba(160, 126, 70, 255))
		line(dst, hx, hy-12.5, hx, hy-2, 1.2, rgba(160, 126, 70, 255))
	case sim.Front: // кожаный шлем и очки на лбу
		fillPoly(dst, halfDome(hx, hy-1, 11.8), ol)
		fillPoly(dst, halfDome(hx, hy-1, 10.2), lk.outfit2)
		for _, s := range []float64{-1, 1} {
			fillCircle(dst, hx+dir*2+s*4.5, hy-5, 3.6, ol)
			fillCircle(dst, hx+dir*2+s*4.5, hy-5, 2.6, rgba(120, 230, 255, 255))
		}
		if !back {
			fillPoly(dst, ellipse(hx-dir*9, hy+1, 2.5, 4, 0, 10), lk.outfit2)
		}
	}
}

// eyes — два глаза с бликом, смещённые в сторону взгляда.
func eyes(dst *ebiten.Image, x, y, dir, r float64) {
	for _, s := range []float64{-1, 1} {
		ex := x + s*r*1.25
		fillPoly(dst, ellipse(ex, y, r*0.62, r*0.85, 0, 12), rgba(30, 20, 30, 255))
		fillCircle(dst, ex+dir*r*0.18-r*0.12, y-r*0.35, r*0.25, rgba(255, 255, 255, 255))
	}
}

// halfDome — верхняя половина круга (каска, шлем).
func halfDome(x, y, r float64) []float64 {
	pts := []float64{}
	for i := 0; i <= 16; i++ {
		a := math.Pi + float64(i)/16*math.Pi
		pts = append(pts, x+math.Cos(a)*r, y+math.Sin(a)*r)
	}
	return pts
}
