package game

import (
	"bytes"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// Логический размер экрана. Ebitengine сам масштабирует его под окно
// и полноэкранный режим любого разрешения, вплоть до 4K.
const (
	ScreenW = 1280
	ScreenH = 720
)

// Шрифты Go (лицензия BSD) — содержат кириллицу.
var (
	regularSrc, boldSrc *text.GoTextFaceSource
	fontCache           = map[[2]int]*text.GoTextFace{}
)

func init() {
	var err error
	regularSrc, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	boldSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
}

func face(size int, bold bool) *text.GoTextFace {
	k := [2]int{size, 0}
	src := regularSrc
	if bold {
		k[1] = 1
		src = boldSrc
	}
	if f, ok := fontCache[k]; ok {
		return f
	}
	f := &text.GoTextFace{Source: src, Size: float64(size)}
	fontCache[k] = f
	return f
}

// Align — выравнивание текста по горизонтали.
type Align int

const (
	Left Align = iota
	Center
	Right
)

// drawText рисует текст; y — верх строки.
func drawText(dst *ebiten.Image, s string, size int, bold bool, x, y float64, a Align, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	switch a {
	case Center:
		op.PrimaryAlign = text.AlignCenter
	case Right:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(dst, s, face(size, bold), op)
}

// drawTextShadow — текст с тенью для читаемости поверх поля.
func drawTextShadow(dst *ebiten.Image, s string, size int, bold bool, x, y float64, a Align, c color.Color) {
	drawText(dst, s, size, bold, x+2, y+2, a, color.RGBA{0, 0, 0, 160})
	drawText(dst, s, size, bold, x, y, a, c)
}

func textWidth(s string, size int, bold bool) float64 {
	w, _ := text.Measure(s, face(size, bold), 0)
	return w
}

type rgbaT = color.RGBA

func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

// withAlpha возвращает цвет с умноженной прозрачностью (премультиплицированный).
func withAlpha(c color.RGBA, a float64) color.RGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func fillCircle(dst *ebiten.Image, x, y, r float64, c color.Color) {
	vector.FillCircle(dst, float32(x), float32(y), float32(r), c, true)
}

func strokeCircle(dst *ebiten.Image, x, y, r, w float64, c color.Color) {
	vector.StrokeCircle(dst, float32(x), float32(y), float32(r), float32(w), c, true)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.Color) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), float32(w), c, true)
}

func colorScale(c color.Color) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	return cs
}

// fillPoly заливает многоугольник.
func fillPoly(dst *ebiten.Image, pts []float64, c color.Color) {
	var p vector.Path
	for i := 0; i+1 < len(pts); i += 2 {
		if i == 0 {
			p.MoveTo(float32(pts[i]), float32(pts[i+1]))
		} else {
			p.LineTo(float32(pts[i]), float32(pts[i+1]))
		}
	}
	p.Close()
	vector.FillPath(dst, &p, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(c)})
}

// roundRect — прямоугольник со скруглёнными углами.
func roundRect(dst *ebiten.Image, x, y, w, h, r float64, c color.Color) {
	var p vector.Path
	rr := float32(r)
	x0, y0, x1, y1 := float32(x), float32(y), float32(x+w), float32(y+h)
	p.MoveTo(x0+rr, y0)
	p.LineTo(x1-rr, y0)
	p.ArcTo(x1, y0, x1, y0+rr, rr)
	p.LineTo(x1, y1-rr)
	p.ArcTo(x1, y1, x1-rr, y1, rr)
	p.LineTo(x0+rr, y1)
	p.ArcTo(x0, y1, x0, y1-rr, rr)
	p.LineTo(x0, y0+rr)
	p.ArcTo(x0, y0, x0+rr, y0, rr)
	p.Close()
	vector.FillPath(dst, &p, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(c)})
}

// roundRectStroke — обводка скруглённого прямоугольника.
func roundRectStroke(dst *ebiten.Image, x, y, w, h, r, sw float64, c color.Color) {
	var p vector.Path
	rr := float32(r)
	x0, y0, x1, y1 := float32(x), float32(y), float32(x+w), float32(y+h)
	p.MoveTo(x0+rr, y0)
	p.LineTo(x1-rr, y0)
	p.ArcTo(x1, y0, x1, y0+rr, rr)
	p.LineTo(x1, y1-rr)
	p.ArcTo(x1, y1, x1-rr, y1, rr)
	p.LineTo(x0+rr, y1)
	p.ArcTo(x0, y1, x0, y1-rr, rr)
	p.LineTo(x0, y0+rr)
	p.ArcTo(x0, y0, x0+rr, y0, rr)
	p.Close()
	vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(sw), LineJoin: vector.LineJoinRound},
		&vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(c)})
}

// Спрайт мягкого свечения: радиальный градиент, рисуется аддитивно.
var glowImg = func() *ebiten.Image {
	const n = 64
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
			d := math.Sqrt(dx*dx + dy*dy)
			a := math.Max(0, 1-d)
			a = a * a
			v := byte(a * 255)
			i := (y*n + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, v
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}()

// glow рисует свечение радиуса r цвета c (аддитивно).
func glow(dst *ebiten.Image, x, y, r float64, c color.RGBA, intensity float64) {
	op := &ebiten.DrawImageOptions{}
	s := r * 2 / 64
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.ColorScale.ScaleAlpha(float32(intensity))
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(glowImg, op)
}

// Маска темноты: прозрачный центр, тёмные края. Масштабируется под фонарь.
var darkImg = func() *ebiten.Image {
	const n = 256
	pix := make([]byte, n*n*4)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)+0.5)/n*2-1, (float64(y)+0.5)/n*2-1
			d := math.Sqrt(dx*dx + dy*dy)
			// Радиус картинки ≈ 1400 px на экране: свет фонаря ~ 4 клетки.
			a := math.Min(1, math.Max(0, (d-0.06)/0.30))
			a = a * a * (3 - 2*a)
			i := (y*n + x) * 4
			pix[i+3] = byte(a * 255)
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}()

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// fmtTime форматирует шаги как М:СС.д.
func fmtTime(ticks int) string {
	if ticks < 0 {
		ticks = 0
	}
	tenths := ticks * 10 / 60
	return itoa(tenths/600) + ":" + pad2(tenths/10%60) + "." + itoa(tenths%10)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// fmtScore — число с разделителем тысяч: 12 345.
func fmtScore(n int) string {
	s := itoa(n)
	neg := false
	if n < 0 {
		neg, s = true, s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
