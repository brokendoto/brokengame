//go:build js

package game

import "syscall/js"

// touchScreen — открыта ли веб-версия на телефоне или планшете: основной
// указатель «грубый» (палец), а не мышь. Тогда экранные кнопки и клавиатура
// для имени видны сразу, ещё до первого касания.
func touchScreen() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	mm := js.Global().Get("matchMedia")
	if mm.Truthy() && js.Global().Call("matchMedia", "(pointer: coarse)").Get("matches").Bool() {
		return true
	}
	return js.Global().Get("navigator").Get("maxTouchPoints").Int() > 0 &&
		!js.Global().Call("matchMedia", "(hover: hover)").Get("matches").Bool()
}

// Быстрые нажатия. Ebitengine смотрит на мышь и пальцы раз в кадр, и если
// кнопку нажали и отпустили между двумя кадрами (тап по тачпаду ноутбука,
// быстрый тап пальцем, телефон, который тянет меньше 60 кадров), игра его
// не видела — пункт меню «не нажимался». Такие нажатия ловим сами по
// событиям браузера и отдаём pointer.update как обычный тап.

type tap struct {
	x, y  float64
	touch bool
}

var (
	frameNo   int
	downFrame = map[int]int{}
	lostTaps  []tap
)

func init() {
	defer func() { _ = recover() }()
	win := js.Global()
	down := js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		if e.Get("button").Int() <= 0 {
			downFrame[e.Get("pointerId").Int()] = frameNo
		}
		return nil
	})
	up := js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		id := e.Get("pointerId").Int()
		f, ok := downFrame[id]
		delete(downFrame, id)
		if !ok || f != frameNo || len(lostTaps) > 8 {
			return nil // нажатие шло хотя бы кадр — его увидел Ebitengine
		}
		if x, y, ok := toGame(e.Get("clientX").Float(), e.Get("clientY").Float()); ok {
			lostTaps = append(lostTaps, tap{x, y, e.Get("pointerType").String() == "touch"})
		}
		return nil
	})
	cancel := js.FuncOf(func(_ js.Value, args []js.Value) any {
		delete(downFrame, args[0].Get("pointerId").Int())
		return nil
	})
	opt := map[string]any{"capture": true, "passive": true}
	win.Call("addEventListener", "pointerdown", down, opt)
	win.Call("addEventListener", "pointerup", up, opt)
	win.Call("addEventListener", "pointercancel", cancel, opt)
}

// toGame переводит координаты страницы в координаты игры (холст вписан в
// окно с полями по краям).
func toGame(cx, cy float64) (x, y float64, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	c := js.Global().Get("document").Call("querySelector", "canvas")
	if !c.Truthy() {
		return 0, 0, false
	}
	r := c.Call("getBoundingClientRect")
	w, h := r.Get("width").Float(), r.Get("height").Float()
	s := min(w/ScreenW, h/ScreenH)
	if s <= 0 {
		return 0, 0, false
	}
	x = (cx - r.Get("left").Float() - (w-ScreenW*s)/2) / s
	y = (cy - r.Get("top").Float() - (h-ScreenH*s)/2) / s
	return x, y, x >= 0 && y >= 0 && x < ScreenW && y < ScreenH
}

// lostTap отдаёт одно пропущенное нажатие; вызывается раз в кадр.
func lostTap() (x, y float64, touch, ok bool) {
	frameNo++
	if len(lostTaps) == 0 {
		return
	}
	t := lostTaps[0]
	lostTaps = lostTaps[1:]
	return t.x, t.y, t.touch, true
}
