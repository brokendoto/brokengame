// Package platform — связь с магазином, в котором запущена игра (Steam).
//
// Обычная сборка работает без магазина: Current — заглушка, которая ничего
// не делает. Сборка с тегом steam (go build -tags steam) подключает
// Steamworks: достижения Steam, имя игрока из профиля Steam и статус
// «во что играет» для друзей. Если Steam не запущен, игра всё равно работает.
package platform

// Platform — возможности магазина.
type Platform interface {
	// Name — название платформы для меню ("" — без магазина).
	Name() string
	// PlayerName — имя игрока в магазине ("" — неизвестно).
	PlayerName() string
	// Unlock открывает достижение с идентификатором id.
	Unlock(id string)
	// SetStatus показывает друзьям, чем занят игрок.
	SetStatus(text string)
	// Update вызывается каждый кадр (обработка событий магазина).
	Update()
}

type none struct{}

func (none) Name() string       { return "" }
func (none) PlayerName() string { return "" }
func (none) Unlock(string)      {}
func (none) SetStatus(string)   {}
func (none) Update()            {}

// Current — платформа, в которой запущена игра. Сборка с тегом steam
// подменяет её при старте, если Steam доступен.
var Current Platform = none{}
