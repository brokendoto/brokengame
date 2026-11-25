//go:build js

package save

import (
	"os"
	"syscall/js"
)

// В браузере файлов нет: профиль и рекорды лежат в localStorage под ключами
// "pyatimirye/profile.json" и т. п. Так прогресс не теряется после закрытия
// вкладки — и при открытии с сайта, и при открытии файла с диска.
const webStorage = true

func storage() (v js.Value) {
	defer func() {
		if recover() != nil { // доступ к хранилищу запрещён (приватный режим и т. п.)
			v = js.Undefined()
		}
	}()
	return js.Global().Get("localStorage")
}

func storeFile(path string, b []byte) (err error) {
	s := storage()
	if !s.Truthy() {
		return os.ErrPermission
	}
	defer func() {
		if recover() != nil { // хранилище переполнено
			err = os.ErrPermission
		}
	}()
	s.Call("setItem", path, string(b))
	return nil
}

func loadFile(path string) (b []byte, err error) {
	s := storage()
	if !s.Truthy() {
		return nil, os.ErrNotExist
	}
	defer func() {
		if recover() != nil {
			b, err = nil, os.ErrNotExist
		}
	}()
	v := s.Call("getItem", path)
	if v.IsNull() || v.IsUndefined() {
		return nil, os.ErrNotExist
	}
	return []byte(v.String()), nil
}
