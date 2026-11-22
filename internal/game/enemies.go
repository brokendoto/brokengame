package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Враги в том же стиле, что и герои: вид сбоку, тёмная обводка, большие
// глаза, лапки перебирают на ходу. Обычный и особый враг каждого мира
// различаются и формой, и цветом, чтобы особого было видно издалека.

var enemyBuf *ebiten.Image

func drawEnemy(dst *ebiten.Image, w sim.World, c sim.Chaser, t, ox, oy float64, st *sim.State) {
	x, y := px(c.X)+ox, py(c.Y)+oy
	if c.Spawn > 0 {
		k := 1 - float64(c.Spawn%45)/45
		strokeCircle(dst, x, y, 6+k*18, 3, withAlpha(rgba(255, 60, 60, 255), 1-k))
		fillCircle(dst, x, y, 6, rgba(40, 20, 30, 200))
		return
	}
	dx := float64(c.TX) + 0.5 - c.X
	if math.Abs(dx) < 0.05 {
		dx = st.P.X - c.X
	}
	dir := 1.0
	if dx < 0 {
		dir = -1
	}
	special := c.Kind != sim.ChaserBasic
	floating := special && (w == sim.Temple || w == sim.Space)
	alpha := 1.0
	if floating {
		alpha = 0.82
		if c.BlinkWarn > 0 {
			alpha = 0.3 + 0.3*math.Sin(t*0.6)
		}
	} else {
		fillPoly(dst, ellipse(x+1, y+17, 15, 5, 0, 16), rgba(0, 0, 0, 70))
	}
	if enemyBuf == nil {
		enemyBuf = ebiten.NewImage(heroBufW, heroBufH)
	}
	enemyBuf.Clear()
	cx, cy := float64(heroBufW)/2, float64(heroBufH)/2+6
	th := &themes[w]
	eye := th.eyeBasic
	if special {
		eye = th.eyeSpecial
	}
	drawCritter(enemyBuf, w, special, cx, cy, dir, t, eye)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-cx, -cy)
	op.GeoM.Scale(heroScale, heroScale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(float32(alpha))
	if c.Stun > 0 {
		op.ColorScale.Scale(0.6, 0.6, 0.6, 1)
	}
	dst.DrawImage(enemyBuf, op)
	if c.Stun > 0 {
		for i := 0; i < 3; i++ {
			a := t*0.15 + float64(i)*2.09
			fillCircle(dst, x+math.Cos(a)*14, y-26+math.Sin(a)*4, 2.5, rgba(255, 240, 120, 255))
		}
	}
}

// drawCritter рисует врага мира w (special — особый) с центром в (x, y).
func drawCritter(dst *ebiten.Image, w sim.World, special bool, x, y, dir, t float64, eye rgbaT) {
	ol := rgba(20, 14, 24, 255)
	blob := func(cx, cy, r float64, c rgbaT) {
		fillCircle(dst, cx, cy, r+2, ol)
		fillCircle(dst, cx, cy, r, c)
	}
	oval := func(cx, cy, rx, ry float64, c rgbaT) {
		fillPoly(dst, ellipse(cx, cy, rx+2, ry+2, 0, 20), ol)
		fillPoly(dst, ellipse(cx, cy, rx, ry, 0, 20), c)
	}
	// angryEyes — глаза с нахмуренными бровями.
	angryEyes := func(ex, ey, r float64, c rgbaT) {
		for _, s := range []float64{-1, 1} {
			px := ex + s*r*1.3
			fillCircle(dst, px, ey, r+1, ol)
			fillCircle(dst, px, ey, r, c)
			fillCircle(dst, px+dir*r*0.3, ey+r*0.15, r*0.45, ol)
			line(dst, px-r*1.1, ey-r*1.3-s*dir*r*0.5, px+r*1.1, ey-r*1.3+s*dir*r*0.5, 2, ol)
		}
	}
	ph := t * 0.35
	step := math.Sin(ph)
	switch w {
	case sim.Mine: // жук; особый — хитрый зелёный
		shell, belly := rgba(110, 70, 150, 255), rgba(70, 40, 96, 255)
		if special {
			shell, belly = rgba(60, 140, 86, 255), rgba(36, 80, 52, 255)
		}
		for i := -1.0; i <= 1; i++ {
			off := step * 2.5 * math.Pow(-1, i+1)
			line(dst, x+i*7, y+4, x+i*7+off, y+14, 3.5, ol)
		}
		oval(x-dir*2, y, 15, 11, shell)
		line(dst, x-dir*2, y-11, x-dir*2, y+10, 2, belly)
		fillPoly(dst, ellipse(x-dir*7, y-5, 4, 2.5, -0.5*dir, 10), rgba(255, 255, 255, 90))
		hx := x + dir*12
		blob(hx, y+3, 7, belly)
		for _, s := range []float64{-1, 1} {
			line(dst, hx+dir*2, y-2, hx+dir*(6+s*2)+math.Sin(t*0.2)*1.5, y-12, 2, ol)
		}
		fillCircle(dst, hx+dir*3-2.5, y+2, 2.4, eye)
		fillCircle(dst, hx+dir*3+2.5, y+2, 2.4, eye)
	case sim.Forest: // лиса; особый — серый волк
		body, light := rgba(226, 112, 40, 255), rgba(250, 240, 228, 255)
		if special {
			body, light = rgba(128, 130, 142, 255), rgba(214, 216, 224, 255)
		}
		// Хвост.
		tw := math.Sin(t*0.2) * 3
		oval(x-dir*14, y-2+tw*0.3, 8, 5, body)
		fillCircle(dst, x-dir*20, y-3+tw*0.3, 3.5, light)
		// Лапы: передняя и задняя пары шагают навстречу.
		for _, lx := range []float64{-7, 6} {
			s := step
			if lx > 0 {
				s = -step
			}
			line(dst, x+dir*lx, y+4, x+dir*lx+dir*s*3, y+14, 4, ol)
			line(dst, x+dir*lx, y+4, x+dir*lx+dir*s*3, y+13, 2, body)
		}
		oval(x, y+1, 12, 8, body)
		fillPoly(dst, ellipse(x+dir*2, y+5, 7, 3.5, 0, 12), light)
		// Голова с острыми ушами и мордочкой.
		hx, hy := x+dir*11, y-7
		for _, s := range []float64{-1, 1} {
			ex := hx + s*4.5
			fillPoly(dst, []float64{ex - 4.5, hy - 3, ex + s*1, hy - 15, ex + 4.5, hy - 3}, ol)
			fillPoly(dst, []float64{ex - 2.5, hy - 4, ex + s*1, hy - 12, ex + 2.5, hy - 4}, body)
		}
		blob(hx, hy, 8, body)
		fillPoly(dst, []float64{hx + dir*3, hy - 1, hx + dir*14, hy + 3, hx + dir*3, hy + 6}, light)
		fillCircle(dst, hx+dir*14, hy+3, 2, ol)
		angryEyes(hx+dir*2, hy-1, 2, eye)
	case sim.Temple:
		if special { // призрак: парит, хвост-волна
			c := rgba(236, 232, 255, 255)
			bob := math.Sin(t*0.12) * 2
			var pts []float64
			for i := 0; i <= 12; i++ {
				a := math.Pi + float64(i)/12*math.Pi
				pts = append(pts, x+math.Cos(a)*12, y-4+bob+math.Sin(a)*12)
			}
			for i := 6; i >= 0; i-- {
				fx := x - 12 + float64(i)*4
				fy := y + 10 + bob + math.Sin(t*0.3+float64(i))*2
				if i%2 == 1 {
					fy -= 4
				}
				pts = append(pts, fx, fy)
			}
			fillPoly(dst, pts, c)
			for _, s := range []float64{-1, 1} {
				fillPoly(dst, ellipse(x+dir*3+s*4.5, y-5+bob, 2.6, 4, 0, 12), rgba(60, 20, 90, 255))
			}
			fillPoly(dst, ellipse(x+dir*3, y+2+bob, 2.5, 3, 0, 10), rgba(60, 20, 90, 200))
			break
		}
		// Óни: красный, рожки, набедренная повязка, дубинка.
		body := rgba(200, 56, 56, 255)
		for _, s := range []float64{-1, 1} {
			oval(x+s*4+dir*step*s*2.5, y+12-math.Max(0, step*s)*2, 3.6, 2.6, rgba(120, 30, 30, 255))
		}
		fillCircle(dst, x, y+4, 10, ol)
		roundRect(dst, x-8, y-4, 16, 16, 6, body)
		fillRect(dst, x-8, y+5, 16, 4, rgba(240, 200, 60, 255))
		line(dst, x+dir*9, y+6, x+dir*14, y-12, 6, ol)
		line(dst, x+dir*9, y+6, x+dir*14, y-12, 3.5, rgba(130, 90, 50, 255))
		blob(x+dir*9, y+4, 3, body)
		blob(x-dir*9, y+3, 3, body)
		hx, hy := x+dir, y-11
		for _, s := range []float64{-1, 1} {
			fillPoly(dst, []float64{hx + s*4 - 3, hy - 6, hx + s*7, hy - 17, hx + s*4 + 3, hy - 6}, ol)
			fillPoly(dst, []float64{hx + s*4 - 1.5, hy - 7, hx + s*6.5, hy - 14.5, hx + s*4 + 1.5, hy - 7}, rgba(245, 235, 205, 255))
		}
		blob(hx, hy, 10, body)
		angryEyes(hx+dir*3, hy, 2.4, eye)
		line(dst, hx+dir*3-3, hy+6, hx+dir*3+3, hy+6, 2, ol)
	case sim.Front: // дрон; особый — жёлтый разведчик
		body := rgba(94, 100, 106, 255)
		if special {
			body = rgba(214, 180, 50, 255)
		}
		hover := math.Sin(t*0.2) * 2
		yy := y - 2 + hover
		// Винты на штангах.
		for _, s := range []float64{-1, 1} {
			bx := x + s*12
			line(dst, x, yy-4, bx, yy-10, 3, ol)
			w := 9 * math.Abs(math.Cos(t*0.9+s))
			line(dst, bx-w, yy-12, bx+w, yy-12, 2.5, rgba(220, 222, 230, 220))
			fillCircle(dst, bx, yy-11, 2.2, ol)
		}
		fillPoly(dst, ellipse(x, yy, 13, 9, 0, 20), ol)
		fillPoly(dst, ellipse(x, yy, 11, 7, 0, 20), body)
		fillRect(dst, x-11, yy+1, 22, 2, rgba(0, 0, 0, 60))
		// Глаз-камера.
		fillCircle(dst, x+dir*5, yy, 5, ol)
		fillCircle(dst, x+dir*5, yy, 3.6, eye)
		fillCircle(dst, x+dir*5.8, yy-1, 1.3, rgba(255, 255, 255, 220))
		line(dst, x-4, yy+8, x-6, yy+12, 2, ol)
		line(dst, x+4, yy+8, x+6, yy+12, 2, ol)
	case sim.Space:
		if special { // медуза-телепорт
			c := rgba(196, 110, 244, 255)
			bob := math.Sin(t*0.1) * 2
			for i := 0; i < 4; i++ {
				tx := x - 7.5 + float64(i)*5
				line(dst, tx, y+bob, tx+math.Sin(t*0.2+float64(i))*3, y+14+bob, 2.5, c)
			}
			fillPoly(dst, halfDome(x, y+1+bob, 13), c)
			fillPoly(dst, ellipse(x-4, y-6+bob, 4, 2.2, -0.4, 10), rgba(255, 255, 255, 110))
			fillCircle(dst, x+dir*2-4, y-3+bob, 2.6, eye)
			fillCircle(dst, x+dir*2+4, y-3+bob, 2.6, eye)
			break
		}
		// Зелёный пришелец: капля с одним большим глазом и антенной.
		body := rgba(100, 210, 100, 255)
		wob := 1 + 0.06*math.Sin(t*0.25)
		for _, s := range []float64{-1, 1} {
			oval(x+s*5+dir*step*s*2.5, y+12-math.Max(0, step*s)*2, 3.4, 2.4, rgba(50, 130, 60, 255))
		}
		line(dst, x, y-12, x+dir*3, y-20, 2, ol)
		fillCircle(dst, x+dir*3, y-21, 3, eye)
		fillPoly(dst, ellipse(x, y, 12*wob+2, 13/wob+2, 0, 20), ol)
		fillPoly(dst, ellipse(x, y, 12*wob, 13/wob, 0, 20), body)
		fillCircle(dst, x+dir*3, y-3, 6.5, ol)
		fillCircle(dst, x+dir*3, y-3, 5.5, rgba(244, 255, 244, 255))
		fillCircle(dst, x+dir*5, y-3, 2.8, rgba(10, 30, 10, 255))
		fillCircle(dst, x+dir*5.8, y-4, 1, rgba(255, 255, 255, 255))
		line(dst, x+dir*1, y+6, x+dir*6, y+5, 2, ol)
	}
}
