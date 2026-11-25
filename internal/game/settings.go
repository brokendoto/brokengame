package game

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/controls"
	"github.com/brokendoto/brokengame/internal/sfx"
)

// applySettings переносит настройки профиля в звук, эффекты и управление.
func (a *App) applySettings() {
	a.Sound.Volume = float64(a.Prof.Volume) / 100
	screenShake = a.Prof.Shake
	setLayouts(controls.FromMap(a.Prof.Keys, keyValid))
}

// --- Настройки -------------------------------------------------------------------

type setItem int

const (
	siVolume setItem = iota
	siMusic
	siSound
	siShake
	siGhost
	siFullscreen
	siKeysSolo
	siKeysDuel1
	siKeysDuel2
	siKeysReset
	siBack
)

type settingsScene struct {
	items []setItem
	sel   int
	nav   nav
	hits  hits
}

func newSettingsScene() *settingsScene {
	s := &settingsScene{items: []setItem{siVolume, siMusic, siSound, siShake, siGhost}}
	if !IsMobile() {
		// В приложении на телефоне нет полного экрана, а клавиатуры нет
		// ни там, ни в браузере на телефоне.
		s.items = append(s.items, siFullscreen)
		if !touchFirst {
			s.items = append(s.items, siKeysSolo, siKeysDuel1, siKeysDuel2, siKeysReset)
		}
	}
	s.items = append(s.items, siBack)
	return s
}

func onOff(b bool) string {
	if b {
		return "вкл"
	}
	return "выкл"
}

func (s *settingsScene) label(a *App, it setItem) string {
	switch it {
	case siVolume:
		return "Громкость:  ‹ " + itoa(a.Prof.Volume) + "% ›"
	case siMusic:
		return "Музыка:  ‹ " + itoa(a.Prof.Music) + "% ›"
	case siSound:
		return "Звук: " + onOff(!a.Sound.Muted)
	case siShake:
		return "Тряска экрана и вспышки: " + onOff(a.Prof.Shake)
	case siGhost:
		return "Призрак соперника: " + onOff(a.Prof.ShowGhost)
	case siFullscreen:
		return "Полный экран: " + onOff(ebiten.IsFullscreen())
	case siKeysSolo:
		return "Клавиши: одиночная игра  ›"
	case siKeysDuel1:
		return "Клавиши: дуэль, игрок 1  ›"
	case siKeysDuel2:
		return "Клавиши: дуэль, игрок 2  ›"
	case siKeysReset:
		return "Вернуть клавиши по умолчанию"
	case siBack:
		return "Назад"
	}
	return ""
}

func (s *settingsScene) Update(a *App) {
	n := len(s.items)
	if s.nav.dir(navUp) {
		s.sel = (s.sel + n - 1) % n
		a.Sound.Play(sfx.Menu)
	}
	if s.nav.dir(navDown) {
		s.sel = (s.sel + 1) % n
		a.Sound.Play(sfx.Menu)
	}
	it := s.items[s.sel]
	if it == siVolume || it == siMusic {
		d := 0
		if s.nav.dir(navLeft) {
			d = -10
		}
		if s.nav.dir(navRight) {
			d = 10
		}
		if d != 0 && it == siVolume {
			s.setVolume(a, a.Prof.Volume+d)
		} else if d != 0 {
			s.setMusic(a, a.Prof.Music+d)
		}
	}
	if backPressed() {
		a.Goto(newMenuScene())
		return
	}
	activate := okPressed()
	if id, ok := s.hits.tapped(a); ok {
		s.sel, activate = id, true
		it = s.items[id]
	}
	if !activate {
		return
	}
	a.Sound.Play(sfx.Select)
	switch it {
	case siVolume:
		// По нажатию громкость идёт по кругу: 0, 10, … 100.
		v := a.Prof.Volume + 10
		if v > 100 {
			v = 0
		}
		s.setVolume(a, v)
	case siMusic:
		v := a.Prof.Music + 10
		if v > 100 {
			v = 0
		}
		s.setMusic(a, v)
	case siSound:
		a.SetMuted(!a.Sound.Muted)
	case siShake:
		a.Prof.Shake = !a.Prof.Shake
		a.applySettings()
		_ = a.Prof.Save()
	case siGhost:
		a.Prof.ShowGhost = !a.Prof.ShowGhost
		_ = a.Prof.Save()
	case siFullscreen:
		a.SetFullscreen(!ebiten.IsFullscreen())
	case siKeysSolo:
		a.Goto(newBindScene(controls.Solo))
	case siKeysDuel1:
		a.Goto(newBindScene(controls.Duel1))
	case siKeysDuel2:
		a.Goto(newBindScene(controls.Duel2))
	case siKeysReset:
		a.Prof.Keys = nil
		a.applySettings()
		_ = a.Prof.Save()
		a.Toast("Клавиши сброшены")
	case siBack:
		a.Goto(newMenuScene())
	}
}

func (s *settingsScene) setVolume(a *App, v int) {
	a.Prof.Volume = min(100, max(0, v))
	a.applySettings()
	_ = a.Prof.Save()
	a.Sound.Play(sfx.Menu)
}

func (s *settingsScene) setMusic(a *App, v int) {
	a.Prof.Music = min(100, max(0, v))
	_ = a.Prof.Save()
	a.Sound.Play(sfx.Menu)
}

func (s *settingsScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	drawTextShadow(dst, "Настройки", 44, true, ScreenW/2, 40, Center, colText)
	s.hits.reset()
	y0 := 140.0
	for i, it := range s.items {
		y := y0 + float64(i)*46
		if it == siKeysSolo || it == siBack {
			y0 += 16 // отступ перед группой
			y += 16
		}
		s.hits.add(rect{ScreenW/2 - 320, y - 8, 640, 44}, i)
		sel := i == s.sel
		if sel {
			w := textWidth(s.label(a, it), 24, true) + 60
			roundRect(dst, ScreenW/2-w/2, y-6, w, 40, 12, rgba(60, 46, 96, 230))
		}
		c := colDim
		if sel {
			c = colText
		}
		drawText(dst, s.label(a, it), 24, sel, ScreenW/2, y, Center, c)
	}
	if a.Pointer.touchMode {
		keyHint(dst, "Нажми на пункт, чтобы изменить")
	} else {
		keyHint(dst, "↑↓ — выбор   ·   Enter — изменить   ·   ←→ — громкость   ·   Esc — назад")
	}
}

// --- Переназначение клавиш ---------------------------------------------------------

// Клавиши, которые игра использует сама: их назначать нельзя.
var reservedKeys = map[ebiten.Key]string{
	ebiten.KeyEscape: "пауза", ebiten.KeyP: "пауза", ebiten.KeyR: "рестарт",
	ebiten.KeyM: "звук", ebiten.KeyF9: "автопилот", ebiten.KeyF11: "полный экран",
	ebiten.KeyBackspace: "очистка", ebiten.KeyDelete: "очистка",
}

type bindScene struct {
	layout  int
	row     int // действие или строка «Сбросить»
	col     int // ячейка 0/1
	capture int // >0 — ждём клавишу (счётчик кадров)
	nav     nav
	hits    hits
	keys    []ebiten.Key
}

const (
	hitBindReset = 600 + iota
	hitBindBack
)

func newBindScene(l int) *bindScene { return &bindScene{layout: l} }

func (b *bindScene) save(a *App) {
	a.Prof.Keys = controls.ToMap(layouts)
	_ = a.Prof.Save()
}

func (b *bindScene) Update(a *App) {
	if b.capture > 0 {
		b.capture++
		if b.capture < 3 {
			return // пропускаем Enter, которым начали ввод
		}
		b.keys = inpututil.AppendJustPressedKeys(b.keys[:0])
		for _, k := range b.keys {
			all := layouts
			switch {
			case k == ebiten.KeyEscape:
				b.capture = 0
			case k == ebiten.KeyBackspace || k == ebiten.KeyDelete:
				controls.Set(&all, b.layout, controls.Action(b.row), b.col, "")
				setLayouts(all)
				b.save(a)
				b.capture = 0
			case reservedKeys[k] != "":
				a.Toast("Клавиша " + keyLabel(k.String()) + " уже занята игрой: " + reservedKeys[k])
				a.Sound.Play(sfx.Block)
			default:
				taken := controls.Set(&all, b.layout, controls.Action(b.row), b.col, k.String())
				setLayouts(all)
				b.save(a)
				a.Sound.Play(sfx.Select)
				if taken != "" {
					a.Toast(keyLabel(k.String()) + " снята с «" + taken + "»")
				}
				b.capture = 0
			}
			if b.capture == 0 {
				break
			}
		}
		return
	}
	rows := int(controls.ActionCount) + 1
	if b.nav.dir(navUp) {
		b.row = (b.row + rows - 1) % rows
		a.Sound.Play(sfx.Menu)
	}
	if b.nav.dir(navDown) {
		b.row = (b.row + 1) % rows
		a.Sound.Play(sfx.Menu)
	}
	if b.nav.dir(navLeft) || b.nav.dir(navRight) {
		b.col = 1 - b.col
		a.Sound.Play(sfx.Menu)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		b.layout = (b.layout + 1) % controls.LayoutCount
		a.Sound.Play(sfx.Menu)
	}
	activate := okPressed()
	if id, ok := b.hits.tapped(a); ok {
		switch {
		case id == hitBindBack:
			b.leave(a)
			return
		case id == hitBindReset:
			b.row, activate = int(controls.ActionCount), true
		case id >= 700:
			b.layout = id - 700
			a.Sound.Play(sfx.Menu)
		default:
			b.row, b.col, activate = id/2, id%2, true
		}
	}
	if backPressed() && !inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		b.leave(a)
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) && b.row < int(controls.ActionCount) {
		all := layouts
		controls.Set(&all, b.layout, controls.Action(b.row), b.col, "")
		setLayouts(all)
		b.save(a)
	}
	if !activate {
		return
	}
	if b.row == int(controls.ActionCount) {
		all := layouts
		all[b.layout] = controls.Defaults()[b.layout]
		// Сброс одного игрока дуэли не должен отдать ему клавиши другого.
		for act := controls.Action(0); act < controls.ActionCount; act++ {
			for sl := 0; sl < controls.Slots; sl++ {
				if k := all[b.layout][act][sl]; k != "" {
					controls.Set(&all, b.layout, act, sl, k)
				}
			}
		}
		setLayouts(all)
		b.save(a)
		a.Sound.Play(sfx.Select)
		a.Toast("Раскладка «" + controls.LayoutNames[b.layout] + "» сброшена")
		return
	}
	b.capture = 1
	a.Sound.Play(sfx.Select)
}

func (b *bindScene) leave(a *App) {
	for l := 0; l < controls.LayoutCount; l++ {
		if e := layouts[l].Empty(); len(e) > 0 {
			var names []string
			for _, act := range e {
				names = append(names, controls.ActionNames[act])
			}
			a.Toast("Внимание: в раскладке «" + controls.LayoutNames[l] + "» нет клавиш для: " + strings.Join(names, ", "))
			break
		}
	}
	a.Goto(newSettingsScene())
}

func (b *bindScene) Draw(a *App, dst *ebiten.Image) {
	a.drawBackground(dst)
	b.hits.reset()
	drawTextShadow(dst, "Управление", 40, true, ScreenW/2, 24, Center, colText)
	// Вкладки раскладок.
	for l := 0; l < controls.LayoutCount; l++ {
		x := ScreenW/2 + float64(l-1)*300
		sel := l == b.layout
		bg := rgba(30, 26, 40, 230)
		c := colDim
		if sel {
			bg, c = rgba(60, 46, 96, 240), colText
		}
		roundRect(dst, x-140, 86, 280, 44, 12, bg)
		if sel {
			roundRectStroke(dst, x-140, 86, 280, 44, 12, 2, colAccent)
		}
		drawText(dst, controls.LayoutNames[l], 19, sel, x, 96, Center, c)
		b.hits.add(rect{x - 140, 82, 280, 52}, 700+l)
	}
	lay := &layouts[b.layout]
	drawText(dst, "Клавиша 1", 16, false, 700, 154, Center, colDim)
	drawText(dst, "Клавиша 2", 16, false, 960, 154, Center, colDim)
	for r := 0; r <= int(controls.ActionCount); r++ {
		y := 182 + float64(r)*56
		if r == int(controls.ActionCount) {
			sel := b.row == r
			c := colDim
			if sel {
				c = colText
				roundRect(dst, ScreenW/2-200, y+10, 400, 42, 10, rgba(60, 46, 96, 240))
			}
			drawText(dst, "Вернуть эту раскладку по умолчанию", 20, sel, ScreenW/2, y+18, Center, c)
			b.hits.add(rect{ScreenW/2 - 220, y + 4, 440, 54}, hitBindReset)
			break
		}
		drawText(dst, controls.ActionNames[r], 22, b.row == r, 520, y+8, Right, colText)
		for sl := 0; sl < controls.Slots; sl++ {
			x := 590 + float64(sl)*260
			sel := b.row == r && b.col == sl
			bg := rgba(32, 27, 42, 235)
			if sel {
				bg = rgba(62, 48, 100, 245)
			}
			roundRect(dst, x, y, 220, 44, 10, bg)
			if sel {
				roundRectStroke(dst, x, y, 220, 44, 10, 2, colAccent)
			}
			label := keyLabel(lay[r][sl])
			lc := colText
			if label == "" {
				label, lc = "—", colDim
			}
			if sel && b.capture > 0 {
				label, lc = "нажми клавишу…", colGold
			}
			drawText(dst, label, 20, true, x+110, y+9, Center, lc)
			b.hits.add(rect{x, y - 4, 220, 52}, r*2+sl)
		}
	}
	drawText(dst, "Геймпад: стик или крестовина, A/RB — рывок, X/LB — способность", 17, false, ScreenW/2, 590, Center, colDim)
	if a.Pointer.touchMode {
		drawBackButton(dst, &b.hits, hitBindBack)
	}
	if b.capture > 0 {
		keyHint(dst, "Нажми новую клавишу   ·   Backspace — очистить   ·   Esc — отмена")
	} else {
		keyHint(dst, "Стрелки — выбор   ·   Enter — назначить   ·   Del — очистить   ·   Tab — другая раскладка   ·   Esc — назад")
	}
}
