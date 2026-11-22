// Пятимирье — аркада о сборе предметов на время с таблицами рекордов,
// призраками и испытанием дня.
package main

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/brokendoto/brokengame/internal/assets"
	"github.com/brokendoto/brokengame/internal/game"
)

func main() {
	ebiten.SetWindowTitle("Пятимирье")
	ebiten.SetWindowSize(game.ScreenW, game.ScreenH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if icon := assets.Icon(); icon != nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}
	app := game.New()
	if app.Prof.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
