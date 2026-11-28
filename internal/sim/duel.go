package sim

import "math"

// Дуэль: двое игроков на одной карте одновременно, матч из раундов до двух
// побед. Каждый раунд — следующий уровень мира. Три вида дуэли:
//
//   - «Раунды и кража» (DuelRounds): счёт — предметы на руках (обычный +1,
//     бонусный +2). Рывок в соперника оглушает его и выбивает предмет, выбитый
//     опасностями роняет два. Раунд кончается, когда предметы на поле
//     кончились или вышло время; у кого больше, тот и взял раунд.
//   - «Банк» (DuelBank): подобранное лежит в мешке, засчитывается, только когда
//     донесёшь до портала. Рывок в соперника вытряхивает весь его мешок.
//   - «Выживание» (DuelSurvival): рывок в соперника ранит его, каждые три
//     предмета лечат. Возвращений нет: кто остался без жизней, проиграл раунд.
//
// Дуэль использует тот же State, что и одиночная игра: в каждый момент в нём
// «загружен» один из игроков, а функции опасностей перебирают обоих через
// State.eachPlayer. В одиночной игре эти функции ничего не меняют, поэтому
// одиночные забеги и записи остаются прежними.

const (
	DuelKnockTicks  = 3 * TPS  // сколько выбитый ждёт возвращения
	DuelKnockDrop   = 2        // сколько предметов роняет выбитый
	DuelGoldenPts   = 2        // бонусный предмет в дуэли
	DuelPushStun    = 50       // оглушение от толчка рывком
	DuelAbilityStun = 90       // оглушение от стоп-времени или импульса соперника
	DuelRoundTime   = 75 * TPS // самое долгое время раунда
	DuelWinsNeeded  = 2        // побед в раундах для победы в матче
	DuelMaxRounds   = 5        // раундов не больше (на случай ничьих)
	DuelBankRadius  = 0.6      // насколько близко к порталу сдаётся мешок
	DuelHealEvery   = 3        // в «Выживании» столько предметов лечат на жизнь
	DuelExtraLives  = 2        // в «Выживании» жизней может быть больше начальных
	duelOut         = 1 << 30  // Down выбывшего насовсем
)

// DuelMode — вид дуэли.
type DuelMode int

const (
	DuelRounds DuelMode = iota
	DuelBank
	DuelSurvival
	DuelModeCount
)

// DuelModeInfo — описание вида дуэли для меню.
type DuelModeInfo struct {
	Name  string
	Short string   // одна строка для карточки
	Rules []string // правила для экрана подготовки
}

var DuelModes = [DuelModeCount]DuelModeInfo{
	{
		Name:  "Раунды и кража",
		Short: "Собери больше соперника, отнимай рывком",
		Rules: []string{
			"Счёт — предметы на руках: обычный +1, бонусный +2.",
			"Рывок в соперника оглушает его и выбивает предмет — подбирай!",
			"Выбитый опасностями роняет 2 предмета и через 3 с возвращается на старт.",
			"Раунд кончается, когда предметов на поле нет или вышло время.",
		},
	},
	{
		Name:  "Банк",
		Short: "Донеси добычу до портала, пока не отняли",
		Rules: []string{
			"Подобранное лежит в мешке. Засчитывается, когда донесёшь до портала.",
			"Рывок в соперника вытряхивает весь его мешок на поле.",
			"Выбитый опасностями тоже теряет мешок и через 3 с возвращается на старт.",
			"Раунд кончается, когда всё сдано или вышло время. Считается только банк.",
		},
	},
	{
		Name:  "Выживание",
		Short: "Последний, у кого остались жизни, побеждает",
		Rules: []string{
			"Рывок в соперника ранит его: −1 жизнь. Проклятый предмет тоже ранит.",
			"Каждые 3 подобранных предмета лечат на жизнь, бонусный — сразу.",
			"Возвращений нет: кто остался без жизней, проиграл раунд.",
			"Вышло время — побеждает тот, у кого больше жизней.",
		},
	},
}

// Duelist — всё, что у каждого игрока дуэли своё.
type Duelist struct {
	P          Player
	Lives      int
	Combo      int
	ComboTimer int
	BestCombo  int
	ItemScore  int
	Hits       int
	Pts        int // счёт раунда: предметы на руках или в банке
	Carry      int // «Банк»: предметов в мешке
	Got        int // сколько предметов подобрал за раунд
	Steals     int // сколько предметов выбил у соперника рывком
	Knocks     int // сколько раз выбит опасностями
	Down       int // >0 — выбит, ждёт возвращения
	Stun       int // >0 — оглушён, не слушается кнопок
	Start      Pt
}

// Duel — один раунд на двоих.
type Duel struct {
	Mode   DuelMode
	S      *State
	D      [2]Duelist
	Over   bool
	Winner int // 0 или 1; -1 — ничья

	cur  int // чей игрок сейчас загружен в S; -1 — ничей
	tmp  DistField
	busy [H][W]bool
}

// NewDuel начинает раунд на уровне.
func NewDuel(lv *Level, mode DuelMode) *Duel {
	if lv.TimeLimit > DuelRoundTime {
		lv.TimeLimit = DuelRoundTime
	}
	s := NewState(lv)
	d := &Duel{S: s, cur: -1, Winner: -1, Mode: mode}
	s.duel = d
	if mode == DuelBank {
		s.ExitOpen = true // портал всегда открыт: это банк
	}
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
	i := d.cur
	s, x := d.S, &d.D[i]
	x.P, x.Lives, x.Combo, x.ComboTimer = s.P, s.Lives, s.Combo, s.ComboTimer
	x.BestCombo, x.ItemScore, x.Hits = s.BestCombo, s.ItemScore, s.Hits
	d.cur = -1
	if x.Lives <= 0 && x.Down == 0 {
		x.Knocks++
		x.Combo, x.ComboTimer = 0, 0
		switch d.Mode {
		case DuelSurvival:
			x.Down = duelOut
		case DuelBank:
			x.Down = DuelKnockTicks
			d.lose(i, x.Carry, 1)
		default:
			x.Down = DuelKnockTicks
			d.lose(i, DuelKnockDrop, 1)
		}
	}
}

func (d *Duel) active(i int) bool { return d.D[i].Down == 0 }

// Out — игрок выбыл из раунда насовсем («Выживание»).
func (d *Duel) Out(i int) bool { return d.D[i].Down >= duelOut/2 }

// Left — сколько обычных предметов ещё лежит на поле.
func (d *Duel) Left() int {
	n := 0
	for _, it := range d.S.Items {
		if !it.Taken {
			n++
		}
	}
	return n
}

// Leader — кто ведёт в раунде; -1 — поровну.
func (d *Duel) Leader() int {
	if d.Mode == DuelSurvival {
		a, b := d.D[0].Lives, d.D[1].Lives
		if d.Out(0) {
			a = 0
		}
		if d.Out(1) {
			b = 0
		}
		if a != b {
			return b2i(b > a)
		}
	}
	switch {
	case d.D[0].Pts > d.D[1].Pts:
		return 0
	case d.D[1].Pts > d.D[0].Pts:
		return 1
	}
	return -1
}

// lose — игрок i теряет до n предметов: они падают на поле не ближе
// minDist клеток от него. Возвращает, сколько упало.
func (d *Duel) lose(i, n, minDist int) int {
	x := &d.D[i]
	held := &x.Pts
	if d.Mode == DuelBank {
		held = &x.Carry
	}
	n = min(n, *held)
	if n <= 0 {
		return 0
	}
	got := d.drop(x.P.X, x.P.Y, n, minDist, i)
	*held -= got
	if got > 0 {
		d.S.Events = append(d.S.Events, Event{Kind: EvLoss, X: x.P.X, Y: x.P.Y, N: i, P: got})
	}
	return got
}

// drop кладёт n предметов на свободные клетки, достижимые от точки,
// на расстоянии от minDist до minDist+2 шагов (если таких нет — дальше).
func (d *Duel) drop(fx, fy float64, n, minDist, who int) int {
	s, lv := d.S, d.S.Lv
	from := Pt{int(fx), int(fy)}
	if lv.Solid(from.X, from.Y) {
		return 0
	}
	lv.BFS(from, &d.tmp)
	b := &d.busy
	*b = [H][W]bool{}
	mark := func(p Pt) { b[p.Y][p.X] = true }
	for _, it := range s.Items {
		if !it.Taken {
			mark(it.P)
		}
	}
	for _, it := range s.Cursed {
		mark(it.P)
	}
	for _, it := range s.Golden {
		mark(it.P)
	}
	for _, h := range lv.Holes {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if q := (Pt{h.X + dx, h.Y + dy}); q.X >= 0 && q.Y >= 0 && q.X < W && q.Y < H {
					mark(q)
				}
			}
		}
	}
	for _, sp := range lv.Spikes {
		mark(sp.P)
	}
	mark(lv.Exit)
	mark(lv.Start)
	if lv.HasHeart {
		mark(lv.Heart)
	}
	placed := 0
	for lo := minDist; lo < W+H && placed < n; lo += 3 {
		var cand []Pt
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				dv := int(d.tmp[y][x])
				if dv >= lo && dv <= lo+2 && !b[y][x] && lv.Grid[y][x] != Wall {
					cand = append(cand, Pt{x, y})
				}
			}
		}
		for placed < n && len(cand) > 0 {
			k := s.RNG.Intn(len(cand))
			p := cand[k]
			cand[k] = cand[len(cand)-1]
			cand = cand[:len(cand)-1]
			if b[p.Y][p.X] {
				continue
			}
			mark(p)
			s.Items = append(s.Items, Item{P: p})
			s.Events = append(s.Events, Event{Kind: EvDrop, X: float64(p.X) + 0.5, Y: float64(p.Y) + 0.5, N: who, P: who})
			placed++
		}
	}
	return placed
}

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

// Step продвигает раунд на один шаг; in[i] — кнопки i-го игрока.
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
		got := s.Collected - col0
		golden := false
		evs := s.Events[ev0:]
		for k := range evs {
			switch evs[k].Kind {
			case EvAbility:
				d.abilityHit(i, Ability(evs[k].N))
			case EvGolden:
				golden = true
			}
		}
		d.gain(i, got, golden)
		if s.Penalty != pen0 {
			// Проклятый предмет в дуэли стоит не времени, а предмета (или жизни).
			s.Penalty = pen0
			if d.Mode == DuelSurvival {
				d.wound(i)
			} else {
				d.lose(i, 1, 2)
			}
		}
		if d.Mode == DuelBank && x.Carry > 0 && d.active(i) {
			ex, ey := float64(s.Lv.Exit.X)+0.5, float64(s.Lv.Exit.Y)+0.5
			if math.Hypot(x.P.X-ex, x.P.Y-ey) < DuelBankRadius {
				x.Pts += x.Carry
				s.Events = append(s.Events, Event{Kind: EvBank, X: ex, Y: ey, N: i, P: x.Carry})
				x.Carry = 0
			}
		}
	}
	d.push()
	if d.roundDone() {
		d.finish()
		return
	}

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
		if !d.active(0) || (d.active(1) && d.D[1].Pts > d.D[0].Pts) {
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
	if d.roundDone() || s.Elapsed() >= s.Lv.TimeLimit {
		d.finish()
	}
}

// roundDone — раунд кончился раньше времени.
func (d *Duel) roundDone() bool {
	switch d.Mode {
	case DuelSurvival:
		return d.Out(0) || d.Out(1)
	case DuelBank:
		return d.Left() == 0 && d.D[0].Carry == 0 && d.D[1].Carry == 0
	}
	return d.Left() == 0
}

// gain — игрок i подобрал got обычных предметов и, может быть, бонусный.
func (d *Duel) gain(i, got int, golden bool) {
	x := &d.D[i]
	if golden {
		x.Got++
	}
	x.Got += got
	switch d.Mode {
	case DuelBank:
		x.Carry += got + DuelGoldenPts*b2i(golden)
	case DuelSurvival:
		x.Pts += got + b2i(golden)
		heal := b2i(golden)
		for k := x.Got - got; k < x.Got; k++ {
			if (k+1)%DuelHealEvery == 0 {
				heal++
			}
		}
		if heal > 0 && x.Lives < d.S.Lv.Lives+DuelExtraLives {
			x.Lives = min(x.Lives+heal, d.S.Lv.Lives+DuelExtraLives)
			d.S.Events = append(d.S.Events, Event{Kind: EvHeal, X: x.P.X, Y: x.P.Y, N: x.Lives, P: i})
		}
	default:
		x.Pts += got + DuelGoldenPts*b2i(golden)
	}
}

// wound ранит игрока i, как опасность.
func (d *Duel) wound(i int) {
	if !d.active(i) {
		return
	}
	d.load(i)
	d.S.hurt("")
	d.store()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// push — толчок рывком: кто врезался в соперника на рывке, тот его оглушил
// и выбил у него предмет.
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
		switch d.Mode {
		case DuelSurvival:
			d.wound(b)
			A.Steals++
		case DuelBank:
			A.Steals += d.lose(b, B.Carry, 2)
		default:
			A.Steals += d.lose(b, 1, 2)
		}
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

func (d *Duel) finish() {
	s := d.S
	if s.Status == Playing {
		s.Status = Won
		s.FinishTick = s.Elapsed()
	}
	d.Over = true
	d.Winner = d.Leader()
}

// TimedOut — раунд кончился по времени (а не потому, что всё собрано).
func (d *Duel) TimedOut() bool { return d.Over && d.Left() > 0 }

// Autopilot — ход бота за игрока i (для тестов и демонстрации).
func (d *Duel) Autopilot(i int) Input {
	if !d.active(i) {
		return 0
	}
	d.load(i)
	s := d.S
	open := s.ExitOpen
	if d.Mode == DuelBank {
		// Бот несёт мешок в банк, когда тот полон или собирать нечего.
		c := d.D[i].Carry
		s.ExitOpen = c >= 3 || (c > 0 && d.Left() == 0)
	}
	in := Autopilot(s)
	s.ExitOpen = open
	d.store()
	return in
}

// --- Матч из нескольких раундов ---------------------------------------------

// DuelMatch — матч до DuelWinsNeeded побед. Каждый следующий раунд идёт на
// следующем уровне того же мира.
type DuelMatch struct {
	Mode   DuelMode
	Base   Track
	Round  int // номер текущего раунда с 1
	D      *Duel
	Wins   [2]int
	Draws  int
	Over   bool
	Winner int // -1 — ничья

	Got, Steals, Knocks [2]int // итоги за матч
	History             []int  // победитель каждого сыгранного раунда (-1 — ничья)
}

// NewDuelMatch начинает матч с первого раунда.
func NewDuelMatch(t Track, mode DuelMode) *DuelMatch {
	m := &DuelMatch{Base: t, Winner: -1, Mode: mode}
	m.NextRound()
	return m
}

// RoundTrack — уровень текущего раунда.
func (m *DuelMatch) RoundTrack() Track {
	t := m.Base
	t.Level = (t.Level-1+m.Round-1)%LevelsPerWorld + 1
	return t
}

// NextRound готовит следующий раунд.
func (m *DuelMatch) NextRound() {
	m.Round++
	m.D = NewDuel(Generate(m.RoundTrack()), m.Mode)
}

// EndRound подводит итог законченного раунда. Возвращает true, если матч
// закончен.
func (m *DuelMatch) EndRound() bool {
	d := m.D
	if !d.Over || len(m.History) >= m.Round {
		return m.Over
	}
	m.History = append(m.History, d.Winner)
	if d.Winner >= 0 {
		m.Wins[d.Winner]++
	} else {
		m.Draws++
	}
	for i := range d.D {
		m.Got[i] += d.D[i].Got
		m.Steals[i] += d.D[i].Steals
		m.Knocks[i] += d.D[i].Knocks
	}
	switch {
	case m.Wins[0] >= DuelWinsNeeded:
		m.Over, m.Winner = true, 0
	case m.Wins[1] >= DuelWinsNeeded:
		m.Over, m.Winner = true, 1
	case m.Round >= DuelMaxRounds:
		m.Over = true
		switch {
		case m.Wins[0] > m.Wins[1]:
			m.Winner = 0
		case m.Wins[1] > m.Wins[0]:
			m.Winner = 1
		}
	}
	return m.Over
}
