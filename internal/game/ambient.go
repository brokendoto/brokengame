package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Атмосферные частицы миров: пыль в шахте, листья и светлячки в лесу,
// лепестки сакуры и огоньки душ в храме, пепел и искры на передовой,
// мерцающие звёзды в космосе. Положение каждой частицы — чистая функция
// времени и её номера, поэтому хранить и обновлять ничего не нужно.

const ambientCount = 46

type mote struct{ x0, y0, sp, ph, k float64 }

var motes = func() [ambientCount]mote {
	var m [ambientCount]mote
	r := sim.NewRNG(4242)
	for i := range m {
		m[i] = mote{r.Float() * FieldW, r.Float() * FieldH, 0.5 + r.Float(), r.Float() * 6.283, r.Float()}
	}
	return m
}()

func wrap(v, n float64) float64 {
	v = math.Mod(v, n)
	if v < 0 {
		v += n
	}
	return v
}

// drawAmbient рисует частицы. lit=false — слой под освещением (его видно
// только там, где светло), lit=true — светящийся слой поверх темноты.
func drawAmbient(dst *ebiten.Image, w sim.World, t, ox, oy float64, lit bool) {
	for i, m := range motes {
		X := func(x float64) float64 { return FieldX + wrap(x, FieldW) + ox }
		Y := func(y float64) float64 { return FieldY + wrap(y, FieldH) + oy }
		switch w {
		case sim.Mine:
			if !lit { // пыль в луче фонаря
				x, y := X(m.x0+t*0.08*m.sp+math.Sin(t*0.01+m.ph)*30), Y(m.y0+t*0.05*m.sp)
				fillCircle(dst, x, y, 1+m.k*1.5, rgba(200, 190, 170, 110))
			} else if i%5 == 0 { // искорки кристаллов
				a := math.Pow(math.Max(0, math.Sin(t*0.03*m.sp+m.ph)), 6)
				x, y := X(m.x0), Y(m.y0+math.Sin(t*0.01+m.ph)*10)
				glow(dst, x, y, 10, rgba(120, 230, 255, 255), a*0.8)
			}
		case sim.Forest:
			if !lit && i%2 == 0 { // падающие листья
				x := X(m.x0 + t*0.35*m.sp + math.Sin(t*0.02*m.sp+m.ph)*40)
				y := Y(m.y0 + t*0.45*m.sp)
				drawLeaf(dst, x, y, t*0.04*m.sp+m.ph, 4+m.k*3, i)
			} else if lit && i%3 == 1 { // светлячки
				x := X(m.x0 + math.Sin(t*0.007*m.sp+m.ph)*70)
				y := Y(m.y0 + math.Cos(t*0.009*m.sp+m.ph*1.3)*50)
				a := math.Max(0, math.Sin(t*0.04*m.sp+m.ph))
				glow(dst, x, y, 16, rgba(220, 255, 120, 255), a*0.55)
				fillCircle(dst, x, y, 1.8, withAlpha(rgba(250, 255, 200, 255), a))
			}
		case sim.Temple:
			if lit && i%2 == 0 { // лепестки сакуры
				x := X(m.x0 + t*0.3*m.sp + math.Sin(t*0.015*m.sp+m.ph)*50)
				y := Y(m.y0 + t*0.3*m.sp)
				drawPetal(dst, x, y, t*0.05*m.sp+m.ph, 3+m.k*2)
			} else if lit && i%5 == 1 { // огоньки душ
				x := X(m.x0 + math.Sin(t*0.006+m.ph)*40)
				y := Y(m.y0 - t*0.12*m.sp)
				a := 0.5 + 0.5*math.Sin(t*0.05+m.ph)
				glow(dst, x, y, 18, rgba(150, 200, 255, 255), 0.35*a)
				fillCircle(dst, x, y, 2, withAlpha(rgba(220, 240, 255, 255), 0.7*a))
			}
		case sim.Front:
			if !lit { // пепел
				x := X(m.x0 + t*0.2*m.sp + math.Sin(t*0.01+m.ph)*20)
				y := Y(m.y0 + t*0.25*m.sp)
				fillRect(dst, x, y, 2+m.k*2, 2+m.k*2, rgba(170, 165, 160, 150))
			} else if i%3 == 0 { // искры, летящие вверх
				x := X(m.x0 + math.Sin(t*0.03*m.sp+m.ph)*14 + t*0.1)
				y := Y(m.y0 - t*0.6*m.sp)
				a := 0.5 + 0.5*math.Sin(t*0.2+m.ph)
				glow(dst, x, y, 8, rgba(255, 140, 50, 255), 0.6*a)
				fillCircle(dst, x, y, 1.4, withAlpha(rgba(255, 220, 150, 255), a))
			}
		case sim.Space:
			if lit { // мерцающие звёзды-блики
				a := math.Pow(math.Max(0, math.Sin(t*0.02*m.sp+m.ph)), 4)
				if a < 0.05 {
					continue
				}
				x, y := X(m.x0-t*0.03), Y(m.y0)
				c := rgba(200, 220, 255, 255)
				if i%4 == 0 {
					c = rgba(255, 220, 160, 255)
				}
				r := (3 + m.k*4) * a
				line(dst, x-r, y, x+r, y, 1.2, withAlpha(c, a))
				line(dst, x, y-r, x, y+r, 1.2, withAlpha(c, a))
				glow(dst, x, y, 8, c, a*0.5)
			}
		}
	}
}

func drawLeaf(dst *ebiten.Image, x, y, ang, s float64, i int) {
	c := rgba(214, 120, 40, 230)
	if i%3 == 0 {
		c = rgba(230, 180, 60, 230)
	} else if i%5 == 0 {
		c = rgba(170, 60, 40, 230)
	}
	ca, sa := math.Cos(ang), math.Sin(ang)
	sq := 0.45 + 0.55*math.Abs(math.Sin(ang*1.7)) // лист переворачивается в полёте
	p := func(u, v float64) (float64, float64) { u *= s; v *= s * sq; return x + u*ca - v*sa, y + u*sa + v*ca }
	ax, ay := p(-1, 0)
	bx, by := p(0, -0.6)
	cx, cy := p(1, 0)
	dx, dy := p(0, 0.6)
	fillPoly(dst, []float64{ax, ay, bx, by, cx, cy, dx, dy}, c)
}

func drawPetal(dst *ebiten.Image, x, y, ang, s float64) {
	sq := 0.4 + 0.6*math.Abs(math.Sin(ang*1.3))
	fillPoly(dst, ellipse(x, y, s, s*0.6*sq+0.5, ang, 10), rgba(255, 170, 200, 200))
	fillPoly(dst, ellipse(x-s*0.2, y-s*0.1, s*0.45, s*0.25*sq+0.3, ang, 8), rgba(255, 220, 235, 200))
}
