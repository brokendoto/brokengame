// Команда storeart собирает картинки для страницы игры в Steam (капсулы,
// фон библиотеки, логотип) из скриншотов пяти миров в docs/screens.
//
//	go run ./cmd/storeart            # результат в steam/art/
//
// Размеры — по требованиям Steamworks (Store Assets / Library Assets).
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	title    = "Пятимирье"
	subtitle = "пять миров · рекорды · дуэль"
)

// Скриншоты миров: шахта, лес, храм, передовая, космос.
var worldShots = []string{"play.png", "world-forest.png", "world-temple.png", "world-front.png", "world-space.png"}

type asset struct {
	name      string
	w, h      int
	text      bool    // рисовать название
	sub       bool    // и подзаголовок
	titleSize float64 // доля высоты под название
}

var assets = []asset{
	{"header_capsule.png", 920, 430, true, true, 0.24},
	{"small_capsule.png", 462, 174, true, false, 0.36},
	{"main_capsule.png", 1232, 706, true, true, 0.2},
	{"vertical_capsule.png", 748, 896, true, true, 0.1},
	{"library_capsule.png", 600, 900, true, true, 0.085},
	{"library_header.png", 920, 430, true, true, 0.24},
	{"library_hero.png", 3840, 1240, false, false, 0},
	{"page_background.png", 1438, 810, false, false, 0},
}

func main() {
	src := "docs/screens"
	out := filepath.Join("steam", "art")
	if err := os.MkdirAll(out, 0o755); err != nil {
		log.Fatal(err)
	}
	var shots []image.Image
	for _, n := range worldShots {
		img, err := load(filepath.Join(src, n))
		if err != nil {
			log.Fatal(err)
		}
		shots = append(shots, img)
	}
	bold := parse(gobold.TTF)
	reg := parse(goregular.TTF)
	for _, a := range assets {
		img := mosaic(shots, a.w, a.h, a.name == "page_background.png")
		if a.text {
			drawTitle(img, bold, reg, a)
		}
		save(filepath.Join(out, a.name), img)
	}
	// Логотип для библиотеки: название на прозрачном фоне.
	logo := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	drawTitle(logo, bold, reg, asset{w: 1280, h: 720, text: true, sub: false, titleSize: 0.26})
	save(filepath.Join(out, "library_logo.png"), logo)
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(path, img.Bounds().Dx(), "x", img.Bounds().Dy())
}

func parse(ttf []byte) *opentype.Font {
	f, err := opentype.Parse(ttf)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

// focus — какой участок скриншота показывать в полосе мира: центр участка
// (в пикселях скриншота 1280×720) и яркость. Участки выбраны в обход
// надписей, которые игра показывает в начале уровня.
type focus struct{ x, y, bright float64 }

var focuses = []focus{
	{640, 260, 1.5}, // шахта темнее остальных
	{210, 400, 1},
	{1060, 330, 1},
	{210, 400, 1},
	{1060, 330, 1},
}

// mosaic режет холст на пять наклонных полос — по одной на мир — и
// заполняет каждую своим участком скриншота. Затем затемняет края, чтобы
// название читалось.
func mosaic(shots []image.Image, w, h int, dim bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	n := float64(len(shots))
	region := 360.0 // высота участка скриншота, которая ложится на холст
	if w < h {
		region = 560 // в вертикальных капсулах полосы узкие: берём участок крупнее
	}
	sc := float64(h) / region
	slope := 0.35
	span := float64(w) + float64(h)*slope
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := (float64(x) + (float64(h)-float64(y))*slope) / span
			i := min(len(shots)-1, int(t*n))
			// Центр полосы на средней строке холста.
			cx := (float64(i)+0.5)/n*span - float64(h)*slope/2
			f := focuses[i]
			sx := f.x + (float64(x)-cx)/sc
			sy := f.y + (float64(y)-float64(h)/2)/sc
			sy = math.Max(92, math.Min(708, sy)) // без панели интерфейса сверху
			r, g, b := bilinear(shots[i], sx, sy)
			r, g, b = r*f.bright, g*f.bright, b*f.bright
			// Тонкая светлая граница между полосами.
			edge := math.Abs(t*n - math.Round(t*n))
			if edge*span/n < 2.5*float64(w)/920 && t > 0.01 && t < 0.99 {
				r, g, b = r*0.4+255*0.6, g*0.4+220*0.6, b*0.4+150*0.6
			}
			// Виньетка.
			dx := (float64(x)/float64(w) - 0.5) * 2
			dy := (float64(y)/float64(h) - 0.5) * 2
			k := 1 - 0.45*math.Min(1, (dx*dx+dy*dy)/2)
			if dim {
				k *= 0.55
			}
			img.SetRGBA(x, y, color.RGBA{clamp(r * k), clamp(g * k), clamp(b * k), 255})
		}
	}
	return img
}

func bilinear(img image.Image, x, y float64) (r, g, b float64) {
	bd := img.Bounds()
	x = math.Max(0, math.Min(float64(bd.Dx()-2), x))
	y = math.Max(0, math.Min(float64(bd.Dy()-2), y))
	x0, y0 := int(x), int(y)
	fx, fy := x-float64(x0), y-float64(y0)
	px := func(xx, yy int) (float64, float64, float64) {
		cr, cg, cb, _ := img.At(bd.Min.X+xx, bd.Min.Y+yy).RGBA()
		return float64(cr >> 8), float64(cg >> 8), float64(cb >> 8)
	}
	r00, g00, b00 := px(x0, y0)
	r10, g10, b10 := px(x0+1, y0)
	r01, g01, b01 := px(x0, y0+1)
	r11, g11, b11 := px(x0+1, y0+1)
	lerp := func(a, b, c, d float64) float64 {
		return (a*(1-fx)+b*fx)*(1-fy) + (c*(1-fx)+d*fx)*fy
	}
	return lerp(r00, r10, r01, r11), lerp(g00, g10, g01, g11), lerp(b00, b10, b01, b11)
}

func clamp(v float64) uint8 { return uint8(math.Max(0, math.Min(255, v))) }

// drawTitle пишет название по центру (в вертикальных капсулах — в нижней
// трети) с тенью и подложкой-полосой.
func drawTitle(img *image.RGBA, bold, reg *opentype.Font, a asset) {
	size := float64(a.h) * a.titleSize
	if a.w < a.h {
		size = float64(a.w) * 0.17
	}
	face := newFace(bold, size)
	tw := measure(face, title)
	if limit := float64(a.w) * 0.9; tw > limit {
		size *= limit / tw
		face = newFace(bold, size)
		tw = measure(face, title)
	}
	cy := float64(a.h) * 0.5
	if a.w < a.h {
		cy = float64(a.h) * 0.72
	}
	subSize := size * 0.32
	if a.sub {
		cy -= subSize * 0.6
	}
	// Полупрозрачная полоса под текстом.
	band := size * 0.85
	if a.sub {
		band += subSize * 1.3
	}
	if a.name != "" {
		top := cy - size*0.62
		for y := int(top); y < int(top+band+size*0.2); y++ {
			for x := 0; x < a.w; x++ {
				if y < 0 || y >= a.h {
					continue
				}
				c := img.RGBAAt(x, y)
				const k = 0.45
				img.SetRGBA(x, y, color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k), uint8(float64(c.B) * k), 255})
			}
		}
	}
	base := cy + size*0.35
	x := (float64(a.w) - tw) / 2
	sh := math.Max(2, size*0.05)
	text(img, face, title, x+sh, base+sh, color.RGBA{0, 0, 0, 200})
	text(img, face, title, x, base, color.RGBA{255, 214, 96, 255})
	if a.sub {
		sf := newFace(reg, subSize)
		sw := measure(sf, subtitle)
		text(img, sf, subtitle, (float64(a.w)-sw)/2, base+subSize*1.35, color.RGBA{236, 230, 245, 255})
	}
}

func newFace(f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		log.Fatal(err)
	}
	return face
}

func measure(face font.Face, s string) float64 {
	return float64(font.MeasureString(face, s)) / 64
}

func text(img *image.RGBA, face font.Face, s string, x, y float64, c color.RGBA) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.Point26_6{X: fixed.I(0) + fixed.Int26_6(x*64), Y: fixed.Int26_6(y * 64)}}
	d.DrawString(s)
}
