// Package save хранит профиль игрока и локальные таблицы рекордов в JSON.
//
// Файлы лежат в папке настроек пользователя (на Windows это
// %AppData%\Pyatimirye). Если папку получить нельзя (например, в браузере),
// данные живут только в памяти.
package save

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var dirOverride string

// SetDir задаёт папку сохранений явно. На Android её передаёт Java-код
// (внутренняя папка приложения), потому что стандартной папки настроек там нет.
func SetDir(dir string) { dirOverride = dir }

// Dir возвращает папку сохранений или "" если её нет.
func Dir() string {
	if dirOverride != "" {
		if err := os.MkdirAll(dirOverride, 0o755); err == nil {
			return dirOverride
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	d := filepath.Join(base, "Pyatimirye")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return ""
	}
	return d
}

var writeMu sync.Mutex

// writeJSON атомарно записывает файл: сначала во временный, потом переименование,
// чтобы сохранение не испортилось при внезапном выключении.
func writeJSON(path string, v any) error {
	if path == "" {
		return nil
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, v any) error {
	if path == "" {
		return os.ErrNotExist
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
