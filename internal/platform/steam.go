//go:build steam && (windows || linux || darwin)

package platform

import (
	"os"
	"strconv"

	"github.com/hajimehoshi/go-steamworks"
)

// SteamAppID — номер приложения в Steam. 480 — тестовое приложение Valve
// (Spacewar), с ним можно проверить интеграцию до получения своего номера.
// Настоящий номер задаётся при сборке:
//
//	go build -tags steam -ldflags "-X github.com/brokendoto/brokengame/internal/platform.SteamAppID=1234560" ./cmd/pyatimirye
var SteamAppID = "480"

type steam struct{}

func init() {
	id, err := strconv.ParseUint(SteamAppID, 10, 32)
	if err != nil {
		return
	}
	// Если игру запустили не из Steam, Steam перезапустит её сам.
	if os.Getenv("PYATIMIRYE_NO_STEAM_RESTART") == "" && steamworks.RestartAppIfNecessary(uint32(id)) {
		os.Exit(0)
	}
	if err := steamworks.Init(); err != nil {
		return // Steam не запущен — играем без него
	}
	Current = steam{}
}

func (steam) Name() string { return "Steam" }

func (steam) PlayerName() string { return steamworks.SteamFriends().GetPersonaName() }

func (steam) Unlock(id string) {
	if done, ok := steamworks.SteamUserStats().GetAchievement(id); ok && !done {
		steamworks.SteamUserStats().SetAchievement(id)
		steamworks.SteamUserStats().StoreStats()
	}
}

func (steam) SetStatus(text string) { steamworks.SteamFriends().SetRichPresence("status", text) }

func (steam) Update() { steamworks.RunCallbacks() }
