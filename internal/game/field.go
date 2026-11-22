package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Отрисовка мира «Кристальная шахта». Вся графика процедурная (векторные
// фигуры и градиенты), картинок-спрайтов нет — как и в оригинале.

const (
	CellPx = 48
	FieldX = (ScreenW - sim.W*CellPx) / 2
	FieldY = 88
	FieldW = sim.W * CellPx
	FieldH = sim.H * CellPx
)

func px(x float64) float64 { return FieldX + x*CellPx }
func py(y float64) float64 { return FieldY + y*CellPx }

// Цвета шахты.
var (
	mineFloorA  = rgba(46, 38, 34, 255)
	mineFloorB  = rgba(38, 31, 29, 255)
	mineWall    = rgba(84, 72, 64, 255)
	mineWallHi  = rgba(122, 106, 92, 255)
	mineWallLo  = rgba(52, 43, 38, 255)
	mineMud     = rgba(78, 52, 30, 255)
	mineMudHi   = rgba(110, 78, 48, 255)
	crystalBody = rgba(90, 220, 255, 255)
	crystalHi   = rgba(210, 250, 255, 255)
	crystalLo   = rgba(40, 130, 200, 255)
	exitGlow    = rgba(120, 255, 190, 255)
)

func cellHash(x, y, k int) uint64 {
	r := sim.NewRNG(uint64(x*73856093) ^ uint64(y*19349663) ^ uint64(k*83492791))
	return r.Next()
}

// fieldView — отрисовщик уровня; статичная часть (пол, стены, грязь)
// рисуется один раз в отдельную картинку.
type fieldView struct {
	lv     *sim.Level
	static *ebiten.Image
}

func newFieldView(lv *sim.Level) *fieldView {
	f := &fieldView{lv: lv, static: ebiten.NewImage(FieldW, FieldH)}
	f.renderStatic()
	return f
}

func (f *fieldView) renderStatic() {
	img := f.static
	lv := f.lv
	img.Fill(mineFloorB)
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			X, Y := float64(x*CellPx), float64(y*CellPx)
			h := cellHash(x, y, 1)
			switch lv.Grid[y][x] {
			case sim.Floor:
				c := lerpColor(mineFloorA, mineFloorB, float64(h%100)/100)
				fillRect(img, X, Y, CellPx, CellPx, c)
				for i := 0; i < 3; i++ {
					hh := cellHash(x, y, 10+i)
					fillCircle(img, X+float64(hh%40)+4, Y+float64(hh/40%40)+4, 1.2+float64(hh%3)*0.6, rgba(60, 51, 46, 255))
				}
			case sim.Mud:
				fillRect(img, X, Y, CellPx, CellPx, mineMud)
				for i := 0; i < 4; i++ {
					hh := cellHash(x, y, 20+i)
					fillCircle(img, X+float64(hh%36)+6, Y+float64(hh/36%36)+6, 3+float64(hh%4), rgba(64, 42, 24, 255))
				}
				hh := cellHash(x, y, 30)
				line(img, X+8+float64(hh%10), Y+12, X+20+float64(hh%10), Y+10, 2, mineMudHi)
			}
		}
	}
	// Стены рисуем поверх, с объёмом: свет сверху, тень снизу.
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] != sim.Wall {
				continue
			}
			X, Y := float64(x*CellPx), float64(y*CellPx)
			if !lv.Solid(x, y+1) {
				fillRect(img, X, Y+CellPx, CellPx, 10, rgba(0, 0, 0, 90))
			}
		}
	}
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] != sim.Wall {
				continue
			}
			X, Y := float64(x*CellPx), float64(y*CellPx)
			h := cellHash(x, y, 2)
			fillRect(img, X, Y, CellPx, CellPx, lerpColor(mineWall, mineWallLo, float64(h%30)/100))
			up, down := lv.Solid(x, y-1) && y > 0, lv.Solid(x, y+1) && y < sim.H-1
			left, right := lv.Solid(x-1, y) && x > 0, lv.Solid(x+1, y) && x < sim.W-1
			if !up {
				fillRect(img, X, Y, CellPx, 7, mineWallHi)
			}
			if !down {
				fillRect(img, X, Y+CellPx-8, CellPx, 8, mineWallLo)
			}
			if !left {
				fillRect(img, X, Y, 4, CellPx, rgba(100, 86, 76, 255))
			}
			if !right {
				fillRect(img, X+CellPx-4, Y, 4, CellPx, rgba(62, 52, 46, 255))
			}
			// Трещины и вкрапления породы.
			cx, cy := X+float64(h%28)+10, Y+float64(h/28%22)+12
			line(img, cx, cy, cx+8, cy+6, 1.5, rgba(48, 40, 35, 255))
			line(img, cx+8, cy+6, cx+5, cy+14, 1.5, rgba(48, 40, 35, 255))
			for i := 0; i < 3; i++ {
				hh := cellHash(x, y, 40+i)
				fillCircle(img, X+float64(hh%38)+5, Y+float64(hh/38%34)+8, 1.5, rgba(140, 124, 110, 255))
			}
		}
	}
}

// drawWorld рисует поле со всеми объектами. ghost может быть nil.
func (f *fieldView) drawWorld(dst *ebiten.Image, st, ghost *sim.State, frame int, fx *fxSystem, trail [][2]float64, ghostName string) {
	ox, oy := fx.offset()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(FieldX+ox, FieldY+oy)
	dst.DrawImage(f.static, op)

	field := dst.SubImage(image.Rect(FieldX, FieldY, FieldX+FieldW, FieldY+FieldH)).(*ebiten.Image)
	t := float64(frame)

	// Шипы.
	for _, sp := range f.lv.Spikes {
		f.drawSpikes(field, st, sp, ox, oy)
	}
	// Выход.
	f.drawExit(field, st, t, ox, oy)
	// Предупреждения обвалов.
	for _, c := range st.Caves {
		f.drawCave(field, c, t, ox, oy)
	}
	// Кристаллы и аптечка.
	for i, it := range st.Items {
		if it.Taken {
			continue
		}
		x, y := px(float64(it.P.X)+0.5)+ox, py(float64(it.P.Y)+0.5)+oy+math.Sin(t*0.07+float64(i))*2.5
		drawCrystal(field, x, y, 1, t+float64(i)*10)
	}
	if f.lv.HasHeart && !st.HeartTaken {
		x, y := px(float64(f.lv.Heart.X)+0.5)+ox, py(float64(f.lv.Heart.Y)+0.5)+oy
		drawHeart(field, x, y+math.Sin(t*0.09)*2, 1+0.08*math.Sin(t*0.15), rgba(255, 80, 100, 255))
	}
	// Валуны.
	for _, r := range st.Rocks {
		f.drawRock(field, r, t, ox, oy)
	}
	// Жуки.
	for _, c := range st.Chasers {
		drawBeetle(field, c, t, ox, oy, st)
	}
	// Призрак.
	if ghost != nil {
		gx, gy := px(ghost.P.X)+ox, py(ghost.P.Y)+oy
		drawMiner(field, ghost.P, gx, gy, t, 0.38, true)
		if ghostName != "" {
			drawText(field, ghostName, 13, false, gx, gy-36, Center, rgba(200, 230, 255, 160))
		}
	}
	// След рывка.
	for i, p := range trail {
		a := float64(i+1) / float64(len(trail)+1) * 0.35
		fillCircle(field, px(p[0])+ox, py(p[1])+oy, 0.3*CellPx, withAlpha(rgba(255, 190, 90, 255), a))
	}
	// Игрок (мигает после удара).
	p := st.P
	if p.Invuln == 0 || (frame/4)%2 == 0 {
		drawMiner(field, p, px(p.X)+ox, py(p.Y)+oy, t, 1, false)
	}
	fx.draw(field, 0, 0)

	// Темнота шахты: светло только вокруг фонаря шахтёра.
	lx, ly := px(p.X)+ox+p.FX*18, py(p.Y)+oy+p.FY*18
	dop := &ebiten.DrawImageOptions{}
	scale := 2800.0 / 256
	dop.GeoM.Translate(-128, -128)
	dop.GeoM.Scale(scale, scale)
	dop.GeoM.Translate(lx, ly)
	dop.ColorScale.ScaleAlpha(0.62)
	dop.Filter = ebiten.FilterLinear
	field.DrawImage(darkImg, dop)

	// Светящиеся объекты видны и в темноте.
	for i, it := range st.Items {
		if it.Taken {
			continue
		}
		x, y := px(float64(it.P.X)+0.5)+ox, py(float64(it.P.Y)+0.5)+oy
		glow(field, x, y, 34, crystalBody, 0.45+0.15*math.Sin(t*0.1+float64(i)))
	}
	if st.ExitOpen {
		glow(field, px(float64(f.lv.Exit.X)+0.5)+ox, py(float64(f.lv.Exit.Y)+0.5)+oy, 80, exitGlow, 0.6+0.2*math.Sin(t*0.12))
	}
	for _, c := range st.Chasers {
		if c.Spawn == 0 {
			eye := rgba(255, 60, 60, 255)
			if c.Kind == sim.ChaserCunning {
				eye = rgba(120, 255, 90, 255)
			}
			glow(field, px(c.X)+ox, py(c.Y)+oy, 22, eye, 0.5)
		}
	}
	glow(field, lx, ly, 70, rgba(255, 220, 150, 255), 0.18)
	if p.ShieldT > 0 {
		glow(field, px(p.X)+ox, py(p.Y)+oy, 50, colAccent, 0.35)
	}
	for _, c := range st.Caves {
		if c.Warn == 0 {
			f.drawCaveFall(field, c, t, ox, oy)
		}
	}
	fx.drawTexts(field, 0, 0)
	if fx.flash > 0.01 {
		fillRect(field, FieldX, FieldY, FieldW, FieldH, withAlpha(rgba(200, 30, 30, 255), fx.flash*0.35))
	}
	roundRectStroke(dst, FieldX-3, FieldY-3, FieldW+6, FieldH+6, 6, 3, rgba(70, 60, 54, 255))
}

func drawCrystal(dst *ebiten.Image, x, y, s, t float64) {
	k := CellPx * 0.42 * s
	pts := []float64{
		x, y - k,
		x + k*0.62, y - k*0.3,
		x + k*0.48, y + k*0.6,
		x, y + k*0.95,
		x - k*0.48, y + k*0.6,
		x - k*0.62, y - k*0.3,
	}
	fillPoly(dst, pts, crystalLo)
	fillPoly(dst, []float64{x, y - k, x + k*0.62, y - k*0.3, x, y + k*0.95, x - k*0.2, y - k*0.1}, crystalBody)
	fillPoly(dst, []float64{x, y - k, x - k*0.2, y - k*0.1, x - k*0.62, y - k*0.3}, crystalHi)
	sp := math.Max(0, math.Sin(t*0.05))
	if sp > 0.9 {
		a := (sp - 0.9) * 10
		line(dst, x-k*0.5, y-k*0.5, x-k*0.1, y-k*0.5, 2, withAlpha(crystalHi, a))
		line(dst, x-k*0.3, y-k*0.7, x-k*0.3, y-k*0.3, 2, withAlpha(crystalHi, a))
	}
}

func drawHeart(dst *ebiten.Image, x, y, s float64, c color.RGBA) {
	r := 7 * s
	fillCircle(dst, x-r*0.8, y-r*0.3, r, c)
	fillCircle(dst, x+r*0.8, y-r*0.3, r, c)
	fillPoly(dst, []float64{x - r*1.75, y, x + r*1.75, y, x, y + r*2}, c)
	fillCircle(dst, x-r*1.0, y-r*0.6, r*0.3, rgba(255, 200, 210, 255))
}

// drawMiner рисует шахтёра. alpha<1 и ghost — для призрака.
func drawMiner(dst *ebiten.Image, p sim.Player, x, y, t, alpha float64, ghost bool) {
	r := CellPx * 0.32
	bob := 0.0
	if p.Moving {
		bob = math.Sin(p.WalkDistance*9) * 1.8
	}
	y += bob
	body := rgba(232, 138, 42, 255)
	helmet := rgba(250, 210, 60, 255)
	outline := rgba(30, 20, 15, 255)
	if ghost {
		body, helmet, outline = rgba(150, 200, 255, 255), rgba(210, 235, 255, 255), rgba(80, 120, 180, 255)
	}
	fillCircle(dst, x+2, y+4, r, withAlpha(rgba(0, 0, 0, 255), 0.3*alpha))
	fillCircle(dst, x, y, r+2, withAlpha(outline, alpha))
	fillCircle(dst, x, y, r, withAlpha(body, alpha))
	// Ремни комбинезона.
	line(dst, x-r*0.5, y-r*0.2, x-r*0.4, y+r*0.8, 3, withAlpha(rgba(60, 80, 140, 255), alpha))
	line(dst, x+r*0.5, y-r*0.2, x+r*0.4, y+r*0.8, 3, withAlpha(rgba(60, 80, 140, 255), alpha))
	// Каска сдвинута в сторону взгляда.
	hx, hy := x+p.FX*r*0.25, y+p.FY*r*0.25-r*0.25
	fillCircle(dst, hx, hy, r*0.72, withAlpha(outline, alpha))
	fillCircle(dst, hx, hy, r*0.65, withAlpha(helmet, alpha))
	fillCircle(dst, hx-r*0.2, hy-r*0.2, r*0.2, withAlpha(rgba(255, 245, 190, 255), alpha))
	// Фонарь.
	lx, ly := hx+p.FX*r*0.55, hy+p.FY*r*0.55
	fillCircle(dst, lx, ly, r*0.24, withAlpha(rgba(255, 255, 230, 255), alpha))
	if p.ShieldT > 0 && !ghost {
		a := 0.6
		if p.ShieldT < 60 && (p.ShieldT/6)%2 == 0 {
			a = 0.2
		}
		strokeCircle(dst, x, y, r+9+math.Sin(t*0.3)*1.5, 3, withAlpha(colAccent, a))
	}
}

func drawBeetle(dst *ebiten.Image, c sim.Chaser, t, ox, oy float64, st *sim.State) {
	x, y := px(c.X)+ox, py(c.Y)+oy
	if c.Spawn > 0 {
		k := 1 - float64(c.Spawn%45)/45
		strokeCircle(dst, x, y, 6+k*18, 3, withAlpha(rgba(255, 60, 60, 255), 1-k))
		fillCircle(dst, x, y, 6, rgba(40, 20, 30, 200))
		return
	}
	body := rgba(70, 40, 96, 255)
	shell := rgba(110, 70, 150, 255)
	eye := rgba(255, 70, 70, 255)
	if c.Kind == sim.ChaserCunning {
		body, shell, eye = rgba(36, 70, 48, 255), rgba(60, 120, 80, 255), rgba(150, 255, 110, 255)
	}
	if c.Stun > 0 {
		body, shell = lerpColor(body, rgba(120, 120, 120, 255), 0.5), lerpColor(shell, rgba(150, 150, 150, 255), 0.5)
	}
	// Направление — к следующей клетке пути.
	dx, dy := float64(c.TX)+0.5-c.X, float64(c.TY)+0.5-c.Y
	if math.Hypot(dx, dy) < 0.05 {
		dx, dy = st.P.X-c.X, st.P.Y-c.Y
	}
	ang := math.Atan2(dy, dx)
	ca, sa := math.Cos(ang), math.Sin(ang)
	rot := func(u, v float64) (float64, float64) { return x + u*ca - v*sa, y + u*sa + v*ca }
	r := CellPx * 0.32
	// Лапки.
	for i := -1; i <= 1; i++ {
		ph := math.Sin(t*0.4+float64(i)*2) * 4
		for _, side := range []float64{-1, 1} {
			x0, y0 := rot(float64(i)*r*0.5, side*r*0.5)
			x1, y1 := rot(float64(i)*r*0.5+ph, side*r*1.15)
			line(dst, x0, y0, x1, y1, 2.5, rgba(30, 18, 40, 255))
		}
	}
	var pts []float64
	for i := 0; i < 16; i++ {
		a := float64(i) / 16 * 2 * math.Pi
		u, v := rot(math.Cos(a)*r*1.05, math.Sin(a)*r*0.78)
		pts = append(pts, u, v)
	}
	fillPoly(dst, pts, body)
	sx0, sy0 := rot(-r*0.9, 0)
	sx1, sy1 := rot(r*0.4, 0)
	line(dst, sx0, sy0, sx1, sy1, 2, shell)
	hx, hy := rot(r*0.75, 0)
	fillCircle(dst, hx, hy, r*0.42, rgba(30, 18, 40, 255))
	e1x, e1y := rot(r*0.95, -r*0.2)
	e2x, e2y := rot(r*0.95, r*0.2)
	fillCircle(dst, e1x, e1y, 3, eye)
	fillCircle(dst, e2x, e2y, 3, eye)
	if c.Stun > 0 {
		for i := 0; i < 3; i++ {
			a := t*0.15 + float64(i)*2.09
			fillCircle(dst, x+math.Cos(a)*14, y-18+math.Sin(a)*4, 2.5, rgba(255, 240, 120, 255))
		}
	}
}

func (f *fieldView) drawSpikes(dst *ebiten.Image, st *sim.State, sp sim.SpikeTrap, ox, oy float64) {
	X, Y := px(float64(sp.P.X))+ox, py(float64(sp.P.Y))+oy
	ph := st.SpikePhase(sp)
	roundRect(dst, X+5, Y+5, CellPx-10, CellPx-10, 6, rgba(28, 22, 20, 255))
	var h float64
	tip := rgba(170, 165, 160, 255)
	switch {
	case ph >= sim.SpikeUpFrom:
		h = 1
		tip = rgba(230, 225, 220, 255)
	case ph >= sim.SpikeWarnFrom:
		h = 0.25 + 0.1*math.Sin(float64(ph)*1.3)
		tip = rgba(240, 150, 70, 255)
	default:
		h = 0.12
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			cx, cy := X+15+float64(i)*18, Y+26+float64(j)*14
			fillPoly(dst, []float64{cx - 6, cy, cx + 6, cy, cx, cy - 4 - 18*h}, tip)
		}
	}
	if ph >= sim.SpikeUpFrom {
		fillRect(dst, X+5, Y+5, CellPx-10, CellPx-10, rgba(120, 20, 20, 50))
	}
}

func (f *fieldView) drawExit(dst *ebiten.Image, st *sim.State, t, ox, oy float64) {
	e := f.lv.Exit
	X, Y := px(float64(e.X))+ox, py(float64(e.Y))+oy
	roundRect(dst, X+3, Y+3, CellPx-6, CellPx-6, 8, rgba(20, 16, 14, 255))
	if !st.ExitOpen {
		roundRectStroke(dst, X+3, Y+3, CellPx-6, CellPx-6, 8, 3, rgba(110, 96, 84, 255))
		for i := 0; i < 4; i++ {
			bx := X + 10 + float64(i)*9.5
			line(dst, bx, Y+8, bx, Y+CellPx-8, 3, rgba(120, 110, 100, 255))
		}
		// Замок.
		fillRect(dst, X+CellPx/2-7, Y+CellPx/2-2, 14, 12, colGold)
		strokeCircle(dst, X+CellPx/2, Y+CellPx/2-3, 5, 2.5, colGold)
		return
	}
	k := 0.5 + 0.5*math.Sin(t*0.12)
	roundRect(dst, X+6, Y+6, CellPx-12, CellPx-12, 6, lerpColor(rgba(40, 140, 100, 255), exitGlow, k*0.5))
	roundRectStroke(dst, X+3, Y+3, CellPx-6, CellPx-6, 8, 3, exitGlow)
	ay := Y + CellPx/2 + math.Sin(t*0.15)*3
	fillPoly(dst, []float64{X + CellPx/2, ay - 12, X + CellPx/2 + 10, ay + 2, X + CellPx/2 - 10, ay + 2}, rgba(230, 255, 240, 255))
	fillRect(dst, X+CellPx/2-4, ay+2, 8, 9, rgba(230, 255, 240, 255))
}

func (f *fieldView) drawRock(dst *ebiten.Image, r sim.Rock, t, ox, oy float64) {
	if r.Warn > 0 {
		X, Y := px(float64(r.Spawn.X)+0.5)+ox, py(float64(r.Spawn.Y)+0.5)+oy
		if (r.Warn/6)%2 == 0 {
			dx, dy := float64(r.DX), float64(r.DY)
			c := rgba(255, 170, 60, 220)
			// Стрелка в сторону движения валуна.
			fillPoly(dst, []float64{
				X + dx*14, Y + dy*14,
				X - dx*6 - dy*12, Y - dy*6 - dx*12,
				X - dx*6 + dy*12, Y - dy*6 + dx*12,
			}, c)
			fillPoly(dst, []float64{
				X - dx*4, Y - dy*4,
				X - dx*20 - dy*10, Y - dy*20 - dx*10,
				X - dx*20 + dy*10, Y - dy*20 + dx*10,
			}, withAlpha(c, 0.6))
		}
		return
	}
	x, y := px(r.X)+ox, py(r.Y)+oy
	rad := sim.RockRadius * CellPx
	fillCircle(dst, x+3, y+5, rad, rgba(0, 0, 0, 80))
	fillCircle(dst, x, y, rad+2, rgba(40, 34, 30, 255))
	fillCircle(dst, x, y, rad, rgba(138, 128, 120, 255))
	for i := 0; i < 3; i++ {
		a := r.Angle + float64(i)*2.1
		fillCircle(dst, x+math.Cos(a)*rad*0.5, y+math.Sin(a)*rad*0.5, rad*0.22, rgba(100, 92, 86, 255))
	}
	fillCircle(dst, x-rad*0.35, y-rad*0.4, rad*0.25, rgba(175, 166, 158, 255))
}

func caveRect(c sim.CaveIn, ox, oy float64) (float64, float64, float64, float64) {
	if c.Horizontal {
		return FieldX + ox, py(float64(c.Line)) + oy, FieldW, CellPx
	}
	return px(float64(c.Line)) + ox, FieldY + oy, CellPx, FieldH
}

func (f *fieldView) drawCave(dst *ebiten.Image, c sim.CaveIn, t, ox, oy float64) {
	x, y, w, h := caveRect(c, ox, oy)
	if c.Warn > 0 {
		k := 1 - float64(c.Warn)/sim.CaveWarnTicks
		blink := 0.35 + 0.35*math.Abs(math.Sin(t*(0.15+k*0.4)))
		fillRect(dst, x, y, w, h, withAlpha(rgba(255, 120, 40, 255), 0.12+0.15*k))
		// Пунктир по краям полосы.
		for i := 0.0; i < math.Max(w, h); i += 24 {
			if c.Horizontal {
				fillRect(dst, x+i, y+2, 12, 3, withAlpha(rgba(255, 170, 60, 255), blink))
				fillRect(dst, x+i, y+h-5, 12, 3, withAlpha(rgba(255, 170, 60, 255), blink))
			} else {
				fillRect(dst, x+2, y+i, 3, 12, withAlpha(rgba(255, 170, 60, 255), blink))
				fillRect(dst, x+w-5, y+i, 3, 12, withAlpha(rgba(255, 170, 60, 255), blink))
			}
		}
	}
}

func (f *fieldView) drawCaveFall(dst *ebiten.Image, c sim.CaveIn, t, ox, oy float64) {
	x, y, w, h := caveRect(c, ox, oy)
	a := float64(c.Active) / sim.CaveHitTicks
	fillRect(dst, x, y, w, h, withAlpha(rgba(120, 100, 80, 255), 0.75*a))
	n := int(math.Max(w, h) / 14)
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
}
