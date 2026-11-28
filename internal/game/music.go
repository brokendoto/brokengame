package game

import (
	"bytes"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/brokendoto/brokengame/internal/music"
)

// musicPlayer крутит фоновую музыку: у меню и у каждого мира своя пьеса.
// Пьесы синтезируются в фоне при первом запросе (на это уходит до пары
// секунд) и плавно сменяют друг друга.
type musicPlayer struct {
	ctx *audio.Context

	mu    sync.Mutex
	ready map[string][]byte
	busy  map[string]bool

	cur    string
	player *audio.Player
	fade   float64 // 0..1, громкость текущей пьесы при смене
}

func init() {
	if runtime.GOOS != "js" {
		return // на ПК и телефоне синтез идёт в отдельном потоке
	}
	// В браузере у Go один поток: синтез отдаёт управление игре каждые
	// ~4 мс, иначе картинка замирала бы на 1–2 секунды при входе в мир.
	last := time.Now()
	music.Yield = func() {
		if time.Since(last) >= 4*time.Millisecond {
			time.Sleep(time.Millisecond)
			last = time.Now()
		}
	}
}

func newMusicPlayer(ctx *audio.Context) *musicPlayer {
	return &musicPlayer{ctx: ctx, ready: map[string][]byte{}, busy: map[string]bool{}}
}

func styleByName(name string) (music.Style, bool) {
	if name == music.Menu.Name {
		return music.Menu, true
	}
	for _, s := range music.Worlds {
		if s.Name == name {
			return s, true
		}
	}
	return music.Style{}, false
}

// prefetch заранее синтезирует пьесу, если сейчас ничего не синтезируется:
// пока игрок в меню, готовится музыка выбранного мира.
func (m *musicPlayer) prefetch(name string) {
	if m == nil || m.ctx == nil {
		return
	}
	m.mu.Lock()
	idle := len(m.busy) == len(m.ready)
	m.mu.Unlock()
	if idle {
		m.data(name)
	}
}

// data возвращает готовую пьесу или запускает её синтез.
func (m *musicPlayer) data(name string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.ready[name]; ok {
		return d
	}
	if !m.busy[name] {
		if st, ok := styleByName(name); ok {
			m.busy[name] = true
			go func() {
				d := music.Render(st)
				m.mu.Lock()
				m.ready[name] = d
				m.mu.Unlock()
			}()
		}
	}
	return nil
}

// update вызывается каждый кадр: want — какая пьеса нужна сцене ("" —
// оставить текущую), vol — итоговая громкость 0..1.
func (m *musicPlayer) update(want string, vol float64) {
	if m == nil || m.ctx == nil {
		return
	}
	if want == "" {
		want = m.cur
	}
	if want != m.cur && m.player != nil {
		// Старая пьеса затихает за полсекунды.
		m.fade -= 1.0 / 30
		if m.fade > 0 {
			m.player.SetVolume(vol * m.fade)
			return
		}
		_ = m.player.Close()
		m.player = nil
	}
	if m.player == nil {
		m.cur = want
		d := m.data(want)
		if d == nil {
			return
		}
		loop := audio.NewInfiniteLoop(bytes.NewReader(d), int64(len(d)))
		p, err := m.ctx.NewPlayer(loop)
		if err != nil {
			return
		}
		m.player, m.fade = p, 0
		p.SetVolume(0)
		p.Play()
	}
	if m.fade < 1 {
		m.fade = min(1, m.fade+1.0/45)
	}
	m.player.SetVolume(vol * m.fade)
}

// musicFor — какая пьеса нужна сцене.
func musicFor(s Scene) string {
	switch s := s.(type) {
	case *playScene:
		return music.Worlds[s.track.World].Name
	case *duelScene:
		return music.Worlds[s.track.World].Name
	case *resultScene, *marathonScene, *duelResultScene:
		return "" // после уровня играет музыка того же мира
	}
	return music.Menu.Name
}

// prefetchMusic: пока играет музыка меню, заранее готовим пьесу мира,
// в который игрок скорее всего пойдёт (выбранный в меню или в дуэли).
func (a *App) prefetchMusic(cur Scene) {
	if musicFor(cur) != music.Menu.Name || a.Frame%30 != 0 {
		return
	}
	w := a.Prof.World
	switch s := cur.(type) {
	case *duelModeScene:
		w = a.duelWorld
	case *duelSetupScene:
		w = s.world
	}
	if w >= 0 && int(w) < len(music.Worlds) {
		a.Music.prefetch(music.Worlds[w].Name)
	}
}
