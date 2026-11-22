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

// Управление в дуэли: левый игрок — WASD, правый — стрелки. Геймпады:
// при двух подключённых первый у левого игрока, второй у правого; если
// геймпад один, он у правого игрока (левый играет на клавиатуре).
var duelKeys = [2]struct{ up, down, left, right, dash, abil []ebiten.Key }{
	{
		up: []ebiten.Key{ebiten.KeyW}, down: []ebiten.Key{ebiten.KeyS},
		left: []ebiten.Key{ebiten.KeyA}, right: []ebiten.Key{ebiten.KeyD},
		dash: []ebiten.Key{ebiten.KeySpace, ebiten.KeyShiftLeft},
		abil: []ebiten.Key{ebiten.KeyE, ebiten.KeyQ},
	},
	{
		up: []ebiten.Key{ebiten.KeyArrowUp}, down: []ebiten.Key{ebiten.KeyArrowDown},
		left: []ebiten.Key{ebiten.KeyArrowLeft}, right: []ebiten.Key{ebiten.KeyArrowRight},
		dash: []ebiten.Key{ebiten.KeyShiftRight, ebiten.KeyEnter, ebiten.KeyNumpad0, ebiten.KeyL},
		abil: []ebiten.Key{ebiten.KeyControlRight, ebiten.KeySlash, ebiten.KeyNumpadDecimal, ebiten.KeyK},
	},
}

// readDuelInput — кнопки i-го игрока дуэли.
func readDuelInput(i int) sim.Input {
	k := &duelKeys[i]
	var in sim.Input
	if anyPressed(k.up) {
		in |= sim.InUp
	}
	if anyPressed(k.down) {
		in |= sim.InDown
	}
	if anyPressed(k.left) {
		in |= sim.InLeft
	}
	if anyPressed(k.right) {
		in |= sim.InRight
	}
	if anyPressed(k.dash) {
		in |= sim.InDash
	}
	if anyPressed(k.abil) {
		in |= sim.InAbility
	}
	ids := pads()
	var std []ebiten.GamepadID
	for _, id := range ids {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			std = append(std, id)
		}
	}
	pi := -1
	switch {
	case len(std) >= 2:
		pi = i
	case len(std) == 1 && i == 1:
		pi = 0
	}
	if pi >= 0 {
		in |= padInput(std[pi])
	}
	return in
}

// padInput — кнопки одного геймпада.
func padInput(id ebiten.GamepadID) sim.Input {
	var in sim.Input
	b := func(x ebiten.StandardGamepadButton) bool { return ebiten.IsStandardGamepadButtonPressed(id, x) }
	ax := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	ay := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
	if b(ebiten.StandardGamepadButtonLeftTop) || ay < -0.4 {
		in |= sim.InUp
	}
	if b(ebiten.StandardGamepadButtonLeftBottom) || ay > 0.4 {
		in |= sim.InDown
	}
	if b(ebiten.StandardGamepadButtonLeftLeft) || ax < -0.4 {
		in |= sim.InLeft
	}
	if b(ebiten.StandardGamepadButtonLeftRight) || ax > 0.4 {
		in |= sim.InRight
	}
	if b(ebiten.StandardGamepadButtonRightBottom) || b(ebiten.StandardGamepadButtonFrontBottomRight) {
		in |= sim.InDash
	}
	if b(ebiten.StandardGamepadButtonRightLeft) || b(ebiten.StandardGamepadButtonFrontBottomLeft) {
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
