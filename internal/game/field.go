package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Отрисовка поля. Вся графика процедурная (векторные фигуры и градиенты),
// картинок-спрайтов нет — как и в оригинале. Оформление миров — в themes.go.

const (
	CellPx = 48
	FieldX = (ScreenW - sim.W*CellPx) / 2
	FieldY = 88
	FieldW = sim.W * CellPx
	FieldH = sim.H * CellPx
)

func px(x float64) float64 { return FieldX + x*CellPx }
func py(y float64) float64 { return FieldY + y*CellPx }

// Цвета игроков дуэли.
var duelColors = [2]rgbaT{rgba(255, 150, 70, 255), rgba(90, 200, 255, 255)}

// Цвета кристалла шахты.
var (
	crystalBody = rgba(90, 220, 255, 255)
	crystalHi   = rgba(210, 250, 255, 255)
	crystalLo   = rgba(40, 130, 200, 255)
	exitGlow    = rgba(120, 255, 190, 255)
)

func cellHash(x, y, k int) uint64 {
	r := sim.NewRNG(uint64(x*73856093) ^ uint64(y*19349663) ^ uint64(k*83492791))
	return r.Next()
}

// fieldView — отрисовщик уровня; статичная часть (пол, стены, зоны)
// рисуется один раз в отдельную картинку.
type fieldView struct {
	duel   *sim.Duel // не nil в дуэли: рисуем двух героев
	lv     *sim.Level
	w      sim.World
	static *ebiten.Image

	lastPos [2][2]float64 // где были герои в прошлый кадр — для пыли из-под ног
	stepT   [2]int
}

func newFieldView(lv *sim.Level) *fieldView {
	f := &fieldView{lv: lv, w: lv.Track.World, static: ebiten.NewImage(FieldW, FieldH)}
	f.renderStatic()
	return f
}

func (f *fieldView) renderStatic() {
	img := f.static
	lv := f.lv
	img.Fill(themes[f.w].floorB)
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			switch lv.Grid[y][x] {
			case sim.Wall:
			case sim.Floor:
				renderFloor(img, f.w, x, y)
			default:
				renderZone(img, f.w, x, y)
			}
		}
	}
	// Мягкие тени от стен (свет сверху-слева): длинная снизу, короткая справа,
	// и лёгкое затенение пола вдоль любой стены (ambient occlusion).
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] == sim.Wall {
				continue
			}
			X, Y := float64(x*CellPx), float64(y*CellPx)
			for i := 0; i < 8; i++ {
				k := 1 - float64(i)/8
				a := uint8(70 * k * k)
				if lv.Solid(x, y-1) && y > 0 {
					fillRect(img, X, Y+float64(i)*2.5, CellPx, 2.5, rgba(0, 0, 0, a))
				}
				if lv.Solid(x-1, y) && x > 0 {
					fillRect(img, X+float64(i)*1.5, Y, 1.5, CellPx, rgba(0, 0, 0, a*2/3))
				}
				if lv.Solid(x+1, y) && x < sim.W-1 {
					fillRect(img, X+CellPx-float64(i+1)*1, Y, 1, CellPx, rgba(0, 0, 0, a/3))
				}
				if lv.Solid(x, y+1) && y < sim.H-1 {
					fillRect(img, X, Y+CellPx-float64(i+1)*1, CellPx, 1, rgba(0, 0, 0, a/3))
				}
			}
		}
	}
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] == sim.Wall {
				renderWall(img, lv, f.w, x, y)
			}
		}
	}
	drawLamps(img, lv, f.w)
}

// drawWorld рисует поле со всеми объектами. ghost может быть nil.
func (f *fieldView) drawWorld(dst *ebiten.Image, st, ghost *sim.State, frame int, fx *fxSystem, trail [][2]float64, ghostName string) {
	ox, oy := fx.offset()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(FieldX+ox, FieldY+oy)
	dst.DrawImage(f.static, op)

	field := dst.SubImage(image.Rect(FieldX, FieldY, FieldX+FieldW, FieldY+FieldH)).(*ebiten.Image)
	t := float64(frame)
	w := f.w
	th := &themes[w]
	cpx := func(p sim.Pt) (float64, float64) { return px(float64(p.X)+0.5) + ox, py(float64(p.Y)+0.5) + oy }
	drawAmbient(field, w, t, ox, oy, false)

	// Чёрные дыры — под всем остальным.
	for _, h := range f.lv.Holes {
		x, y := cpx(h)
		drawHole(field, x, y, t)
	}
	// Шипы/мины.
	for _, sp := range f.lv.Spikes {
		drawTrap(field, w, px(float64(sp.P.X))+ox, py(float64(sp.P.Y))+oy, st.SpikePhase(sp))
	}
	f.drawExit(field, st, t, ox, oy)
	// Предупреждения обвалов/лучей.
	for _, c := range st.Caves {
		f.drawCave(field, c, t, ox, oy)
	}
	// Предметы.
	for i, it := range st.Items {
		if it.Taken {
			continue
		}
		x, y := cpx(it.P)
		drawItem(field, w, x, y+math.Sin(t*0.07+float64(i))*2.5, 1, t+float64(i)*10, 0)
	}
	for i, it := range st.Cursed {
		if it.Taken {
			continue
		}
		x, y := cpx(it.P)
		drawItem(field, w, x, y+math.Sin(t*0.06+float64(i))*2, 0.9, t, 2)
	}
	for i, it := range st.Golden {
		if it.Taken {
			continue
		}
		left := st.GoldenLeft(i)
		if left < 3*sim.TPS && (frame/5)%2 == 0 {
			continue
		}
		x, y := cpx(it.P)
		s := 1.05 + 0.08*math.Sin(t*0.2)
		drawItem(field, w, x, y+math.Sin(t*0.08)*2.5, s, t, 1)
		// Кольцо-таймер: сколько осталось жить бонусу.
		k := math.Min(1, float64(left)/float64(sim.GoldenLife))
		arc := []float64{}
		for a := 0.0; a <= k*2*math.Pi+0.01; a += 0.2 {
			arc = append(arc, x+math.Sin(a)*22, y-math.Cos(a)*22)
		}
		for j := 0; j+3 < len(arc); j += 2 {
			line(field, arc[j], arc[j+1], arc[j+2], arc[j+3], 2, withAlpha(colGold, 0.7))
		}
	}
	if f.lv.HasHeart && !st.HeartTaken {
		x, y := cpx(f.lv.Heart)
		drawHeart(field, x, y+math.Sin(t*0.09)*2, 1+0.08*math.Sin(t*0.15), rgba(255, 80, 100, 255))
	}
	// Катящиеся объекты.
	for _, r := range st.Rocks {
		if r.Warn > 0 {
			drawRockWarn(field, r, ox, oy)
			continue
		}
		drawRolling(field, w, r, px(r.X)+ox, py(r.Y)+oy)
	}
	// Метки, куда прыгнет призрак.
	for _, c := range st.Chasers {
		if c.BlinkWarn > 0 {
			x, y := px(float64(c.BX)+0.5)+ox, py(float64(c.BY)+0.5)+oy
			k := float64(c.BlinkWarn) / sim.PhantomWarn
			strokeCircle(field, x, y, 8+16*k, 3, withAlpha(th.eyeSpecial, 0.9-0.5*k))
			fillCircle(field, x, y, 5, withAlpha(th.eyeSpecial, 0.6))
		}
	}
	// Враги.
	for _, c := range st.Chasers {
		drawEnemy(field, w, c, t, ox, oy, st)
	}
	// Призрак лучшего забега.
	if ghost != nil {
		gx, gy := px(ghost.P.X)+ox, py(ghost.P.Y)+oy
		drawHero(field, w, ghost.P, gx, gy, t, 0.38, true)
		if ghostName != "" {
			drawText(field, ghostName, 13, false, gx, gy-36, Center, rgba(200, 230, 255, 160))
		}
	}
	// След рывка.
	for i, p := range trail {
		a := float64(i+1) / float64(len(trail)+1) * 0.45
		glow(field, px(p[0])+ox, py(p[1])+oy, 26, th.lampColor, a*0.6)
		drawHero(field, w, st.P, px(p[0])+ox, py(p[1])+oy, t, a, true)
	}
	// Игрок (мигает после удара). В дуэли — оба, с цветными кольцами.
	heroes := []sim.Player{st.P}
	if f.duel != nil {
		heroes = heroes[:0]
		for i, d := range f.duel.D {
			if d.Down > 0 {
				x, y := px(float64(d.Start.X)+0.5)+ox, py(float64(d.Start.Y)+0.5)+oy
				strokeCircle(field, x, y, 18, 3, withAlpha(duelColors[i], 0.6))
				drawText(field, itoa(d.Down/sim.TPS+1), 20, true, x, y-12, Center, duelColors[i])
				continue
			}
			heroes = append(heroes, d.P)
		}
	}
	for _, p := range heroes {
		x, y := px(p.X)+ox, py(p.Y)+oy
		if p.MagnetT > 0 {
			a := 0.25
			if p.MagnetT < 60 && (p.MagnetT/6)%2 == 0 {
				a = 0.1
			}
			strokeCircle(field, x, y, sim.MagnetRadius*CellPx, 2, withAlpha(abilityColor(sim.AbMagnet), a))
			fillCircle(field, x, y, sim.MagnetRadius*CellPx, withAlpha(abilityColor(sim.AbMagnet), a*0.15))
		}
	}
	if f.duel != nil {
		for i, d := range f.duel.D {
			if d.Down > 0 {
				continue
			}
			x, y := px(d.P.X)+ox, py(d.P.Y)+oy
			fillPoly(field, ellipse(x, y+12, 19, 9, 0, 18), withAlpha(duelColors[i], 0.55))
			if d.P.Invuln == 0 || (frame/4)%2 == 0 {
				drawHero(field, w, d.P, x, y, t, 1, false)
			}
			drawTextShadow(field, itoa(i+1), 15, true, x, y-38, Center, duelColors[i])
			if d.Stun > 0 {
				for k := 0; k < 3; k++ {
					a := t*0.2 + float64(k)*2.09
					fillCircle(field, x+math.Cos(a)*15, y-24+math.Sin(a)*4, 3, rgba(255, 240, 120, 255))
				}
			}
		}
	} else if st.P.Invuln == 0 || (frame/4)%2 == 0 {
		drawHero(field, w, st.P, px(st.P.X)+ox, py(st.P.Y)+oy, t, 1, false)
	}

	// Туман храма — поверх всего, что в нём.
	if w == sim.Temple {
		for y := 0; y < sim.H; y++ {
			for x := 0; x < sim.W; x++ {
				if f.lv.Grid[y][x] != sim.Fog {
					continue
				}
				X, Y := px(float64(x))+ox, py(float64(y))+oy
				for i := 0; i < 3; i++ {
					hh := cellHash(x, y, 70+i)
					cx := X + float64(hh%32) + 8 + math.Sin(t*0.02+float64(hh%7))*6
					cy := Y + float64(hh/32%32) + 8
					fillCircle(field, cx, cy, 16+float64(hh%6), withAlpha(th.zoneHi, 0.16))
				}
			}
		}
	}

	f.footsteps(heroes, fx, frame)
	f.drawLighting(field, st, heroes, fx, t, ox, oy)
	drawAmbient(field, w, t, ox, oy, true)
	fx.draw(field, 0, 0)
	fx.drawRings(field)
	// Стоп-время: мир становится сиреневым.
	if st.FreezeT > 0 {
		a := 0.18
		if st.FreezeT < 40 {
			a *= float64(st.FreezeT) / 40
		}
		fillRect(field, FieldX, FieldY, FieldW, FieldH, withAlpha(abilityColor(sim.AbTimeStop), a))
	}

	// Светящиеся объекты видны и в темноте.
	for i, it := range st.Items {
		if !it.Taken {
			x, y := cpx(it.P)
			glow(field, x, y, 34, th.itemGlow, 0.45+0.15*math.Sin(t*0.1+float64(i)))
		}
	}
	for _, it := range st.Golden {
		if !it.Taken {
			x, y := cpx(it.P)
			glow(field, x, y, 40, colGold, 0.5)
		}
	}
	for _, it := range st.Cursed {
		if !it.Taken {
			x, y := cpx(it.P)
			glow(field, x, y, 30, rgba(255, 60, 140, 255), 0.35)
		}
	}
	for _, h := range f.lv.Holes {
		x, y := cpx(h)
		glow(field, x, y, 60, rgba(170, 90, 255, 255), 0.35)
	}
	if st.ExitOpen {
		x, y := cpx(f.lv.Exit)
		glow(field, x, y, 80, exitGlow, 0.6+0.2*math.Sin(t*0.12))
	}
	for _, c := range st.Chasers {
		if c.Spawn == 0 {
			eye := th.eyeBasic
			if c.Kind != sim.ChaserBasic {
				eye = th.eyeSpecial
			}
			glow(field, px(c.X)+ox, py(c.Y)+oy, 22, eye, 0.5)
		}
	}
	for _, h := range heroes {
		if h.ShieldT > 0 {
			glow(field, px(h.X)+ox, py(h.Y)+oy, 50, colAccent, 0.35)
		}
	}
	for _, c := range st.Caves {
		if c.Warn == 0 {
			x, y, bw, bh := caveRect(c, ox, oy)
			drawBeamActive(field, w, c, x, y, bw, bh, t)
		}
	}
	fx.drawTexts(field, 0, 0)
	if fx.flash > 0.01 && screenShake {
		fillRect(field, FieldX, FieldY, FieldW, FieldH, withAlpha(rgba(200, 30, 30, 255), fx.flash*0.35))
	}
	roundRectStroke(dst, FieldX-3, FieldY-3, FieldW+6, FieldH+6, 6, 3, lerpColor(th.wall, th.wallLo, 0.3))
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
	drawExitGlyph(dst, f.w, X, Y, t)
}

// drawRockWarn — мигающая стрелка там, откуда сейчас покатится камень.
func drawRockWarn(dst *ebiten.Image, r sim.Rock, ox, oy float64) {
	if (r.Warn/6)%2 != 0 {
		return
	}
	X, Y := px(float64(r.Spawn.X)+0.5)+ox, py(float64(r.Spawn.Y)+0.5)+oy
	dx, dy := float64(r.DX), float64(r.DY)
	c := rgba(255, 170, 60, 220)
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
