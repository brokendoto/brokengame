// Package replay упаковывает запись нажатий партии в короткую строку.
//
// Формат: RLE (значение кнопок, длина серии в varint) → base64.
// Минута игры обычно занимает меньше килобайта: кнопки меняются редко.
package replay

import (
	"encoding/base64"
	"encoding/binary"
	"errors"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Encode упаковывает нажатия.
func Encode(in []sim.Input) string {
	var buf []byte
	for i := 0; i < len(in); {
		j := i
		for j < len(in) && in[j] == in[i] {
			j++
		}
		buf = append(buf, byte(in[i]))
		buf = binary.AppendUvarint(buf, uint64(j-i))
		i = j
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// MaxTicks ограничивает длину записи (защита от испорченных файлов).
const MaxTicks = 60 * 60 * sim.TPS

// Decode распаковывает нажатия.
func Decode(s string) ([]sim.Input, error) {
	buf, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var out []sim.Input
	for len(buf) > 0 {
		v := sim.Input(buf[0])
		n, k := binary.Uvarint(buf[1:])
		if k <= 0 {
			return nil, errors.New("replay: битая запись")
		}
		if uint64(len(out))+n > MaxTicks {
			return nil, errors.New("replay: слишком длинная запись")
		}
		for ; n > 0; n-- {
			out = append(out, v)
		}
		buf = buf[1+k:]
	}
	return out, nil
}
