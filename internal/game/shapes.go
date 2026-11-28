package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Сглаженные фигуры одним вызовом DrawTriangles: заливка многоугольника и
// края шириной в пиксель, которые плавно уходят в прозрачность.
//
// Раньше фигуры рисовал пакет vector со сглаживанием. Он рисует через
// буфер-трафарет и сбрасывает его всякий раз, когда между фигурами на тот же
// холст рисуется картинка или текст. В меню и на уровне это сотни
// перерисовок большого буфера за кадр — на встроенной видеокарте ноутбука и в
// браузере игра опускалась до единиц кадров в секунду. Треугольники отсюда
// склеиваются Ebitengine в общие пакеты, как обычные спрайты.

var whitePix = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

type vec struct{ x, y float32 }

var (
	shapeVs  []ebiten.Vertex
	shapeIs  []uint32
	shapePts []vec
	shapeNrm []vec
	earIdx   []int
	triOpts  = &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha}
)

// premul переводит цвет в четыре float32 (color.RGBA уже премультиплицирован).
func premul(c color.Color) [4]float32 {
	r, g, b, a := c.RGBA()
	return [4]float32{float32(r) / 0xffff, float32(g) / 0xffff, float32(b) / 0xffff, float32(a) / 0xffff}
}

func addV(x, y float32, c [4]float32) {
	shapeVs = append(shapeVs, ebiten.Vertex{DstX: x, DstY: y, SrcX: 1.5, SrcY: 1.5,
		ColorR: c[0], ColorG: c[1], ColorB: c[2], ColorA: c[3]})
}

func flushShape(dst *ebiten.Image) {
	if len(shapeIs) > 0 {
		dst.DrawTriangles32(shapeVs, shapeIs, whitePix, triOpts)
	}
	shapeVs, shapeIs = shapeVs[:0], shapeIs[:0]
}

// cleanPts убирает совпадающие соседние точки (и замыкающую копию первой).
func cleanPts(pts []vec, closed bool) []vec {
	out := pts[:0]
	for _, p := range pts {
		if n := len(out); n > 0 && absf(out[n-1].x-p.x) < 1e-3 && absf(out[n-1].y-p.y) < 1e-3 {
			continue
		}
		out = append(out, p)
	}
	if closed {
		for len(out) > 1 && absf(out[0].x-out[len(out)-1].x) < 1e-3 && absf(out[0].y-out[len(out)-1].y) < 1e-3 {
			out = out[:len(out)-1]
		}
	}
	return out
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// normals считает для каждой точки смещение наружу (со «скосом» на углах).
// sign — ориентация многоугольника; для незамкнутой линии концы берут
// нормаль своего отрезка.
func normals(pts []vec, closed bool, sign float32) []vec {
	n := len(pts)
	shapeNrm = shapeNrm[:0]
	edge := func(i int) vec {
		a, b := pts[i], pts[(i+1)%n]
		dx, dy := b.x-a.x, b.y-a.y
		l := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if l < 1e-6 {
			return vec{}
		}
		return vec{dy / l * sign, -dx / l * sign}
	}
	for i := 0; i < n; i++ {
		var n0, n1 vec
		switch {
		case closed:
			n0, n1 = edge((i+n-1)%n), edge(i)
		case i == 0:
			n0 = edge(0)
			n1 = n0
		case i == n-1:
			n0 = edge(n - 2)
			n1 = n0
		default:
			n0, n1 = edge(i-1), edge(i)
		}
		m := vec{(n0.x + n1.x) / 2, (n0.y + n1.y) / 2}
		if d2 := m.x*m.x + m.y*m.y; d2 > 1e-6 {
			inv := 1 / d2
			if inv > 25 { // острые углы: скос не длиннее 5 пикселей
				inv = 25
			}
			m.x *= inv
			m.y *= inv
		}
		shapeNrm = append(shapeNrm, m)
	}
	return shapeNrm
}

func area(pts []vec) float32 {
	var s float32
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		s += a.x*b.y - b.x*a.y
	}
	return s
}

// fillShape заливает многоугольник (выпуклый или нет) со сглаженным краем.
func fillShape(dst *ebiten.Image, pts []vec, c color.Color) {
	pts = cleanPts(pts, true)
	n := len(pts)
	if n < 3 {
		return
	}
	ar := area(pts)
	if ar == 0 {
		return
	}
	sign := float32(1)
	if ar < 0 {
		sign = -1
	}
	col := premul(c)
	zero := [4]float32{}
	nm := normals(pts, true, sign)
	base := uint32(len(shapeVs))
	// Внутренние точки (на полпикселя внутрь) — сама заливка,
	// внешние (на полпикселя наружу) — прозрачная кромка.
	for i, p := range pts {
		addV(p.x-nm[i].x*0.5, p.y-nm[i].y*0.5, col)
		addV(p.x+nm[i].x*0.5, p.y+nm[i].y*0.5, zero)
	}
	triangulate(pts, sign, func(a, b, c int) {
		shapeIs = append(shapeIs, base+uint32(a*2), base+uint32(b*2), base+uint32(c*2))
	})
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		in0, out0, in1, out1 := base+uint32(i*2), base+uint32(i*2+1), base+uint32(j*2), base+uint32(j*2+1)
		shapeIs = append(shapeIs, in0, in1, out1, in0, out1, out0)
	}
	flushShape(dst)
}

// triangulate режет многоугольник на треугольники «отрезанием ушей».
func triangulate(pts []vec, sign float32, emit func(a, b, c int)) {
	n := len(pts)
	if isConvex(pts, sign) {
		for i := 1; i+1 < n; i++ {
			emit(0, i, i+1)
		}
		return
	}
	earIdx = earIdx[:0]
	for i := 0; i < n; i++ {
		earIdx = append(earIdx, i)
	}
	cross := func(a, b, c vec) float32 { return ((b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)) * sign }
	for guard := 0; len(earIdx) > 3 && guard < n*n; guard++ {
		m := len(earIdx)
		found := false
		for k := 0; k < m; k++ {
			ia, ib, ic := earIdx[(k+m-1)%m], earIdx[k], earIdx[(k+1)%m]
			a, b, c := pts[ia], pts[ib], pts[ic]
			if cross(a, b, c) <= 0 {
				continue
			}
			ok := true
			for _, j := range earIdx {
				if j == ia || j == ib || j == ic {
					continue
				}
				p := pts[j]
				if cross(a, b, p) >= 0 && cross(b, c, p) >= 0 && cross(c, a, p) >= 0 {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			emit(ia, ib, ic)
			earIdx = append(earIdx[:k], earIdx[k+1:]...)
			found = true
			break
		}
		if !found {
			break // самопересечение: остаток заливаем веером
		}
	}
	for i := 1; i+1 < len(earIdx); i++ {
		emit(earIdx[0], earIdx[i], earIdx[i+1])
	}
}

func isConvex(pts []vec, sign float32) bool {
	n := len(pts)
	for i := 0; i < n; i++ {
		a, b, c := pts[i], pts[(i+1)%n], pts[(i+2)%n]
		if ((b.x-a.x)*(c.y-b.y)-(b.y-a.y)*(c.x-b.x))*sign < 0 {
			return false
		}
	}
	return true
}

// strokeShape обводит ломаную линией ширины w со сглаженными краями.
func strokeShape(dst *ebiten.Image, pts []vec, closed bool, w float32, c color.Color) {
	pts = cleanPts(pts, closed)
	n := len(pts)
	if n < 2 || w <= 0 {
		return
	}
	col := premul(c)
	if w < 1 { // тонкая линия: рисуем в пиксель, но бледнее
		for i := range col {
			col[i] *= w
		}
		w = 1
	}
	zero := [4]float32{}
	core := (w - 1) / 2 // полуширина непрозрачной части
	nm := normals(pts, closed, 1)
	base := uint32(len(shapeVs))
	for i, p := range pts {
		d := nm[i]
		addV(p.x+d.x*(core+1), p.y+d.y*(core+1), zero)
		addV(p.x+d.x*core, p.y+d.y*core, col)
		addV(p.x-d.x*core, p.y-d.y*core, col)
		addV(p.x-d.x*(core+1), p.y-d.y*(core+1), zero)
	}
	segs := n - 1
	if closed {
		segs = n
	}
	for i := 0; i < segs; i++ {
		a, b := base+uint32(i*4), base+uint32((i+1)%n*4)
		for k := uint32(0); k < 3; k++ {
			shapeIs = append(shapeIs, a+k, b+k, b+k+1, a+k, b+k+1, a+k+1)
		}
	}
	flushShape(dst)
}

// circlePts — точки окружности; число сторон зависит от радиуса.
func circlePts(x, y, r float32) []vec {
	seg := int(r * 1.2)
	seg = max(12, min(seg, 96))
	shapePts = shapePts[:0]
	for i := 0; i < seg; i++ {
		s, c := math.Sincos(float64(i) * 2 * math.Pi / float64(seg))
		shapePts = append(shapePts, vec{x + r*float32(c), y + r*float32(s)})
	}
	return shapePts
}

// roundRectPts — точки скруглённого прямоугольника (по часовой на экране).
func roundRectPts(x, y, w, h, r float32) []vec {
	shapePts = shapePts[:0]
	r = max(0, min(r, w/2, h/2))
	if r < 0.5 {
		return append(shapePts, vec{x, y}, vec{x + w, y}, vec{x + w, y + h}, vec{x, y + h})
	}
	seg := max(3, min(int(r/2), 12))
	corner := func(cx, cy float32, a0 float64) {
		for i := 0; i <= seg; i++ {
			s, c := math.Sincos(a0 + float64(i)*math.Pi/2/float64(seg))
			shapePts = append(shapePts, vec{cx + r*float32(c), cy + r*float32(s)})
		}
	}
	corner(x+w-r, y+r, -math.Pi/2)
	corner(x+w-r, y+h-r, 0)
	corner(x+r, y+h-r, math.Pi/2)
	corner(x+r, y+r, math.Pi)
	return shapePts
}
