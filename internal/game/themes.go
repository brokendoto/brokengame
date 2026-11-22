package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Оформление пяти миров. Каждый мир — своя палитра и свои фигуры героя,
// предметов, врагов и ловушек. Вся графика процедурная.

type theme struct {
	floorA, floorB, speck    rgbaT
	wall, wallHi, wallLo     rgbaT
	zone, zoneDark, zoneHi   rgbaT
	bg                       rgbaT
	itemGlow                 rgbaT
	darkness                 float64 // 0 — светло, 0.6 — темно, светит только фонарь
	lampColor                rgbaT
	eyeBasic, eyeSpecial     rgbaT
	beam                     rgbaT
	short                    string // короткое имя мира для HUD
	itemWord, itemsWordShort string
}

var themes = [sim.WorldCount]theme{
	sim.Mine: {
		floorA: rgba(46, 38, 34, 255), floorB: rgba(38, 31, 29, 255), speck: rgba(60, 51, 46, 255),
		wall: rgba(84, 72, 64, 255), wallHi: rgba(122, 106, 92, 255), wallLo: rgba(52, 43, 38, 255),
		zone: rgba(78, 52, 30, 255), zoneDark: rgba(64, 42, 24, 255), zoneHi: rgba(110, 78, 48, 255),
		bg: rgba(12, 10, 14, 255), itemGlow: rgba(90, 220, 255, 255),
		darkness: 0.62, lampColor: rgba(255, 220, 150, 255),
		eyeBasic: rgba(255, 60, 60, 255), eyeSpecial: rgba(120, 255, 90, 255),
		beam: rgba(255, 150, 60, 255), short: "Шахта",
	},
	sim.Forest: {
		floorA: rgba(70, 92, 44, 255), floorB: rgba(62, 82, 40, 255), speck: rgba(96, 120, 56, 255),
		wall: rgba(44, 66, 34, 255), wallHi: rgba(84, 116, 52, 255), wallLo: rgba(30, 46, 24, 255),
		zone: rgba(56, 96, 130, 255), zoneDark: rgba(44, 78, 110, 255), zoneHi: rgba(150, 200, 230, 255),
		bg: rgba(20, 24, 14, 255), itemGlow: rgba(255, 200, 90, 255),
		darkness: 0.12, lampColor: rgba(255, 230, 180, 255),
		eyeBasic: rgba(255, 220, 90, 255), eyeSpecial: rgba(255, 140, 60, 255),
		beam: rgba(255, 170, 60, 255), short: "Лес",
	},
	sim.Temple: {
		floorA: rgba(40, 38, 52, 255), floorB: rgba(34, 32, 46, 255), speck: rgba(56, 52, 70, 255),
		wall: rgba(110, 36, 44, 255), wallHi: rgba(212, 170, 80, 255), wallLo: rgba(70, 22, 30, 255),
		zone: rgba(120, 120, 150, 255), zoneDark: rgba(90, 90, 120, 255), zoneHi: rgba(200, 200, 230, 255),
		bg: rgba(10, 8, 18, 255), itemGlow: rgba(160, 220, 255, 255),
		darkness: 0.5, lampColor: rgba(255, 190, 120, 255),
		eyeBasic: rgba(255, 220, 120, 255), eyeSpecial: rgba(200, 160, 255, 255),
		beam: rgba(200, 150, 255, 255), short: "Храм",
	},
	sim.Front: {
		floorA: rgba(66, 62, 54, 255), floorB: rgba(58, 55, 48, 255), speck: rgba(82, 77, 66, 255),
		wall: rgba(112, 100, 76, 255), wallHi: rgba(150, 136, 104, 255), wallLo: rgba(74, 66, 50, 255),
		zone: rgba(24, 22, 26, 255), zoneDark: rgba(14, 12, 16, 255), zoneHi: rgba(120, 90, 200, 255),
		bg: rgba(18, 16, 14, 255), itemGlow: rgba(120, 255, 140, 255),
		darkness: 0.28, lampColor: rgba(255, 230, 200, 255),
		eyeBasic: rgba(255, 70, 50, 255), eyeSpecial: rgba(255, 220, 40, 255),
		beam: rgba(255, 50, 50, 255), short: "Фронт",
	},
	sim.Space: {
		floorA: rgba(28, 32, 50, 255), floorB: rgba(24, 27, 44, 255), speck: rgba(50, 56, 84, 255),
		wall: rgba(66, 74, 104, 255), wallHi: rgba(120, 132, 176, 255), wallLo: rgba(40, 46, 68, 255),
		zone: rgba(14, 18, 40, 255), zoneDark: rgba(8, 10, 26, 255), zoneHi: rgba(150, 190, 255, 255),
		bg: rgba(4, 4, 12, 255), itemGlow: rgba(255, 230, 110, 255),
		darkness: 0.4, lampColor: rgba(200, 230, 255, 255),
		eyeBasic: rgba(140, 255, 120, 255), eyeSpecial: rgba(230, 120, 255, 255),
		beam: rgba(80, 220, 255, 255), short: "Космос",
	},
}

// --- статичная часть поля ------------------------------------------------------

func renderFloor(img *ebiten.Image, w sim.World, x, y int) {
	th := &themes[w]
	X, Y := float64(x*CellPx), float64(y*CellPx)
	h := cellHash(x, y, 1)
	fillRect(img, X, Y, CellPx, CellPx, lerpColor(th.floorA, th.floorB, float64(h%100)/100))
	switch w {
	case sim.Forest:
		for i := 0; i < 4; i++ {
			hh := cellHash(x, y, 10+i)
			gx, gy := X+float64(hh%40)+4, Y+float64(hh/40%40)+6
			line(img, gx, gy, gx-2, gy-6, 1.5, th.speck)
			line(img, gx, gy, gx+2, gy-5, 1.5, th.speck)
		}
		if h%7 == 0 { // опавший лист
			hh := cellHash(x, y, 20)
			lx, ly := X+float64(hh%30)+8, Y+float64(hh/30%30)+8
			fillPoly(img, []float64{lx, ly - 5, lx + 6, ly, lx, ly + 5, lx - 6, ly}, rgba(200, 110, 40, 255))
		}
	case sim.Temple:
		strokeRect(img, X+1, Y+1, CellPx-2, CellPx-2, 1.5, th.speck)
		if h%5 == 0 {
			line(img, X+8, Y+12, X+18, Y+20, 1.2, th.speck)
		}
	case sim.Front:
		for i := 0; i < 3; i++ {
			hh := cellHash(x, y, 10+i)
			fillCircle(img, X+float64(hh%40)+4, Y+float64(hh/40%40)+4, 1.5, th.speck)
		}
		if h%6 == 0 {
			hh := cellHash(x, y, 30)
			cx, cy := X+float64(hh%24)+12, Y+float64(hh/24%24)+12
			fillCircle(img, cx, cy, 9, rgba(48, 45, 40, 255)) // воронка
			fillCircle(img, cx-1, cy-1, 6, rgba(40, 37, 33, 255))
		}
	case sim.Space:
		strokeRect(img, X+2, Y+2, CellPx-4, CellPx-4, 1, th.speck)
		fillCircle(img, X+6, Y+6, 1.5, th.speck)
		fillCircle(img, X+CellPx-6, Y+CellPx-6, 1.5, th.speck)
	default:
		for i := 0; i < 3; i++ {
			hh := cellHash(x, y, 10+i)
			fillCircle(img, X+float64(hh%40)+4, Y+float64(hh/40%40)+4, 1.2+float64(hh%3)*0.6, th.speck)
		}
	}
}

func renderZone(img *ebiten.Image, w sim.World, x, y int) {
	th := &themes[w]
	X, Y := float64(x*CellPx), float64(y*CellPx)
	switch w {
	case sim.Forest: // лужа
		fillRect(img, X, Y, CellPx, CellPx, th.zone)
		hh := cellHash(x, y, 30)
		cx, cy := X+float64(hh%20)+14, Y+float64(hh/20%20)+14
		strokeCircle(img, cx, cy, 8, 1.5, withAlpha(th.zoneHi, 0.5))
		strokeCircle(img, cx, cy, 14, 1, withAlpha(th.zoneHi, 0.25))
	case sim.Temple: // туман рисуется поверх объектов, здесь только пол темнее
		renderFloor(img, w, x, y)
		fillRect(img, X, Y, CellPx, CellPx, withAlpha(th.zoneDark, 0.35))
	case sim.Front: // масляное пятно
		renderFloor(img, w, x, y)
		hh := cellHash(x, y, 30)
		fillCircle(img, X+24, Y+24, 22, th.zone)
		fillCircle(img, X+float64(hh%10)+14, Y+float64(hh/10%10)+14, 8, withAlpha(th.zoneHi, 0.35))
		fillCircle(img, X+float64(hh%10)+20, Y+float64(hh/10%10)+18, 5, withAlpha(rgba(80, 200, 160, 255), 0.25))
	case sim.Space: // зона невесомости
		fillRect(img, X, Y, CellPx, CellPx, th.zone)
		for i := 0; i < 3; i++ {
			hh := cellHash(x, y, 40+i)
			fillCircle(img, X+float64(hh%44)+2, Y+float64(hh/44%44)+2, 1+float64(hh%2), withAlpha(th.zoneHi, 0.7))
		}
		strokeRect(img, X+1, Y+1, CellPx-2, CellPx-2, 1, withAlpha(th.zoneHi, 0.2))
	default: // грязь
		fillRect(img, X, Y, CellPx, CellPx, th.zone)
		for i := 0; i < 4; i++ {
			hh := cellHash(x, y, 20+i)
			fillCircle(img, X+float64(hh%36)+6, Y+float64(hh/36%36)+6, 3+float64(hh%4), th.zoneDark)
		}
		hh := cellHash(x, y, 30)
		line(img, X+8+float64(hh%10), Y+12, X+20+float64(hh%10), Y+10, 2, th.zoneHi)
	}
}

func renderWall(img *ebiten.Image, lv *sim.Level, w sim.World, x, y int) {
	th := &themes[w]
	X, Y := float64(x*CellPx), float64(y*CellPx)
	h := cellHash(x, y, 2)
	up, down := lv.Solid(x, y-1) && y > 0, lv.Solid(x, y+1) && y < sim.H-1
	left, right := lv.Solid(x-1, y) && x > 0, lv.Solid(x+1, y) && x < sim.W-1
	switch w {
	case sim.Forest: // густые кусты и деревья
		fillRect(img, X, Y, CellPx, CellPx, th.wallLo)
		for i := 0; i < 4; i++ {
			hh := cellHash(x, y, 50+i)
			fillCircle(img, X+float64(hh%28)+10, Y+float64(hh/28%28)+10, 12+float64(hh%5), th.wall)
		}
		for i := 0; i < 3; i++ {
			hh := cellHash(x, y, 60+i)
			fillCircle(img, X+float64(hh%30)+9, Y+float64(hh/30%26)+8, 5+float64(hh%3), th.wallHi)
		}
		if h%4 == 0 {
			fillCircle(img, X+float64(h%30)+9, Y+float64(h/30%30)+9, 3, rgba(220, 90, 40, 255)) // ягода/лист
		}
		return
	case sim.Temple: // лакированные стены с золотой каймой
		fillRect(img, X, Y, CellPx, CellPx, th.wall)
		if !up {
			fillRect(img, X, Y, CellPx, 6, th.wallHi)
		}
		if !down {
			fillRect(img, X, Y+CellPx-8, CellPx, 8, th.wallLo)
		}
		if !left {
			fillRect(img, X, Y, 4, CellPx, th.wallLo)
		}
		if !right {
			fillRect(img, X+CellPx-4, Y, 4, CellPx, th.wallLo)
		}
		if (x+y)%3 == 0 {
			fillCircle(img, X+CellPx/2, Y+CellPx/2, 4, withAlpha(th.wallHi, 0.6))
		}
		return
	case sim.Front: // мешки с песком и руины
		fillRect(img, X, Y, CellPx, CellPx, th.wallLo)
		for row := 0; row < 3; row++ {
			off := float64((row + x) % 2 * 12)
			for i := -1; i < 2; i++ {
				bx := X + off + float64(i)*24
				x0, x1 := math.Max(X+1, bx), math.Min(X+CellPx-1, bx+22)
				if x1-x0 > 4 {
					roundRect(img, x0, Y+float64(row)*16+1, x1-x0, 14, math.Min(6, (x1-x0)/2), th.wall)
				}
			}
		}
		if !up {
			fillRect(img, X, Y, CellPx, 3, th.wallHi)
		}
		return
	case sim.Space: // обшивка станции с огоньками
		fillRect(img, X, Y, CellPx, CellPx, th.wall)
		strokeRect(img, X+3, Y+3, CellPx-6, CellPx-6, 1.5, th.wallLo)
		if !up {
			fillRect(img, X, Y, CellPx, 5, th.wallHi)
		}
		if !down {
			fillRect(img, X, Y+CellPx-6, CellPx, 6, th.wallLo)
		}
		if h%3 == 0 {
			c := rgba(80, 220, 255, 255)
			if h%2 == 0 {
				c = rgba(255, 120, 80, 255)
			}
			fillCircle(img, X+CellPx-10, Y+10, 2.5, c)
		}
		return
	}
	// Шахта.
	fillRect(img, X, Y, CellPx, CellPx, lerpColor(th.wall, th.wallLo, float64(h%30)/100))
	if !up {
		fillRect(img, X, Y, CellPx, 7, th.wallHi)
	}
	if !down {
		fillRect(img, X, Y+CellPx-8, CellPx, 8, th.wallLo)
	}
	if !left {
		fillRect(img, X, Y, 4, CellPx, rgba(100, 86, 76, 255))
	}
	if !right {
		fillRect(img, X+CellPx-4, Y, 4, CellPx, rgba(62, 52, 46, 255))
	}
	cx, cy := X+float64(h%28)+10, Y+float64(h/28%22)+12
	line(img, cx, cy, cx+8, cy+6, 1.5, rgba(48, 40, 35, 255))
	line(img, cx+8, cy+6, cx+5, cy+14, 1.5, rgba(48, 40, 35, 255))
	for i := 0; i < 3; i++ {
		hh := cellHash(x, y, 40+i)
		fillCircle(img, X+float64(hh%38)+5, Y+float64(hh/38%34)+8, 1.5, rgba(140, 124, 110, 255))
	}
}

func strokeRect(dst *ebiten.Image, x, y, w, h, sw float64, c rgbaT) {
	line(dst, x, y, x+w, y, sw, c)
	line(dst, x+w, y, x+w, y+h, sw, c)
	line(dst, x+w, y+h, x, y+h, sw, c)
	line(dst, x, y+h, x, y, sw, c)
}

// --- предметы --------------------------------------------------------------

// drawItem рисует предмет мира. kind: 0 — обычный, 1 — бонусный (золотой), 2 — проклятый.
func drawItem(dst *ebiten.Image, w sim.World, x, y, s, t float64, kind int) {
	k := CellPx * 0.42 * s
	switch w {
	case sim.Forest:
		body, cap := rgba(196, 130, 60, 255), rgba(110, 70, 36, 255)
		if kind == 1 {
			body, cap = rgba(255, 210, 70, 255), rgba(200, 150, 40, 255)
		}
		fillPoly(dst, ellipse(x, y+k*0.2, k*0.55, k*0.7, 0, 14), body)
		fillPoly(dst, ellipse(x, y-k*0.25, k*0.65, k*0.38, 0, 14), cap)
		line(dst, x, y-k*0.6, x+k*0.15, y-k*0.9, 2.5, cap)
		fillCircle(dst, x-k*0.2, y+k*0.1, k*0.12, rgba(255, 240, 200, 200))
	case sim.Temple:
		c, hi := rgba(150, 210, 255, 255), rgba(235, 250, 255, 255)
		if kind == 2 {
			c, hi = rgba(150, 40, 110, 255), rgba(255, 90, 140, 255)
		}
		wob := math.Sin(t*0.15) * k * 0.15
		fillPoly(dst, []float64{x - k*0.5, y + k*0.1, x + wob, y - k*1.05, x + k*0.5, y + k*0.1}, withAlpha(c, 0.8))
		fillCircle(dst, x, y+k*0.15, k*0.5, withAlpha(c, 0.85))
		fillCircle(dst, x, y+k*0.15, k*0.28, hi)
		if kind == 2 {
			fillCircle(dst, x-k*0.15, y+k*0.1, 2, rgba(20, 0, 10, 255))
			fillCircle(dst, x+k*0.15, y+k*0.1, 2, rgba(20, 0, 10, 255))
		}
	case sim.Front:
		var pts, inner []float64
		for i := 0; i < 6; i++ {
			a := float64(i)/6*2*math.Pi + t*0.02
			pts = append(pts, x+math.Cos(a)*k*0.75, y+math.Sin(a)*k*0.75)
			inner = append(inner, x+math.Cos(a)*k*0.42, y+math.Sin(a)*k*0.42)
		}
		fillPoly(dst, pts, rgba(40, 120, 60, 255))
		fillPoly(dst, inner, rgba(140, 255, 160, 255))
		line(dst, x-k*0.2, y, x+k*0.2, y, 2.5, rgba(30, 90, 40, 255))
		line(dst, x, y-k*0.2, x, y+k*0.2, 2.5, rgba(30, 90, 40, 255))
	case sim.Space:
		var pts []float64
		for i := 0; i < 10; i++ {
			a := -math.Pi/2 + float64(i)/10*2*math.Pi + math.Sin(t*0.03)*0.2
			r := k * 0.85
			if i%2 == 1 {
				r = k * 0.38
			}
			pts = append(pts, x+math.Cos(a)*r, y+math.Sin(a)*r)
		}
		fillPoly(dst, pts, rgba(255, 215, 80, 255))
		fillCircle(dst, x-k*0.12, y-k*0.12, k*0.15, rgba(255, 250, 220, 255))
	default:
		drawCrystal(dst, x, y, s, t)
	}
}

func ellipse(x, y, rx, ry, ang float64, n int) []float64 {
	ca, sa := math.Cos(ang), math.Sin(ang)
	pts := make([]float64, 0, n*2)
	for i := 0; i < n; i++ {
		a := float64(i) / float64(n) * 2 * math.Pi
		u, v := math.Cos(a)*rx, math.Sin(a)*ry
		pts = append(pts, x+u*ca-v*sa, y+u*sa+v*ca)
	}
	return pts
}

// --- герои -----------------------------------------------------------------

// --- враги -----------------------------------------------------------------

func drawEnemy(dst *ebiten.Image, w sim.World, c sim.Chaser, t, ox, oy float64, st *sim.State) {
	if c.Spawn > 0 {
		x, y := px(c.X)+ox, py(c.Y)+oy
		k := 1 - float64(c.Spawn%45)/45
		strokeCircle(dst, x, y, 6+k*18, 3, withAlpha(rgba(255, 60, 60, 255), 1-k))
		fillCircle(dst, x, y, 6, rgba(40, 20, 30, 200))
		return
	}
	if w == sim.Mine {
		drawBeetle(dst, c, t, ox, oy, st)
		return
	}
	x, y := px(c.X)+ox, py(c.Y)+oy
	if c.Kind == sim.ChaserBasic || (w != sim.Temple && w != sim.Space) {
		fillPoly(dst, ellipse(x+2, y+13, 14, 5, 0, 16), rgba(0, 0, 0, 70))
	}
	dx, dy := float64(c.TX)+0.5-c.X, float64(c.TY)+0.5-c.Y
	if math.Hypot(dx, dy) < 0.05 {
		dx, dy = st.P.X-c.X, st.P.Y-c.Y
	}
	ang := math.Atan2(dy, dx)
	ca, sa := math.Cos(ang), math.Sin(ang)
	rot := func(u, v float64) (float64, float64) { return x + u*ca - v*sa, y + u*sa + v*ca }
	r := CellPx * 0.32
	special := c.Kind != sim.ChaserBasic
	th := &themes[w]
	eye := th.eyeBasic
	if special {
		eye = th.eyeSpecial
	}
	stunMix := func(col rgbaT) rgbaT {
		if c.Stun > 0 {
			return lerpColor(col, rgba(130, 130, 130, 255), 0.5)
		}
		return col
	}
	switch w {
	case sim.Forest: // лиса; особая — серый волк-зигзаг
		body, tip := rgba(220, 110, 40, 255), rgba(250, 245, 235, 255)
		if special {
			body = rgba(130, 130, 140, 255)
		}
		body = stunMix(body)
		tx, ty := rot(-r*1.3, math.Sin(t*0.3)*4)
		fillPoly(dst, ellipse((x+tx)/2, (y+ty)/2, r*0.7, r*0.28, ang, 12), body)
		fillCircle(dst, tx, ty, r*0.25, tip)
		fillPoly(dst, ellipse(x, y, r*0.95, r*0.6, ang, 16), body)
		hx, hy := rot(r*0.8, 0)
		fillCircle(dst, hx, hy, r*0.5, body)
		e1x, e1y := rot(r*0.9, -r*0.45)
		e2x, e2y := rot(r*0.9, r*0.45)
		fillPoly(dst, []float64{hx, hy, e1x, e1y, hx + (e1x-hx)*0.3 + ca*6, hy + (e1y-hy)*0.3 + sa*6}, body)
		fillPoly(dst, []float64{hx, hy, e2x, e2y, hx + (e2x-hx)*0.3 + ca*6, hy + (e2y-hy)*0.3 + sa*6}, body)
		nx, ny := rot(r*1.25, 0)
		fillCircle(dst, nx, ny, 2.5, rgba(30, 20, 15, 255))
		i1x, i1y := rot(r*0.95, -r*0.2)
		i2x, i2y := rot(r*0.95, r*0.2)
		fillCircle(dst, i1x, i1y, 2.5, eye)
		fillCircle(dst, i2x, i2y, 2.5, eye)
	case sim.Temple: // красный óни; особый — бледный призрак
		if special {
			a := 0.75
			if c.BlinkWarn > 0 {
				a = 0.3 + 0.3*math.Sin(t*0.6)
			}
			col := stunMix(rgba(230, 225, 255, 255))
			fillCircle(dst, x, y-r*0.2, r*0.85, withAlpha(col, a))
			var pts []float64
			pts = append(pts, x-r*0.85, y-r*0.2)
			for i := 0; i <= 6; i++ {
				fx := x - r*0.85 + float64(i)*r*0.283
				fy := y + r*0.75 + math.Sin(t*0.3+float64(i))*3
				if i%2 == 1 {
					fy -= 6
				}
				pts = append(pts, fx, fy)
			}
			pts = append(pts, x+r*0.85, y-r*0.2)
			fillPoly(dst, pts, withAlpha(col, a))
			fillCircle(dst, x-r*0.3, y-r*0.3, 3.5, withAlpha(rgba(60, 20, 90, 255), a+0.2))
			fillCircle(dst, x+r*0.3, y-r*0.3, 3.5, withAlpha(rgba(60, 20, 90, 255), a+0.2))
			break
		}
		body := stunMix(rgba(190, 50, 50, 255))
		fillCircle(dst, x, y, r+2, rgba(40, 10, 10, 255))
		fillCircle(dst, x, y, r, body)
		h1x, h1y := rot(r*0.3, -r*0.7)
		h2x, h2y := rot(r*0.3, r*0.7)
		b1x, b1y := rot(r*0.1, -r*0.45)
		b2x, b2y := rot(r*0.1, r*0.45)
		fillPoly(dst, []float64{b1x - 4, b1y, b1x + 4, b1y, h1x, h1y}, rgba(240, 230, 200, 255))
		fillPoly(dst, []float64{b2x - 4, b2y, b2x + 4, b2y, h2x, h2y}, rgba(240, 230, 200, 255))
		e1x, e1y := rot(r*0.5, -r*0.25)
		e2x, e2y := rot(r*0.5, r*0.25)
		fillCircle(dst, e1x, e1y, 3, eye)
		fillCircle(dst, e2x, e2y, 3, eye)
	case sim.Front: // дрон с четырьмя винтами
		body := stunMix(rgba(90, 96, 100, 255))
		if special {
			body = stunMix(rgba(200, 170, 40, 255))
		}
		for i := 0; i < 4; i++ {
			a := ang + math.Pi/4 + float64(i)*math.Pi/2
			rx, ry := x+math.Cos(a)*r*0.9, y+math.Sin(a)*r*0.9
			line(dst, x, y, rx, ry, 3, rgba(40, 40, 44, 255))
			strokeCircle(dst, rx, ry, r*0.38, 1.5, withAlpha(rgba(200, 200, 210, 255), 0.5))
			pa := t * 0.9
			line(dst, rx-math.Cos(pa)*r*0.35, ry-math.Sin(pa)*r*0.35, rx+math.Cos(pa)*r*0.35, ry+math.Sin(pa)*r*0.35, 2, rgba(220, 220, 230, 255))
		}
		fillPoly(dst, ellipse(x, y, r*0.55, r*0.55, ang+math.Pi/4, 4), body)
		lx, ly := rot(r*0.25, 0)
		fillCircle(dst, lx, ly, 3.5, eye)
	case sim.Space: // зелёный пришелец; особый — фиолетовая медуза-телепорт
		if special {
			a := 0.8
			if c.BlinkWarn > 0 {
				a = 0.3 + 0.3*math.Sin(t*0.6)
			}
			col := stunMix(rgba(190, 100, 240, 255))
			fillPoly(dst, ellipse(x, y-r*0.2, r*0.9, r*0.65, 0, 16), withAlpha(col, a))
			for i := 0; i < 4; i++ {
				tx := x - r*0.6 + float64(i)*r*0.4
				line(dst, tx, y+r*0.2, tx+math.Sin(t*0.2+float64(i))*4, y+r*0.95, 2.5, withAlpha(col, a))
			}
			fillCircle(dst, x-r*0.3, y-r*0.25, 3, withAlpha(eye, a+0.2))
			fillCircle(dst, x+r*0.3, y-r*0.25, 3, withAlpha(eye, a+0.2))
			break
		}
		body := stunMix(rgba(90, 200, 90, 255))
		wob := 1 + 0.08*math.Sin(t*0.25)
		fillPoly(dst, ellipse(x, y, r*wob, r/wob, ang, 16), body)
		ax, ay := rot(-r*0.2, 0)
		line(dst, ax, ay, ax-ca*r*0.6, ay-sa*r*0.6-r*0.4, 2, body)
		fillCircle(dst, ax-ca*r*0.6, ay-sa*r*0.6-r*0.4, 3, eye)
		ex, ey := rot(r*0.4, 0)
		fillCircle(dst, ex, ey, r*0.32, rgba(240, 255, 240, 255))
		fillCircle(dst, ex+ca*2, ey+sa*2, r*0.15, rgba(10, 30, 10, 255))
	}
	if c.Stun > 0 {
		for i := 0; i < 3; i++ {
			a := t*0.15 + float64(i)*2.09
			fillCircle(dst, x+math.Cos(a)*14, y-18+math.Sin(a)*4, 2.5, rgba(255, 240, 120, 255))
		}
	}
}

// --- катящиеся объекты -----------------------------------------------------

func drawRolling(dst *ebiten.Image, w sim.World, r sim.Rock, x, y float64) {
	rad := sim.RockRadius * CellPx
	fillCircle(dst, x+3, y+5, rad, rgba(0, 0, 0, 80))
	switch w {
	case sim.Forest: // бревно поперёк движения
		ang := math.Atan2(float64(r.DY), float64(r.DX)) + math.Pi/2
		ca, sa := math.Cos(ang), math.Sin(ang)
		L := CellPx * 0.48
		pts := []float64{
			x - ca*L - sa*rad*0.6, y - sa*L + ca*rad*0.6,
			x + ca*L - sa*rad*0.6, y + sa*L + ca*rad*0.6,
			x + ca*L + sa*rad*0.6, y + sa*L - ca*rad*0.6,
			x - ca*L + sa*rad*0.6, y - sa*L - ca*rad*0.6,
		}
		fillPoly(dst, pts, rgba(120, 76, 40, 255))
		stripe := math.Mod(r.Angle*rad, 10)
		for k := -rad * 0.5; k <= rad*0.5; k += 10 {
			s := k + stripe - 5
			line(dst, x-ca*L-sa*s, y-sa*L+ca*s, x+ca*L-sa*s, y+sa*L+ca*s, 1.5, rgba(90, 56, 28, 255))
		}
		fillPoly(dst, ellipse(x+ca*L, y+sa*L, rad*0.32, rad*0.6, ang, 10), rgba(200, 160, 100, 255))
		fillPoly(dst, ellipse(x-ca*L, y-sa*L, rad*0.32, rad*0.6, ang, 10), rgba(200, 160, 100, 255))
	case sim.Front: // катящаяся мина
		fillCircle(dst, x, y, rad+2, rgba(20, 20, 20, 255))
		fillCircle(dst, x, y, rad, rgba(60, 64, 60, 255))
		for i := 0; i < 6; i++ {
			a := r.Angle + float64(i)*math.Pi/3
			line(dst, x+math.Cos(a)*rad*0.8, y+math.Sin(a)*rad*0.8, x+math.Cos(a)*rad*1.3, y+math.Sin(a)*rad*1.3, 3, rgba(40, 40, 40, 255))
		}
		fillCircle(dst, x, y, rad*0.25, rgba(255, 60, 40, 255))
	case sim.Space: // астероид неправильной формы
		var pts []float64
		for i := 0; i < 9; i++ {
			a := r.Angle + float64(i)/9*2*math.Pi
			k := 0.8 + 0.25*float64(cellHash(i, 0, 7)%10)/10
			pts = append(pts, x+math.Cos(a)*rad*k, y+math.Sin(a)*rad*k)
		}
		fillPoly(dst, pts, rgba(110, 100, 120, 255))
		fillCircle(dst, x+math.Cos(r.Angle)*rad*0.3, y+math.Sin(r.Angle)*rad*0.3, rad*0.2, rgba(80, 72, 90, 255))
	default: // валун
		fillCircle(dst, x, y, rad+2, rgba(40, 34, 30, 255))
		fillCircle(dst, x, y, rad, rgba(138, 128, 120, 255))
		for i := 0; i < 3; i++ {
			a := r.Angle + float64(i)*2.1
			fillCircle(dst, x+math.Cos(a)*rad*0.5, y+math.Sin(a)*rad*0.5, rad*0.22, rgba(100, 92, 86, 255))
		}
		fillCircle(dst, x-rad*0.35, y-rad*0.4, rad*0.25, rgba(175, 166, 158, 255))
	}
}

// --- шипы ------------------------------------------------------------------

func drawTrap(dst *ebiten.Image, w sim.World, X, Y float64, ph int) {
	up, warn := ph >= sim.SpikeUpFrom, ph >= sim.SpikeWarnFrom && ph < sim.SpikeUpFrom
	if w == sim.Front { // мина: мигает перед взрывом
		fillCircle(dst, X+CellPx/2, Y+CellPx/2, 14, rgba(50, 52, 46, 255))
		strokeCircle(dst, X+CellPx/2, Y+CellPx/2, 14, 2, rgba(30, 30, 28, 255))
		c := rgba(80, 30, 30, 255)
		if warn && (ph/5)%2 == 0 {
			c = rgba(255, 180, 60, 255)
		}
		if up {
			c = rgba(255, 50, 40, 255)
			fillCircle(dst, X+CellPx/2, Y+CellPx/2, 22, rgba(255, 120, 40, 90))
		}
		fillCircle(dst, X+CellPx/2, Y+CellPx/2, 5, c)
		return
	}
	base, tipCol := rgba(28, 22, 20, 255), rgba(170, 165, 160, 255)
	switch w {
	case sim.Forest:
		base, tipCol = rgba(50, 60, 30, 255), rgba(120, 140, 70, 255)
	case sim.Temple:
		base, tipCol = rgba(24, 22, 32, 255), rgba(200, 205, 220, 255)
	}
	roundRect(dst, X+5, Y+5, CellPx-10, CellPx-10, 6, base)
	h := 0.12
	switch {
	case up:
		h = 1
		tipCol = lerpColor(tipCol, rgba(255, 255, 255, 255), 0.4)
	case warn:
		h = 0.25 + 0.1*math.Sin(float64(ph)*1.3)
		tipCol = rgba(240, 150, 70, 255)
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			cx, cy := X+15+float64(i)*18, Y+26+float64(j)*14
			fillPoly(dst, []float64{cx - 6, cy, cx + 6, cy, cx, cy - 4 - 18*h}, tipCol)
		}
	}
	if up {
		fillRect(dst, X+5, Y+5, CellPx-10, CellPx-10, rgba(120, 20, 20, 50))
	}
}

// --- выход -----------------------------------------------------------------

func drawExitGlyph(dst *ebiten.Image, w sim.World, X, Y, t float64) {
	cx, cy := X+CellPx/2, Y+CellPx/2+math.Sin(t*0.15)*2
	white := rgba(240, 255, 245, 255)
	switch w {
	case sim.Forest: // дупло
		fillPoly(dst, ellipse(cx, cy, 11, 14, 0, 14), rgba(50, 30, 15, 255))
		strokeCircle(dst, cx, cy, 14, 2, white)
	case sim.Temple: // ворота-тории
		fillRect(dst, cx-12, cy-12, 24, 4, rgba(230, 60, 50, 255))
		fillRect(dst, cx-9, cy-6, 18, 3, rgba(230, 60, 50, 255))
		fillRect(dst, cx-8, cy-12, 4, 24, rgba(230, 60, 50, 255))
		fillRect(dst, cx+4, cy-12, 4, 24, rgba(230, 60, 50, 255))
	case sim.Front: // вертолётная площадка
		strokeCircle(dst, cx, cy, 15, 2, white)
		drawText(dst, "H", 22, true, cx, cy-14, Center, white)
	case sim.Space: // ракета
		fillPoly(dst, []float64{cx, cy - 16, cx + 7, cy + 4, cx - 7, cy + 4}, white)
		fillRect(dst, cx-7, cy+4, 14, 8, white)
		fillCircle(dst, cx, cy-2, 3, rgba(80, 160, 255, 255))
		fillPoly(dst, []float64{cx - 5, cy + 12, cx + 5, cy + 12, cx, cy + 20 + math.Sin(t*0.6)*3}, rgba(255, 150, 50, 255))
	default: // стрелка-подъёмник
		fillPoly(dst, []float64{cx, cy - 12, cx + 10, cy + 2, cx - 10, cy + 2}, white)
		fillRect(dst, cx-4, cy+2, 8, 9, white)
	}
}

// --- лучи/обвалы -----------------------------------------------------------

func drawBeamActive(dst *ebiten.Image, w sim.World, c sim.CaveIn, x, y, bw, bh, t float64) {
	a := float64(c.Active) / sim.CaveHitTicks
	if w == sim.Mine {
		fillRect(dst, x, y, bw, bh, withAlpha(rgba(120, 100, 80, 255), 0.75*a))
		n := int(math.Max(bw, bh) / 14)
		for i := 0; i < n; i++ {
			hh := cellHash(i, c.Line, c.Active)
			var bx, by float64
			if c.Horizontal {
				bx, by = x+float64(i)*14, y+float64(hh%uint64(CellPx))
			} else {
				bx, by = x+float64(hh%uint64(CellPx)), y+float64(i)*14
			}
			fillRect(dst, bx, by, 6+float64(hh%6), 5+float64(hh%5), withAlpha(rgba(70, 58, 48, 255), a))
		}
		return
	}
	col := themes[w].beam
	fillRect(dst, x, y, bw, bh, withAlpha(col, 0.35*a))
	if c.Horizontal {
		fillRect(dst, x, y+bh*0.3, bw, bh*0.4, withAlpha(col, 0.8*a))
		fillRect(dst, x, y+bh*0.42, bw, bh*0.16, withAlpha(rgba(255, 255, 255, 255), a))
	} else {
		fillRect(dst, x+bw*0.3, y, bw*0.4, bh, withAlpha(col, 0.8*a))
		fillRect(dst, x+bw*0.42, y, bw*0.16, bh, withAlpha(rgba(255, 255, 255, 255), a))
	}
}

// --- чёрная дыра -----------------------------------------------------------

func drawHole(dst *ebiten.Image, x, y, t float64) {
	strokeCircle(dst, x, y, sim.HoleRadius*CellPx, 1.5, rgba(140, 80, 255, 40))
	for i := 0; i < 3; i++ {
		rr := CellPx * (0.9 + 0.35*float64(i))
		a := t*0.05*float64(3-i) + float64(i)
		var pts []float64
		for k := 0; k < 12; k++ {
			ang := a + float64(k)/12*math.Pi*1.3
			pts = append(pts, x+math.Cos(ang)*rr, y+math.Sin(ang)*rr*0.75)
		}
		for k := 0; k+3 < len(pts); k += 2 {
			line(dst, pts[k], pts[k+1], pts[k+2], pts[k+3], 2.5, withAlpha(rgba(190, 120, 255, 255), 0.5-0.12*float64(i)))
		}
	}
	fillCircle(dst, x, y, CellPx*0.55, rgba(255, 150, 60, 200))
	fillCircle(dst, x, y, CellPx*0.45, rgba(0, 0, 0, 255))
}

// --- значки способностей для HUD ------------------------------------------

func abilityColor(ab sim.Ability) rgbaT {
	switch ab {
	case sim.AbMagnet:
		return rgba(255, 110, 110, 255)
	case sim.AbTimeStop:
		return rgba(190, 150, 255, 255)
	case sim.AbBlink:
		return rgba(120, 255, 160, 255)
	case sim.AbPulse:
		return rgba(255, 230, 110, 255)
	}
	return colAccent
}
