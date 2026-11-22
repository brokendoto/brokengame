// Package mobile — точка входа для Android и iOS.
//
// Собирается командой ebitenmobile (см. README): она превращает этот пакет
// в библиотеку .aar для Android или .xcframework для iOS с готовым
// элементом интерфейса EbitenView, в котором и работает игра.
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2"
	ebmobile "github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/brokendoto/brokengame/internal/game"
	"github.com/brokendoto/brokengame/internal/save"
)

// lazyGame создаёт игру при первом кадре, чтобы Java/Swift успели
// передать папку сохранений через SetDataDir.
type lazyGame struct{ app *game.App }

func (g *lazyGame) Update() error {
	if g.app == nil {
		g.app = game.New()
	}
	return g.app.Update()
}

func (g *lazyGame) Draw(screen *ebiten.Image) {
	if g.app != nil {
		g.app.Draw(screen)
	}
}

func (g *lazyGame) Layout(w, h int) (int, int) { return game.ScreenW, game.ScreenH }

func init() {
	ebmobile.SetGame(&lazyGame{})
}

// SetDataDir задаёт папку для сохранений и таблиц рекордов.
func SetDataDir(dir string) { save.SetDir(dir) }

// Dummy нужен, чтобы у пакета была хотя бы одна экспортируемая функция.
func Dummy() {}
