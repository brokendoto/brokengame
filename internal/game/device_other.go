//go:build !js

package game

func touchScreen() bool { return IsMobile() }
