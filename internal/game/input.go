package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/brokendoto/brokengame/internal/sim"
)

// Управление: клавиатура (WASD/стрелки) и геймпад (для Steam Deck).

var (
	keysUp     = []ebiten.Key{ebiten.KeyW, ebiten.KeyArrowUp}
	keysDown   = []ebiten.Key{ebiten.KeyS, ebiten.KeyArrowDown}
	keysLeft   = []ebiten.Key{ebiten.KeyA, ebiten.KeyArrowLeft}
	keysRight  = []ebiten.Key{ebiten.KeyD, ebiten.KeyArrowRight}
	keysDash   = []ebiten.Key{ebiten.KeySpace, ebiten.KeyShiftLeft, ebiten.KeyShiftRight}
	keysAbil   = []ebiten.Key{ebiten.KeyE, ebiten.KeyQ}
	keysOK     = []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace}
	keysBack   = []ebiten.Key{ebiten.KeyEscape, ebiten.KeyBackspace}
	keysPause  = []ebiten.Key{ebiten.KeyEscape, ebiten.KeyP}
	keysRetry  = []ebiten.Key{ebiten.KeyR}
	gamepadIDs []ebiten.GamepadID
)

func anyPressed(keys []ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func anyJust(keys []ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func pads() []ebiten.GamepadID {
	gamepadIDs = ebiten.AppendGamepadIDs(gamepadIDs[:0])
	return gamepadIDs
}

func padPressed(b ebiten.StandardGamepadButton) bool {
	for _, id := range pads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && ebiten.IsStandardGamepadButtonPressed(id, b) {
			return true
		}
	}
	return false
}

func padJust(b ebiten.StandardGamepadButton) bool {
	for _, id := range pads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) && inpututil.IsStandardGamepadButtonJustPressed(id, b) {
			return true
		}
	}
	return false
}

func padAxis(a ebiten.StandardGamepadAxis) float64 {
	v := 0.0
	for _, id := range pads() {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			if x := ebiten.StandardGamepadAxisValue(id, a); x*x > v*v {
				v = x
			}
		}
	}
	return v
}

// readPlayInput собирает нажатия игрока в маску для симуляции.
func readPlayInput() sim.Input {
	var in sim.Input
	ax, ay := padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal), padAxis(ebiten.StandardGamepadAxisLeftStickVertical)
	if anyPressed(keysUp) || padPressed(ebiten.StandardGamepadButtonLeftTop) || ay < -0.4 {
		in |= sim.InUp
	}
	if anyPressed(keysDown) || padPressed(ebiten.StandardGamepadButtonLeftBottom) || ay > 0.4 {
		in |= sim.InDown
	}
	if anyPressed(keysLeft) || padPressed(ebiten.StandardGamepadButtonLeftLeft) || ax < -0.4 {
		in |= sim.InLeft
	}
	if anyPressed(keysRight) || padPressed(ebiten.StandardGamepadButtonLeftRight) || ax > 0.4 {
		in |= sim.InRight
	}
	if anyPressed(keysDash) || padPressed(ebiten.StandardGamepadButtonRightBottom) || padPressed(ebiten.StandardGamepadButtonFrontBottomRight) {
		in |= sim.InDash
	}
	if anyPressed(keysAbil) || padPressed(ebiten.StandardGamepadButtonRightLeft) || padPressed(ebiten.StandardGamepadButtonFrontBottomLeft) {
		in |= sim.InAbility
	}
	return in
}

// Навигация по меню.
type nav struct {
	held  [4]int
	axisP [4]bool
}

const (
	navUp = iota
	navDown
	navLeft
	navRight
)

// dir возвращает true в момент нажатия и с автоповтором при удержании.
func (n *nav) dir(d int) bool {
	var keys []ebiten.Key
	var btn ebiten.StandardGamepadButton
	var axis bool
	switch d {
	case navUp:
		keys, btn = keysUp, ebiten.StandardGamepadButtonLeftTop
		axis = padAxis(ebiten.StandardGamepadAxisLeftStickVertical) < -0.5
	case navDown:
		keys, btn = keysDown, ebiten.StandardGamepadButtonLeftBottom
		axis = padAxis(ebiten.StandardGamepadAxisLeftStickVertical) > 0.5
	case navLeft:
		keys, btn = keysLeft, ebiten.StandardGamepadButtonLeftLeft
		axis = padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal) < -0.5
	case navRight:
		keys, btn = keysRight, ebiten.StandardGamepadButtonLeftRight
		axis = padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal) > 0.5
	}
	if anyPressed(keys) || padPressed(btn) || axis {
		n.held[d]++
	} else {
		n.held[d] = 0
	}
	h := n.held[d]
	return h == 1 || (h > 24 && h%6 == 0)
}

func okPressed() bool {
	return anyJust(keysOK) || padJust(ebiten.StandardGamepadButtonRightBottom)
}

func backPressed() bool {
	return anyJust(keysBack) || padJust(ebiten.StandardGamepadButtonRightRight)
}

func pausePressed() bool {
	return anyJust(keysPause) || padJust(ebiten.StandardGamepadButtonCenterRight)
}

func retryPressed() bool {
	return anyJust(keysRetry) || padJust(ebiten.StandardGamepadButtonRightTop)
}
