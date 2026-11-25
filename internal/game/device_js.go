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
