package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/sfx"
	"github.com/brokendoto/brokengame/internal/sim"
)

// --- Выбор героя ---------------------------------------------------------------
//
// Все пять героев открыты сразу. «Свой в каждом мире» — как раньше: в шахте
// шахтёр, в лесу белочка и т. д. Любой другой выбор — этот герой во всех мирах.
// Выбор меняет только внешний вид: способность (щит, магнит…) зависит от мира.

func skinName(skin int) string {
	if skin <= 0 || skin > int(sim.WorldCount) {
		return "свой в каждом мире"
	}
	return sim.Worlds[skin-1].Hero
}

type skinScene struct {
	sel  int // 0 — «свой в каждом мире», 1..5 — герои
	nav  nav
	hits hits
	t    int
	big  *ebiten.Image
}

const (
	skinCount   = int(sim.WorldCount) + 1
	hitSkinBack = 900
)

func newSkinScene(a *App) *skinScene {
	return &skinScene{sel: a.Prof.Skin, big: ebiten.NewImage(heroBufW, heroBufH)}
}

func (s *skinScene) choose(a *App, i int) {
	a.Prof.Skin = i
	a.applySettings()
	_ = a.Prof.Save()
	a.Sound.Play(sfx.Select)
	a.Toast("Герой: " + skinName(i))
	a.Goto(newMenuScene())
}

func (s *skinScene) Update(a *App) {
	s.t++
	if s.nav.dir(navLeft) || s.nav.dir(navUp) {
		s.sel = (s.sel + skinCount - 1) % skinCount
		a.Sound.Play(sfx.Menu)
	}
	if s.nav.dir(navRight) || s.nav.dir(navDown) {
		s.sel = (s.sel + 1) % skinCount
		a.Sound.Play(sfx.Menu)
	}
	if backPressed() {
		a.Goto(newMenuScene())
		return
	}
	if id, ok := s.hits.tapped(a); ok {
		if id == hitSkinBack {
			a.Goto(newMenuScene())
			return
		}
		s.choose(a, id)
		return
	}
	if okPressed() {
		s.choose(a, s.sel)
	}
}

// skinCard — где на экране карточка героя i.
func skinCard(i int) (x, y, w, h float64) {
	w, h = 186, 300
	gap := 14.0
	total := float64(skinCount)*w + float64(skinCount-1)*gap
	x = (ScreenW-total)/2 + float64(i)*(w+gap)
	return x, 170, w, h
}

func (s *skinScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawTextShadow(dst, "Выбор героя", 44, true, ScreenW/2, 40, Center, colText)
	drawText(dst, "Все герои открыты. Способность зависит от мира, а не от героя.", 20, false, ScreenW/2, 108, Center, colDim)
	s.hits.reset()
	t := float64(s.t)
	for i := 0; i < skinCount; i++ {
		x, y, w, h := skinCard(i)
		sel := i == s.sel
		s.hits.add(rect{x, y, w, h}, i)
		drawPanel(dst, x, y, w, h)
		if sel {
			roundRectStroke(dst, x-3, y-3, w+6, h+6, 18, 4, colAccent)
			glow(dst, x+w/2, y+h/2, w*0.8, colAccent, 0.18)
		}
		if i == a.Prof.Skin {
			drawText(dst, "выбран", 15, true, x+w/2, y+12, Center, colGreen)
		}
		cx, cy := x+w/2, y+128
		if i == 0 {
			// Все пять по кругу — «каждому миру свой».
			for k := 0; k < int(sim.WorldCount); k++ {
				ang := t*0.01 + float64(k)*2*math.Pi/float64(sim.WorldCount)
				s.drawBig(dst, sim.World(k), cx+math.Cos(ang)*52, cy+math.Sin(ang)*40, t+float64(k)*20, 1.1, sel)
			}
			drawText(dst, "Свой в", 22, true, cx, y+222, Center, colText)
			drawText(dst, "каждом мире", 22, true, cx, y+250, Center, colText)
			continue
		}
		w0 := sim.World(i - 1)
		s.drawBig(dst, w0, cx, cy, t, 2.2, sel)
		drawText(dst, sim.Worlds[w0].Hero, 24, true, cx, y+222, Center, colText)
		drawText(dst, themes[w0].short, 17, false, cx, y+256, Center, colDim)
	}
	bx, by := ScreenW/2-110.0, 520.0
	s.hits.add(rect{bx, by, 220, 48}, hitSkinBack)
	roundRect(dst, bx, by, 220, 48, 12, rgba(50, 40, 76, 230))
	drawText(dst, "Назад", 24, false, ScreenW/2, by+10, Center, colText)
	if a.Pointer.touchMode {
		keyHint(dst, "Нажми на героя, чтобы выбрать")
	} else {
		keyHint(dst, "←→ — выбор   ·   Enter — выбрать   ·   Esc — назад")
	}
}

// drawBig рисует героя крупно: шагает, если карточка выбрана.
func (s *skinScene) drawBig(dst *ebiten.Image, w sim.World, x, y, t, scale float64, walking bool) {
	p := sim.Player{FX: 0.3, FY: 1}
	if walking {
		p.Moving, p.WalkDistance = true, t*0.012
		p.FX = math.Sin(t*0.01) * 1
	}
	s.big.Clear()
	cx, cy := float64(heroBufW)/2, float64(heroBufH)/2+6
	drawChibi(s.big, w, p, cx, cy, t, false)
	fillPoly(dst, ellipse(x+1, y+20*scale/1.2, 15*scale, 5*scale, 0, 18), rgba(0, 0, 0, 70))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-cx, -cy)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	dst.DrawImage(s.big, op)
}

// heroLabel — кто играет в мире w: выбранный герой или герой мира.
func heroLabel(w sim.World) string {
	if heroSkin > 0 && heroSkin <= int(sim.WorldCount) {
		return sim.Worlds[heroSkin-1].Hero
	}
	return sim.Worlds[w].Hero
}
