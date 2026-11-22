package sim

import "math"

// Дуэль: двое игроков на одной карте одновременно. Предметы общие — кто
// первым взял, тому и очки. Рывком можно толкнуть соперника и оглушить его,
// способности мира тоже бьют по сопернику. Выбитый игрок теряет очки и через
// 3 секунды возвращается на старт. Партия кончается, когда кто-то выходит
// через открытый выход (+500 очков) или кончается время. Побеждает тот, у кого
// больше очков.
//
// Дуэль использует тот же State, что и одиночная игра: в каждый момент в нём
// «загружен» один из игроков, а функции опасностей перебирают обоих через
// State.eachPlayer. В одиночной игре эти функции ничего не меняют, поэтому
// одиночные забеги и записи остаются прежними.

const (
	DuelExitBonus    = 500     // первому, кто вышел
	DuelKnockTicks   = 3 * TPS // сколько выбитый ждёт возвращения
	DuelKnockPenalty = 300     // штраф за то, что тебя выбили
	DuelCursedCost   = 200     // проклятый предмет в дуэли стоит очков, а не времени
	DuelPushStun     = 50      // оглушение от толчка рывком
	DuelAbilityStun  = 90      // оглушение от стоп-времени или импульса соперника
)

// Duelist — всё, что у каждого игрока дуэли своё.
type Duelist struct {
	P          Player
	Lives      int
	Combo      int
	ComboTimer int
	BestCombo  int
	ItemScore  int
	Hits       int
	Got        int // сколько обычных предметов собрал
	Knocks     int // сколько раз выбит
	Down       int // >0 — выбит, ждёт возвращения
	Stun       int // >0 — оглушён, не слушается кнопок
	Start      Pt
}

// Duel — партия на двоих.
type Duel struct {
	S      *State
	D      [2]Duelist
	Over   bool
	Winner int // 0 или 1; -1 — ничья
	Exited int // кто вышел через выход; -1 — никто (кончилось время)

	cur int // чей игрок сейчас загружен в S; -1 — ничей
}

// NewDuel начинает дуэль на уровне.
func NewDuel(lv *Level) *Duel {
	s := NewState(lv)
	d := &Duel{S: s, cur: -1, Winner: -1, Exited: -1}
	s.duel = d
	// Второй игрок стартует в соседней клетке: вокруг старта генератор
	// не ставит стен.
	st2 := Pt{lv.Start.X, lv.Start.Y + 1}
	if lv.Solid(st2.X, st2.Y) {
		st2.Y = lv.Start.Y - 1
	}
	for i, st := range [2]Pt{lv.Start, st2} {
		d.D[i] = Duelist{Lives: lv.Lives, Start: st}
		d.D[i].P = Player{X: float64(st.X) + 0.5, Y: float64(st.Y) + 0.5, FX: 1}
	}
	return d
}

func (d *Duel) load(i int) {
	d.store()
	s, x := d.S, &d.D[i]
	s.P, s.Lives, s.Combo, s.ComboTimer = x.P, x.Lives, x.Combo, x.ComboTimer
	s.BestCombo, s.ItemScore, s.Hits = x.BestCombo, x.ItemScore, x.Hits
	d.cur = i
}

func (d *Duel) store() {
	if d.cur < 0 {
		return
	}
	s, x := d.S, &d.D[d.cur]
	x.P, x.Lives, x.Combo, x.ComboTimer = s.P, s.Lives, s.Combo, s.ComboTimer
	x.BestCombo, x.ItemScore, x.Hits = s.BestCombo, s.ItemScore, s.Hits
	if x.Lives <= 0 && x.Down == 0 {
		x.Down = DuelKnockTicks
		x.Knocks++
		x.ItemScore = max(0, x.ItemScore-DuelKnockPenalty)
		x.Combo, x.ComboTimer = 0, 0
	}
	d.cur = -1
}

func (d *Duel) active(i int) bool { return d.D[i].Down == 0 }

// each вызывает fn для каждого игрока на поле, загрузив его в State.
func (d *Duel) each(fn func()) {
	d.store()
	for i := range d.D {
		if d.active(i) {
			d.load(i)
			fn()
			d.store()
		}
	}
}

func (s *State) eachPlayer(fn func()) {
	if s.duel == nil {
		fn()
		return
	}
	s.duel.each(fn)
}

// focus загружает игрока, ближайшего к точке (за ним охотится враг).
func (s *State) focus(x, y float64) {
	d := s.duel
	if d == nil {
		return
	}
	d.store()
	best, bd := -1, math.Inf(1)
	for i := range d.D {
		p := d.D[i].P
		if dd := math.Hypot(p.X-x, p.Y-y); d.active(i) && dd < bd {
			best, bd = i, dd
		}
	}
	if best >= 0 {
		d.load(best)
	}
}

// focusAny загружает случайного игрока (в него целится обвал).
func (s *State) focusAny() {
	d := s.duel
	if d == nil {
		return
	}
	d.store()
	i := s.RNG.Intn(2)
	if !d.active(i) {
		i = 1 - i
	}
	if d.active(i) {
		d.load(i)
	}
}

// Step продвигает дуэль на один шаг; in[i] — кнопки i-го игрока.
func (d *Duel) Step(in [2]Input) {
	s := d.S
	s.Events = s.Events[:0]
	if s.Status != Playing {
		return
	}
	s.Tick++
	for i := range d.D {
		x := &d.D[i]
		if x.Down > 0 {
			x.Down--
			if x.Down == 0 {
				x.P = Player{X: float64(x.Start.X) + 0.5, Y: float64(x.Start.Y) + 0.5, FX: 1, Invuln: InvulnTicks}
				x.Lives = s.Lv.Lives
				s.Events = append(s.Events, Event{Kind: EvRespawn, X: x.P.X, Y: x.P.Y, N: i, P: i})
			}
			continue
		}
		pin := in[i]
		if x.Stun > 0 {
			x.Stun--
			pin = 0
		}
		ev0, col0, pen0 := len(s.Events), s.Collected, s.Penalty
		d.load(i)
		s.updatePlayer(pin)
		s.updatePickups()
		if s.Status == Playing {
			s.updateHoles()
		}
		if s.ComboTimer > 0 {
			s.ComboTimer--
			if s.ComboTimer == 0 {
				s.Combo = 0
			}
		}
		d.store()
		x.Got += s.Collected - col0
		if s.Penalty != pen0 {
			s.Penalty = pen0
			x.ItemScore = max(0, x.ItemScore-DuelCursedCost)
		}
		for _, e := range s.Events[ev0:] {
			if e.Kind == EvAbility {
				d.abilityHit(i, Ability(e.N))
			}
		}
		if s.Status == Won {
			d.finish(i)
			return
		}
	}
	d.push()

	// Поле расстояний сразу от обоих игроков: враг идёт к ближайшему.
	// «Хитрый» враг бежит наперерез лидеру.
	var src [2]Pt
	n := 0
	for i := range d.D {
		if d.active(i) {
			src[n] = Pt{int(d.D[i].P.X), int(d.D[i].P.Y)}
			n++
		}
	}
	s.Lv.BFSMulti(src[:n], &s.distP)
	if s.hasCunning() && n > 0 {
		lead := 0
		if !d.active(0) || (d.active(1) && d.D[1].ItemScore > d.D[0].ItemScore) {
			lead = 1
		}
		d.load(lead)
		s.Lv.BFS(s.aheadTarget(), &s.distAhead)
		d.store()
	}
	if s.FreezeT > 0 {
		s.FreezeT--
	} else {
		s.HazTick++
		s.updateChasers()
		s.updateRocks()
		s.updateCaves()
		s.updateSpikes()
	}
	d.store()
	if s.Status == Playing && s.Elapsed() >= s.Lv.TimeLimit {
		s.Status = Lost
		s.LoseReason = "Время вышло"
		s.FinishTick = s.Elapsed()
		d.finish(-1)
	}
}

// push — толчок рывком: кто врезался в соперника на рывке, тот его оглушил.
func (d *Duel) push() {
	s := d.S
	for a := range d.D {
		b := 1 - a
		A, B := &d.D[a], &d.D[b]
		if !d.active(a) || !d.active(b) || A.P.DashT == 0 || B.Stun > 0 || B.P.Invuln > 0 || B.P.DashT > 0 {
			continue
		}
		if math.Hypot(A.P.X-B.P.X, A.P.Y-B.P.Y) > 2*PlayerRadius+0.08 {
			continue
		}
		A.P.DashT = 0
		if B.P.ShieldT > 0 {
			// Щит отражает толчок.
			A.Stun = DuelPushStun
			s.Events = append(s.Events, Event{Kind: EvBlock, X: B.P.X, Y: B.P.Y, N: a, P: b})
			s.Events = append(s.Events, Event{Kind: EvStun, X: A.P.X, Y: A.P.Y, N: a, P: b})
			continue
		}
		B.Stun = DuelPushStun
		B.Combo, B.ComboTimer = 0, 0
		d.load(b)
		s.movePlayer(A.P.DashDX*0.8, A.P.DashDY*0.8)
		d.store()
		s.Events = append(s.Events, Event{Kind: EvStun, X: B.P.X, Y: B.P.Y, N: b, P: a})
	}
}

// abilityHit — действие способности игрока i на соперника.
func (d *Duel) abilityHit(i int, ab Ability) {
	o := &d.D[1-i]
	if !d.active(1 - i) {
		return
	}
	me := d.D[i].P
	hit := false
	switch ab {
	case AbTimeStop:
		hit = true // время стоит и для соперника
	case AbPulse:
		hit = math.Hypot(o.P.X-me.X, o.P.Y-me.Y) < PulseRadius
	}
	if hit && o.P.ShieldT == 0 {
		o.Stun = max(o.Stun, DuelAbilityStun)
		d.S.Events = append(d.S.Events, Event{Kind: EvStun, X: o.P.X, Y: o.P.Y, N: 1 - i, P: i})
	}
}

func (d *Duel) finish(exited int) {
	d.Over = true
	d.Exited = exited
	if exited >= 0 {
		d.D[exited].ItemScore += DuelExitBonus
	}
	a, b := d.D[0].ItemScore, d.D[1].ItemScore
	switch {
	case a > b:
		d.Winner = 0
	case b > a:
		d.Winner = 1
	default:
		d.Winner = exited // при равенстве очков побеждает вышедший
	}
}

// Autopilot — ход бота за игрока i (для тестов и демонстрации).
func (d *Duel) Autopilot(i int) Input {
	if !d.active(i) {
		return 0
	}
	d.load(i)
	in := Autopilot(d.S)
	d.store()
	return in
}
