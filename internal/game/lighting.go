package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Цветное освещение поля. Каждый кадр собирается «карта света»: она заливается
// окружающим светом мира, поверх складываются (BlendLighter) пятна от фонаря
// героя, предметов, выхода, глаз врагов, фонарей храма и т. д., а потом карта
// умножается на картинку поля. Так шахта и храм тёмные, но всё светящееся
// подсвечивает пол и стены своим цветом, а лес днём лишь чуть оттеняется.

// multiplyBlend: итог = поле × карта света.
var multiplyBlend = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorZero,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorSourceColor,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

// lightImg — мягкое световое пятно (плавный спад к краю, без резкой границы).
var lightImg = func() *ebiten.Image {
	const n = 128
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
			d2 := dx*dx + dy*dy
			a := math.Max(0, 1-d2)
			a = a * a * (0.6 + 0.4*math.Max(0, 1-math.Sqrt(d2)*1.6)) // яркое ядро
			v := byte(math.Min(255, a*255))
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, 255
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}()

// vignetteImg — затемнение к краям поля (маленькая картинка, растягивается).
var vignetteImg = func() *ebiten.Image {
	const n = 64
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
			d := math.Sqrt(dx*dx*0.8 + dy*dy)
			a := clamp((d-0.55)/0.75, 0, 1)
			a = a * a * 0.55
			i := (y*n + x) * 4
			pix[i+3] = byte(a * 255)
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}()

type lightMap struct {
	img *ebiten.Image
}

var sharedLight *lightMap

func getLightMap() *lightMap {
	if sharedLight == nil {
		sharedLight = &lightMap{img: ebiten.NewImage(FieldW, FieldH)}
	}
	return sharedLight
}

func (l *lightMap) begin(ambient rgbaT) { l.img.Fill(ambient) }

// add добавляет пятно света; x, y — координаты на экране.
func (l *lightMap) add(x, y, r float64, c rgbaT, k float64) {
	if k <= 0 || r <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	s := r * 2 / float64(lightImg.Bounds().Dx())
	op.GeoM.Translate(-float64(lightImg.Bounds().Dx())/2, -float64(lightImg.Bounds().Dy())/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(x-FieldX, y-FieldY)
	op.ColorScale.Scale(float32(float64(c.R)/255*k), float32(float64(c.G)/255*k), float32(float64(c.B)/255*k), 1)
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	l.img.DrawImage(lightImg, op)
}

// apply умножает поле на карту света и добавляет виньетку.
func (l *lightMap) apply(field *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(FieldX, FieldY)
	op.Blend = multiplyBlend
	field.DrawImage(l.img, op)
	vop := &ebiten.DrawImageOptions{}
	vop.GeoM.Scale(float64(FieldW)/64, float64(FieldH)/64)
	vop.GeoM.Translate(FieldX, FieldY)
	vop.Filter = ebiten.FilterLinear
	field.DrawImage(vignetteImg, vop)
}

// flicker — живое мерцание огня: сумма двух синусов, 0.85..1.15.
func flicker(t, seed float64) float64 {
	return 1 + 0.09*math.Sin(t*0.31+seed) + 0.06*math.Sin(t*0.73+seed*2.1)
}

// worldLights — свет, который даёт сам мир: фонари храма, солнечные пятна
// в лесу, огни на передовой, лампы на станции.
func (f *fieldView) worldLights(l *lightMap, t, ox, oy float64) {
	th := &themes[f.w]
	lv := f.lv
	switch f.w {
	case sim.Forest: // солнечные зайчики сквозь листву медленно плывут
		for i := 0; i < 7; i++ {
			h := float64(cellHash(i, 3, 91) % 1000)
			x := math.Mod(h*1.37+t*0.12*(1+float64(i%3)*0.3), FieldW+300) - 150
			y := math.Mod(h*0.71, FieldH) + math.Sin(t*0.004+h)*40
			l.add(FieldX+x+ox, FieldY+y+oy, 170+h/10, rgba(255, 236, 190, 255), 0.22)
		}
	}
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] != sim.Wall || !lampSpot(lv, x, y) {
				continue
			}
			X, Y := px(float64(x)+0.5)+ox, py(float64(y)+1)+oy
			s := float64(x*31 + y*17)
			switch f.w {
			case sim.Temple:
				l.add(X, Y+4, 150, rgba(255, 170, 90, 255), 0.75*flicker(t, s))
			case sim.Mine:
				l.add(X, Y+2, 110, rgba(255, 190, 110, 255), 0.55*flicker(t, s))
			case sim.Front:
				l.add(X, Y+2, 120, rgba(255, 120, 60, 255), 0.5*flicker(t*1.3, s))
			case sim.Space:
				k := 0.45 + 0.1*math.Sin(t*0.05+s)
				l.add(X, Y+2, 110, th.zoneHi, k)
			}
		}
	}
}

// lampSpot — висит ли на этой стене светильник: стена над полом, примерно
// каждая девятая (решает хэш клетки, поэтому место постоянное).
func lampSpot(lv *sim.Level, x, y int) bool {
	if y+1 >= sim.H || lv.Solid(x, y+1) {
		return false
	}
	return cellHash(x, y, 77)%9 == 0
}

// drawLamps рисует сами светильники на стенах (в статичной картинке поля).
func drawLamps(img *ebiten.Image, lv *sim.Level, w sim.World) {
	if w == sim.Forest {
		return
	}
	for y := 0; y < sim.H; y++ {
		for x := 0; x < sim.W; x++ {
			if lv.Grid[y][x] != sim.Wall || !lampSpot(lv, x, y) {
				continue
			}
			X, Y := float64(x*CellPx)+CellPx/2, float64((y+1)*CellPx)
			switch w {
			case sim.Temple: // бумажный фонарик
				line(img, X, Y-14, X, Y-6, 1.5, rgba(40, 20, 20, 255))
				fillPoly(img, ellipse(X, Y+2, 8, 10, 0, 16), rgba(255, 150, 70, 255))
				fillPoly(img, ellipse(X, Y+2, 5, 8, 0, 16), rgba(255, 220, 150, 255))
				fillRect(img, X-5, Y-9, 10, 3, rgba(60, 24, 24, 255))
				fillRect(img, X-5, Y+11, 10, 3, rgba(60, 24, 24, 255))
			case sim.Mine: // фонарь на крюке
				line(img, X, Y-12, X, Y-4, 2, rgba(50, 40, 34, 255))
				roundRect(img, X-6, Y-5, 12, 13, 3, rgba(60, 46, 36, 255))
				fillRect(img, X-4, Y-3, 8, 9, rgba(255, 210, 120, 255))
			case sim.Front: // бочка-костёр
				roundRect(img, X-8, Y-10, 16, 12, 3, rgba(70, 60, 50, 255))
				fillPoly(img, []float64{X - 6, Y - 10, X, Y - 20, X + 6, Y - 10}, rgba(255, 140, 50, 255))
				fillPoly(img, []float64{X - 3, Y - 10, X, Y - 16, X + 3, Y - 10}, rgba(255, 230, 140, 255))
			case sim.Space: // сигнальная лампа
				fillRect(img, X-8, Y-6, 16, 5, rgba(40, 46, 70, 255))
				fillRect(img, X-5, Y-4, 10, 3, themes[w].zoneHi)
			}
		}
	}
}

// drawLighting собирает карту света кадра и накладывает её на поле.
func (f *fieldView) drawLighting(field *ebiten.Image, st *sim.State, heroes []sim.Player, fx *fxSystem, t, ox, oy float64) {
	th := &themes[f.w]
	l := getLightMap()
	amb := th.ambient
	if f.duel != nil { // в дуэли светлее, чтобы оба игрока видели всё поле
		amb = lerpColor(amb, rgba(255, 255, 255, 255), 0.55)
	}
	l.begin(amb)
	cpx := func(p sim.Pt) (float64, float64) { return px(float64(p.X)+0.5) + ox, py(float64(p.Y)+0.5) + oy }
	f.worldLights(l, t, ox, oy)

	// Фонарь героя (в тёмных мирах — большой и тёплый) — светит чуть вперёд.
	lampR, lampK := 150.0, 0.35
	if th.darkness >= 0.3 {
		lampR, lampK = 250, 1.0
	} else if th.darkness > 0.2 {
		lampR, lampK = 200, 0.6
	}
	for i, h := range heroes {
		x, y := px(h.X)+ox, py(h.Y)+oy
		c := th.lampColor
		if f.duel != nil {
			c = lerpColor(c, duelColors[i%2], 0.35)
		}
		k := lampK * (1 + 0.03*math.Sin(t*0.2))
		l.add(x+h.FX*30, y+h.FY*30, lampR, c, k)
		l.add(x, y, lampR*0.45, c, k*0.5)
		if h.ShieldT > 0 {
			l.add(x, y, 120, colAccent, 0.5)
		}
		if h.MagnetT > 0 {
			l.add(x, y, sim.MagnetRadius*CellPx*1.2, abilityColor(sim.AbMagnet), 0.25)
		}
	}
	for i, it := range st.Items {
		if !it.Taken {
			x, y := cpx(it.P)
			l.add(x, y, 90, th.itemGlow, 0.5+0.12*math.Sin(t*0.1+float64(i)))
		}
	}
	for _, it := range st.Golden {
		if !it.Taken {
			x, y := cpx(it.P)
			l.add(x, y, 110, colGold, 0.65)
		}
	}
	for _, it := range st.Cursed {
		if !it.Taken {
			x, y := cpx(it.P)
			l.add(x, y, 70, rgba(255, 60, 140, 255), 0.45)
		}
	}
	if f.lv.HasHeart && !st.HeartTaken {
		x, y := cpx(f.lv.Heart)
		l.add(x, y, 70, rgba(255, 90, 110, 255), 0.45)
	}
	for _, h := range f.lv.Holes {
		x, y := cpx(h)
		l.add(x, y, 120, rgba(170, 90, 255, 255), 0.55)
	}
	{
		x, y := cpx(f.lv.Exit)
		if st.ExitOpen {
			l.add(x, y, 190, exitGlow, 0.85+0.15*math.Sin(t*0.12))
		} else {
			l.add(x, y, 70, colGold, 0.25)
		}
	}
	for _, c := range st.Chasers {
		if c.Spawn == 0 {
			eye := th.eyeBasic
			if c.Kind != sim.ChaserBasic {
				eye = th.eyeSpecial
			}
			l.add(px(c.X)+ox, py(c.Y)+oy, 70, eye, 0.45)
		}
	}
	for _, c := range st.Caves {
		x, y, bw, bh := caveRect(c, ox, oy)
		k, col := 0.25, rgba(255, 140, 60, 255)
		if c.Warn == 0 {
			k, col = 0.7, th.beam
		}
		for d := 0.0; d <= math.Max(bw, bh); d += CellPx {
			if c.Horizontal {
				l.add(x+d, y+bh/2, 80, col, k)
			} else {
				l.add(x+bw/2, y+d, 80, col, k)
			}
		}
	}
	for _, p := range fx.parts {
		if p.glow {
			a := float64(p.life) / float64(p.max)
			l.add(p.x, p.y, p.size*9, p.c, a*0.3)
		}
	}
	l.apply(field)
	// Сами светильники светятся поверх темноты.
	if f.w != sim.Forest {
		for y := 0; y < sim.H; y++ {
			for x := 0; x < sim.W; x++ {
				if f.lv.Grid[y][x] == sim.Wall && lampSpot(f.lv, x, y) {
					c := rgba(255, 180, 90, 255)
					if f.w == sim.Space {
						c = th.zoneHi
					}
					glow(field, px(float64(x)+0.5)+ox, py(float64(y)+1)+oy, 22, c, 0.5*flicker(t, float64(x*31+y*17)))
				}
			}
		}
	}
}

// footsteps — пыль из-под ног бегущего героя (в космосе — искры ранца).
func (f *fieldView) footsteps(heroes []sim.Player, fx *fxSystem, frame int) {
	for i, h := range heroes {
		if i > 1 {
			break
		}
		last := f.lastPos[i]
		f.lastPos[i] = [2]float64{h.X, h.Y}
		d := math.Hypot(h.X-last[0], h.Y-last[1])
		if d < 0.02 || d > 1 {
			continue
		}
		f.stepT[i]++
		if f.stepT[i]%7 != 0 {
			continue
		}
		x, y := px(h.X), py(h.Y)+16
		if f.w == sim.Space {
			fx.trailSpark(x-h.FX*14, y-10-h.FY*14, rgba(120, 200, 255, 255))
		} else {
			fx.puff(x, y, lerpColor(themes[f.w].speck, rgba(220, 210, 200, 255), 0.4))
		}
	}
}
