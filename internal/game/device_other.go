//go:build !js

package game

func touchScreen() bool { return IsMobile() }

// lostTap — только для браузера (см. device_js.go).
func lostTap() (x, y float64, touch, ok bool) { return }
