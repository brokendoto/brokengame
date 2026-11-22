package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/achieve"
	"github.com/brokendoto/brokengame/internal/sfx"
)

const achShowTicks = 200 // сколько висит всплывающее уведомление

// Grant выдаёт достижения: новые сохраняются в профиле и показываются.
func (a *App) Grant(ids ...achieve.ID) {
	fresh := achieve.Unlock(a.Prof.Achievements, ids, today())
	if len(fresh) == 0 {
		return
	}
	_ = a.Prof.Save()
	if len(a.achQueue) == 0 {
		a.achT = 0
		a.Sound.Play(sfx.Win)
	}
	a.achQueue = append(a.achQueue, fresh...)
}

func (a *App) drawAchievementPopup(dst *ebiten.Image) {
	if len(a.achQueue) == 0 {
		return
	}
	info, _ := achieve.ByID(a.achQueue[0])
	// Выезжает справа сверху и уезжает обратно.
	k := math.Min(1, math.Min(float64(a.achT)/15, float64(achShowTicks-a.achT)/15))
	w := math.Max(380, math.Max(textWidth(info.Name, 22, true), textWidth(info.Desc, 16, false))+110)
	x := ScreenW - 16 - w*k
	y := 92.0
	roundRect(dst, x, y, w, 76, 14, rgba(30, 24, 44, 245))
	roundRectStroke(dst, x, y, w, 76, 14, 2, colGold)
	glow(dst, x+44, y+38, 60, colGold, 0.25)
	drawTrophy(dst, x+44, y+38, 1, true)
	drawText(dst, "Достижение получено!", 14, false, x+84, y+10, Left, colGold)
	drawText(dst, info.Name, 22, true, x+84, y+26, Left, colText)
	drawText(dst, info.Desc, 15, false, x+84, y+52, Left, colDim)
}

// drawTrophy рисует кубок; got=false — серый, ещё не получен.
func drawTrophy(dst *ebiten.Image, x, y, s float64, got bool) {
	c, hi := colGold, rgba(255, 240, 170, 255)
	if !got {
		c, hi = rgba(80, 74, 92, 255), rgba(110, 104, 122, 255)
	}
	strokeCircle(dst, x-13*s, y-8*s, 7*s, 3*s, c)
	strokeCircle(dst, x+13*s, y-8*s, 7*s, 3*s, c)
	fillPoly(dst, []float64{x - 14*s, y - 18*s, x + 14*s, y - 18*s, x + 9*s, y + 2*s, x - 9*s, y + 2*s}, c)
	fillCircle(dst, x, y+1*s, 9*s, c)
	fillRect(dst, x-3*s, y+8*s, 6*s, 8*s, c)
	fillRect(dst, x-11*s, y+15*s, 22*s, 5*s, c)
	fillRect(dst, x-8*s, y-15*s, 4*s, 10*s, hi)
}

// --- Экран достижений ------------------------------------------------------------

type achievementsScene struct {
	hits hits
}

func newAchievementsScene() *achievementsScene { return &achievementsScene{} }

func (s *achievementsScene) Update(a *App) {
	if id, ok := s.hits.tapped(a); (ok && id == hitBack) || backPressed() || okPressed() {
		a.Sound.Play(sfx.Select)
		a.Goto(newMenuScene())
	}
}

func (s *achievementsScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	s.hits.reset()
	got := a.Prof.Achievements
	drawTextShadow(dst, "Достижения", 40, true, ScreenW/2, 14, Center, colText)
	drawText(dst, "Получено "+itoa(len(got))+" из "+itoa(len(achieve.List)), 18, false, ScreenW/2, 64, Center, colGold)
	const colW, rowH = 590.0, 56.0
	perCol := (len(achieve.List) + 1) / 2
	for i, info := range achieve.List {
		x := 40 + float64(i/perCol)*(colW+20)
		y := 98 + float64(i%perCol)*rowH
		date, has := got[string(info.ID)]
		bg := rgba(30, 26, 40, 220)
		if has {
			bg = rgba(52, 42, 30, 235)
		}
		roundRect(dst, x, y, colW, rowH-6, 10, bg)
		drawTrophy(dst, x+30, y+24, 0.8, has)
		nc, desc := colDim, info.Desc
		if has {
			nc = colText
		} else if info.Hidden {
			desc = "Секретное достижение"
		}
		drawText(dst, info.Name, 19, true, x+60, y+4, Left, nc)
		drawText(dst, desc, 14, false, x+60, y+27, Left, colDim)
		right := ""
		if has {
			right = date
		} else if cur, total, ok := achieve.Progress(info.ID, a.Prof.Medals); ok {
			right = itoa(cur) + " / " + itoa(total)
		}
		drawText(dst, right, 14, false, x+colW-14, y+16, Right, colGold)
	}
	if a.Pointer.touchMode {
		drawBackButton(dst, &s.hits, hitBack)
	} else {
		keyHint(dst, "Esc — назад")
	}
}
