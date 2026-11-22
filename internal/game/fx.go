package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Частицы, всплывающие надписи и тряска экрана — чистая косметика,
// на ход партии не влияют, поэтому могут использовать любой генератор случайностей.

type particle struct {
	x, y, vx, vy float64
	life, max    int
	c            color.RGBA
	size         float64
	glow         bool
	gravity      float64
}

type floater struct {
	s    string
	x, y float64
	life int
	c    color.RGBA
	size int
	big  bool
}

type fxSystem struct {
	parts  []particle
	floats []floater
	shake  float64
	flash  float64
	rng    sim.RNG
}

func newFX() *fxSystem { return &fxSystem{rng: sim.NewRNG(7)} }

func (f *fxSystem) burst(x, y float64, n int, c color.RGBA, speed, size float64, glow bool) {
	for i := 0; i < n; i++ {
		a := f.rng.Float() * 2 * math.Pi
		v := speed * (0.3 + f.rng.Float()*0.7)
		life := 25 + f.rng.Intn(25)
		f.parts = append(f.parts, particle{
			x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v,
			life: life, max: life, c: c, size: size * (0.6 + f.rng.Float()*0.6), glow: glow,
		})
	}
}

func (f *fxSystem) debris(x, y float64, n int, c color.RGBA) {
	for i := 0; i < n; i++ {
		life := 30 + f.rng.Intn(30)
		f.parts = append(f.parts, particle{
			x: x + (f.rng.Float()-0.5)*20, y: y + (f.rng.Float()-0.5)*20,
			vx: (f.rng.Float() - 0.5) * 4, vy: -1 - f.rng.Float()*3,
			life: life, max: life, c: c, size: 2 + f.rng.Float()*4, gravity: 0.25,
		})
	}
}

func (f *fxSystem) text(s string, x, y float64, c color.RGBA, size int) {
	f.floats = append(f.floats, floater{s: s, x: x, y: y, life: 60, c: c, size: size})
}

func (f *fxSystem) update() {
	alive := f.parts[:0]
	for _, p := range f.parts {
		p.life--
		if p.life <= 0 {
			continue
		}
		p.x += p.vx
		p.y += p.vy
		p.vx *= 0.93
		p.vy = p.vy*0.93 + p.gravity
		alive = append(alive, p)
	}
	f.parts = alive
	fl := f.floats[:0]
	for _, t := range f.floats {
		t.life--
		t.y -= 0.8
		if t.life > 0 {
			fl = append(fl, t)
		}
	}
	f.floats = fl
	f.shake *= 0.85
	if f.shake < 0.3 {
		f.shake = 0
	}
	f.flash *= 0.9
}

func (f *fxSystem) offset() (float64, float64) {
	if f.shake == 0 {
		return 0, 0
	}
	return (f.rng.Float() - 0.5) * f.shake, (f.rng.Float() - 0.5) * f.shake
}

func (f *fxSystem) draw(dst *ebiten.Image, ox, oy float64) {
	for _, p := range f.parts {
		a := float64(p.life) / float64(p.max)
		if p.glow {
			glow(dst, p.x+ox, p.y+oy, p.size*4, p.c, a*0.8)
		} else {
			fillRect(dst, p.x+ox-p.size/2, p.y+oy-p.size/2, p.size, p.size, withAlpha(p.c, a))
		}
	}
}

func (f *fxSystem) drawTexts(dst *ebiten.Image, ox, oy float64) {
	for _, t := range f.floats {
		a := math.Min(1, float64(t.life)/20)
		drawTextShadow(dst, t.s, t.size, true, t.x+ox, t.y+oy, Center, withAlpha(t.c, a))
	}
}
