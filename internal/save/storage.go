//go:build !js

package save

import "os"

// webStorage — сохранения живут в localStorage браузера, а не в файлах.
const webStorage = false

// storeFile атомарно записывает файл: сначала во временный, потом переименование,
// чтобы сохранение не испортилось при внезапном выключении.
func storeFile(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadFile(path string) ([]byte, error) { return os.ReadFile(path) }
