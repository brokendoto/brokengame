// Package assets — встроенные в исполняемый файл ресурсы.
package assets

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
)

//go:embed icon_256.png
var iconPNG []byte

// Icon возвращает иконку окна.
func Icon() image.Image {
	img, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return nil
	}
	return img
}
