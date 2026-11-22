package sim

import "math"

// Status — состояние партии.
type Status int

const (
	Playing Status = iota
	Won
	Lost
)

// EventKind — что произошло за шаг. События нужны только графике и звуку,
// на исход партии они не влияют.
type EventKind int

const (
	EvPickup EventKind = iota
	EvExitOpen
	EvHit
	EvBlock
	EvDash
	EvShield
	EvRockWarn
	EvRockBreak
	EvCaveWarn
	EvCaveFall
	EvHeal
	EvWin
	EvLose
)

// Event — событие шага.
type Event struct {
	Kind EventKind
	X, Y float64
	N    int
}

// Player — герой.
type Player struct {
	X, Y         float64
	FX, FY       float64 // куда смотрит
	DashT        int
	DashCD       int
	DashDX       float64
	DashDY       float64
	Invuln       int
	ShieldT      int
	AbilityCD    int
	prev         Input
	Moving       bool
	WalkDistance float64
}

// Chaser — преследователь («глубинный жук»).
type Chaser struct {
	X, Y   float64
	Kind   ChaserKind
	Spawn  int // >0 — ещё появляется
	Stun   int
	TX, TY int // клетка, к которой идёт
}

// Rock — катящийся валун.
type Rock struct {
	X, Y   float64
	DX, DY int
	Warn   int
	Angle  float64
	Dead   bool
	Speed  float64
	Spawn  Pt
	Origin [2]float64
}

// CaveIn — обвал по строке или столбцу.
type CaveIn struct {
	Horizontal bool
	Line       int
	Warn       int
	Active     int
}

// Item — кристалл на поле.
type Item struct {
	P     Pt
	Taken bool
}

// State — изменяемое состояние партии.
type State struct {
	Lv         *Level
	Tick       int
	RNG        RNG
	P          Player
	Items      []Item
	Collected  int
	ExitOpen   bool
	Chasers    []Chaser
	Rocks      []Rock
	Caves      []CaveIn
	Lives      int
	Combo      int
	ComboTimer int
	BestCombo  int
	ItemScore  int
	Hits       int
	HeartTaken bool
	Status     Status
	LoseReason string
	FinishTick int
	Events     []Event

	nextRock, nextCave int
	distP, distAhead   DistField
}

// NewState начинает партию на уровне.
func NewState(lv *Level) *State {
	s := &State{
		Lv:    lv,
		RNG:   NewRNG(lv.Track.Seed() ^ 0xA11CE),
		Lives: lv.Lives,
	}
	s.P.X, s.P.Y = float64(lv.Start.X)+0.5, float64(lv.Start.Y)+0.5
	s.P.FX = 1
	for _, p := range lv.Items {
		s.Items = append(s.Items, Item{P: p})
	}
	for i, c := range lv.Chasers {
		s.Chasers = append(s.Chasers, Chaser{
			X: float64(c.P.X) + 0.5, Y: float64(c.P.Y) + 0.5,
			Kind: c.Kind, TX: c.P.X, TY: c.P.Y,
			Spawn: 90 + i*45,
		})
	}
	if lv.Cfg.RockEvery > 0 {
		s.nextRock = 3 * TPS
	}
	if lv.Cfg.CaveEvery > 0 {
		s.nextCave = 5 * TPS
	}
	return s
}

// TimeLeft — сколько шагов осталось до конца времени.
func (s *State) TimeLeft() int {
	t := s.Lv.TimeLimit - s.Tick
	if t < 0 {
		return 0
	}
	return t
}

// Multiplier — множитель очков текущей серии.
func (s *State) Multiplier() float64 { return comboMul(s.Combo) }

func comboMul(combo int) float64 {
	if combo < 1 {
		return 1
	}
	if combo > 9 {
		combo = 9
	}
	return 1 + 0.25*float64(combo-1)
}

func (s *State) emit(k EventKind, x, y float64, n int) {
	s.Events = append(s.Events, Event{Kind: k, X: x, Y: y, N: n})
}

// Step продвигает партию на один шаг с нажатыми кнопками in.
func (s *State) Step(in Input) {
	s.Events = s.Events[:0]
	if s.Status != Playing {
		return
	}
	s.Tick++
	s.updatePlayer(in)
	s.updatePickups()
	if s.Status != Playing {
		return
	}
	s.Lv.BFS(Pt{int(s.P.X), int(s.P.Y)}, &s.distP)
	if s.hasCunning() {
		s.Lv.BFS(s.aheadTarget(), &s.distAhead)
	}
	s.updateChasers()
	s.updateRocks()
	s.updateCaves()
	s.updateSpikes()
	if s.ComboTimer > 0 {
		s.ComboTimer--
		if s.ComboTimer == 0 {
			s.Combo = 0
		}
	}
	if s.Status == Playing && s.Tick >= s.Lv.TimeLimit {
		s.lose("Время вышло")
	}
}

func (s *State) lose(reason string) {
	s.Status = Lost
	s.LoseReason = reason
	s.FinishTick = s.Tick
	s.emit(EvLose, s.P.X, s.P.Y, 0)
}

// --- игрок -----------------------------------------------------------------

func inputDir(in Input) (float64, float64) {
	var dx, dy float64
	if in&InLeft != 0 {
		dx--
	}
	if in&InRight != 0 {
		dx++
	}
	if in&InUp != 0 {
		dy--
	}
	if in&InDown != 0 {
		dy++
	}
	if dx != 0 && dy != 0 {
		dx *= math.Sqrt2 / 2
		dy *= math.Sqrt2 / 2
	}
	return dx, dy
}

func (s *State) updatePlayer(in Input) {
	p := &s.P
	pressed := in &^ p.prev
	p.prev = in
	dx, dy := inputDir(in)
	if dx != 0 || dy != 0 {
		p.FX, p.FY = dx, dy
	}

	if pressed&InDash != 0 && p.DashCD == 0 && p.DashT == 0 {
		p.DashT, p.DashCD = DashTicks, DashCooldown
		p.DashDX, p.DashDY = p.FX, p.FY
		s.emit(EvDash, p.X, p.Y, 0)
	}
	if pressed&InAbility != 0 && p.AbilityCD == 0 {
		p.ShieldT, p.AbilityCD = ShieldTicks, ShieldCD
		s.emit(EvShield, p.X, p.Y, 0)
	}

	var vx, vy float64
	if p.DashT > 0 {
		vx, vy = p.DashDX*DashSpeed, p.DashDY*DashSpeed
		p.DashT--
	} else {
		sp := PlayerSpeed
		if s.Lv.CellAt(int(p.X), int(p.Y)) == Mud {
			sp *= MudFactor
		}
		vx, vy = dx*sp, dy*sp
	}
	ox, oy := p.X, p.Y
	s.movePlayer(vx/TPS, vy/TPS)
	moved := math.Hypot(p.X-ox, p.Y-oy)
	p.Moving = moved > 1e-4
	p.WalkDistance += moved

	if p.DashCD > 0 {
		p.DashCD--
	}
	if p.AbilityCD > 0 {
		p.AbilityCD--
	}
	if p.ShieldT > 0 {
		p.ShieldT--
	}
	if p.Invuln > 0 {
		p.Invuln--
	}
}

// blocked — пересекает ли квадрат игрока в точке (x, y) стену.
func (s *State) blocked(x, y float64) bool {
	const r = PlayerRadius
	x0, x1 := int(math.Floor(x-r)), int(math.Floor(x+r))
	y0, y1 := int(math.Floor(y-r)), int(math.Floor(y+r))
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			if s.Lv.Solid(cx, cy) {
				return true
			}
		}
	}
	return false
}

// movePlayer двигает игрока по осям по отдельности, скользя вдоль стен.
// Если игрок упирается в угол, его мягко «подталкивает» к центру прохода —
// так клеточное поле не цепляет за края.
func (s *State) movePlayer(dx, dy float64) {
	p := &s.P
	if dx != 0 {
		if !s.blocked(p.X+dx, p.Y) {
			p.X += dx
		} else {
			p.X = slide(p.X, dx, func(v float64) bool { return s.blocked(v, p.Y) })
			if dy == 0 {
				p.Y = s.nudge(p.Y, math.Abs(dx),
					func(v float64) bool { return !s.blocked(p.X+dx, v) },
					func(v float64) bool { return s.blocked(p.X, v) })
			}
		}
	}
	if dy != 0 {
		if !s.blocked(p.X, p.Y+dy) {
			p.Y += dy
		} else {
			p.Y = slide(p.Y, dy, func(v float64) bool { return s.blocked(p.X, v) })
			if dx == 0 {
				p.X = s.nudge(p.X, math.Abs(dy),
					func(v float64) bool { return !s.blocked(v, p.Y+dy) },
					func(v float64) bool { return s.blocked(v, p.Y) })
			}
		}
	}
}

// slide находит, насколько можно сдвинуться до стены (бинарный поиск).
func slide(v, d float64, hit func(float64) bool) float64 {
	lo, hi := 0.0, 1.0
	for i := 0; i < 8; i++ {
		m := (lo + hi) / 2
		if hit(v + d*m) {
			hi = m
		} else {
			lo = m
		}
	}
	return v + d*lo
}

// nudge сдвигает поперечную координату v к центру ближайшего прохода,
// через который движение вперёд свободно (passable), не влезая в стену (hit).
func (s *State) nudge(v, amount float64, passable, hit func(float64) bool) float64 {
	base := math.Floor(v) + 0.5
	cands := [3]float64{base, base - 1, base + 1}
	if v > base {
		cands[1], cands[2] = base+1, base-1
	}
	for _, c := range cands {
		off := c - v
		if math.Abs(off) < 1e-6 || math.Abs(off) > 0.62 || !passable(c) {
			continue
		}
		step := math.Min(math.Abs(off), amount)
		if off < 0 {
			step = -step
		}
		if !hit(v + step) {
			return v + step
		}
		return v
	}
	return v
}

// hurt пытается ранить игрока. Возвращает true, если удар пришёлся в щит.
func (s *State) hurt(reason string) (blocked bool) {
	p := &s.P
	if p.Invuln > 0 || p.DashT > 0 {
		return false
	}
	if p.ShieldT > 0 {
		s.emit(EvBlock, p.X, p.Y, 0)
		return true
	}
	s.Lives--
	s.Hits++
	s.Combo, s.ComboTimer = 0, 0
	p.Invuln = InvulnTicks
	s.emit(EvHit, p.X, p.Y, s.Lives)
	if s.Lives <= 0 {
		s.lose(reason)
	}
	return false
}

// --- предметы и выход -------------------------------------------------------

func (s *State) updatePickups() {
	p := &s.P
	for i := range s.Items {
		it := &s.Items[i]
		if it.Taken {
			continue
		}
		ix, iy := float64(it.P.X)+0.5, float64(it.P.Y)+0.5
		if math.Hypot(p.X-ix, p.Y-iy) < PickupRadius {
			it.Taken = true
			s.Collected++
			s.Combo++
			s.ComboTimer = ComboWindow
			if s.Combo > s.BestCombo {
				s.BestCombo = s.Combo
			}
			s.ItemScore += int(100 * comboMul(s.Combo))
			s.emit(EvPickup, ix, iy, s.Combo)
		}
	}
	if !s.ExitOpen && s.Collected == len(s.Items) {
		s.ExitOpen = true
		s.emit(EvExitOpen, float64(s.Lv.Exit.X)+0.5, float64(s.Lv.Exit.Y)+0.5, 0)
	}
	if s.Lv.HasHeart && !s.HeartTaken {
		hx, hy := float64(s.Lv.Heart.X)+0.5, float64(s.Lv.Heart.Y)+0.5
		if math.Hypot(p.X-hx, p.Y-hy) < PickupRadius {
			s.HeartTaken = true
			if s.Lives < s.Lv.Lives {
				s.Lives++
			} else {
				s.ItemScore += 300
			}
			s.emit(EvHeal, hx, hy, s.Lives)
		}
	}
	if s.ExitOpen {
		ex, ey := float64(s.Lv.Exit.X)+0.5, float64(s.Lv.Exit.Y)+0.5
		if math.Hypot(p.X-ex, p.Y-ey) < 0.5 {
			s.Status = Won
			s.FinishTick = s.Tick
			s.emit(EvWin, ex, ey, 0)
		}
	}
}

// --- преследователи ---------------------------------------------------------

func (s *State) hasCunning() bool {
	for _, c := range s.Chasers {
		if c.Kind == ChaserCunning {
			return true
		}
	}
	return false
}

// aheadTarget — клетка в 3 шагах впереди игрока (для «хитрого» жука).
func (s *State) aheadTarget() Pt {
	best := Pt{int(s.P.X), int(s.P.Y)}
	for k := 1.0; k <= 3; k++ {
		c := Pt{int(s.P.X + s.P.FX*k), int(s.P.Y + s.P.FY*k)}
		if s.Lv.Solid(c.X, c.Y) {
			break
		}
		best = c
	}
	return best
}

func (s *State) updateChasers() {
	for i := range s.Chasers {
		c := &s.Chasers[i]
		if c.Spawn > 0 {
			c.Spawn--
			continue
		}
		if c.Stun > 0 {
			c.Stun--
			continue
		}
		speed := s.Lv.Cfg.EnemySpeed
		if c.Kind == ChaserCunning {
			speed *= 0.92
		}
		if s.Lv.CellAt(int(c.X), int(c.Y)) == Mud {
			speed *= 0.7
		}
		step := speed / TPS

		pd := math.Hypot(s.P.X-c.X, s.P.Y-c.Y)
		cell := Pt{clampInt(int(c.X), 0, W-1), clampInt(int(c.Y), 0, H-1)}
		if pd < 1.3 && s.distP[cell.Y][cell.X] >= 0 && s.distP[cell.Y][cell.X] <= 1 {
			// Вплотную — бросок прямо на игрока.
			if pd > 1e-9 {
				k := math.Min(step, pd) / pd
				c.X += (s.P.X - c.X) * k
				c.Y += (s.P.Y - c.Y) * k
			}
			c.TX, c.TY = clampInt(int(c.X), 0, W-1), clampInt(int(c.Y), 0, H-1)
		} else {
			field := &s.distP
			if c.Kind == ChaserCunning && pd > 3 {
				field = &s.distAhead
			}
			s.stepChaser(c, i, step, field)
		}

		if math.Hypot(s.P.X-c.X, s.P.Y-c.Y) < PlayerRadius+ChaserRadius {
			if s.hurt("Тебя догнал жук") {
				c.Stun = 90
			} else if s.P.Invuln == InvulnTicks {
				c.Stun = 45
			}
		}
	}
}

// stepChaser ведёт жука от центра клетки к центру клетки по полю расстояний:
// на каждой развилке он выбирает соседа, который ближе к цели.
func (s *State) stepChaser(c *Chaser, idx int, step float64, field *DistField) {
	for step > 1e-9 {
		tx, ty := float64(c.TX)+0.5, float64(c.TY)+0.5
		dx, dy := tx-c.X, ty-c.Y
		d := math.Hypot(dx, dy)
		if d > step {
			c.X += dx / d * step
			c.Y += dy / d * step
			return
		}
		c.X, c.Y = tx, ty
		step -= d
		best := field[c.TY][c.TX]
		nx, ny := c.TX, c.TY
		for k := 0; k < 4; k++ {
			dv := dirs[(k+idx)%4]
			x, y := c.TX+dv.X, c.TY+dv.Y
			if s.Lv.Solid(x, y) {
				continue
			}
			f := field[y][x]
			if f >= 0 && (best < 0 || f < best) {
				best, nx, ny = f, x, y
			}
		}
		if nx == c.TX && ny == c.TY {
			return
		}
		c.TX, c.TY = nx, ny
	}
}

// --- валуны -----------------------------------------------------------------

func (s *State) updateRocks() {
	if s.Lv.Cfg.RockEvery > 0 {
		s.nextRock--
		if s.nextRock <= 0 {
			s.spawnRock()
			s.nextRock = s.Lv.Cfg.RockEvery + s.RNG.Intn(s.Lv.Cfg.RockEvery/2)
		}
	}
	alive := s.Rocks[:0]
	for _, r := range s.Rocks {
		if r.Warn > 0 {
			r.Warn--
		} else {
			step := r.Speed / TPS
			r.X += float64(r.DX) * step
			r.Y += float64(r.DY) * step
			r.Angle += step / RockRadius * float64(r.DX+r.DY)
			cx, cy := int(math.Floor(r.X+float64(r.DX)*RockRadius)), int(math.Floor(r.Y+float64(r.DY)*RockRadius))
			inside := cx >= 0 && cy >= 0 && cx < W && cy < H
			if inside && s.Lv.Solid(cx, cy) {
				r.Dead = true
				s.emit(EvRockBreak, r.X, r.Y, 0)
			} else if r.X < -1.5 || r.Y < -1.5 || r.X > W+1.5 || r.Y > H+1.5 {
				r.Dead = true
			}
			if !r.Dead && math.Hypot(s.P.X-r.X, s.P.Y-r.Y) < PlayerRadius+RockRadius {
				s.hurt("Под валуном")
			}
		}
		if !r.Dead {
			alive = append(alive, r)
		}
	}
	s.Rocks = alive
}

func (s *State) spawnRock() {
	for try := 0; try < 8; try++ {
		r := Rock{Warn: RockWarnTicks, Speed: 5.0 + 0.15*float64(s.Lv.Track.Level)}
		dir := 1
		if s.RNG.Intn(2) == 0 {
			dir = -1
		}
		if s.RNG.Intn(2) == 0 {
			row := s.RNG.Intn(H)
			x := 0
			if dir < 0 {
				x = W - 1
			}
			if s.Lv.Solid(x, row) {
				continue
			}
			r.DX, r.X, r.Y = dir, float64(x)+0.5-float64(dir)*0.9, float64(row)+0.5
			r.Spawn = Pt{x, row}
		} else {
			col := s.RNG.Intn(W)
			y := 0
			if dir < 0 {
				y = H - 1
			}
			if s.Lv.Solid(col, y) {
				continue
			}
			r.DY, r.X, r.Y = dir, float64(col)+0.5, float64(y)+0.5-float64(dir)*0.9
			r.Spawn = Pt{col, y}
		}
		s.Rocks = append(s.Rocks, r)
		s.emit(EvRockWarn, float64(r.Spawn.X)+0.5, float64(r.Spawn.Y)+0.5, 0)
		return
	}
}

// --- обвалы -----------------------------------------------------------------

func (s *State) updateCaves() {
	if s.Lv.Cfg.CaveEvery > 0 {
		s.nextCave--
		if s.nextCave <= 0 {
			c := CaveIn{Horizontal: s.RNG.Intn(2) == 0, Warn: CaveWarnTicks, Active: CaveHitTicks}
			// Обвал прицеливается рядом с игроком, чтобы держать его в тонусе.
			if c.Horizontal {
				c.Line = clampInt(int(s.P.Y)+s.RNG.Intn(5)-2, 0, H-1)
			} else {
				c.Line = clampInt(int(s.P.X)+s.RNG.Intn(7)-3, 0, W-1)
			}
			s.Caves = append(s.Caves, c)
			s.emit(EvCaveWarn, s.P.X, s.P.Y, 0)
			s.nextCave = s.Lv.Cfg.CaveEvery + s.RNG.Intn(s.Lv.Cfg.CaveEvery/2)
		}
	}
	alive := s.Caves[:0]
	for _, c := range s.Caves {
		if c.Warn > 0 {
			c.Warn--
			if c.Warn == 0 {
				s.emit(EvCaveFall, 0, 0, c.Line)
			}
		} else {
			c.Active--
			var d float64
			if c.Horizontal {
				d = math.Abs(s.P.Y - (float64(c.Line) + 0.5))
			} else {
				d = math.Abs(s.P.X - (float64(c.Line) + 0.5))
			}
			if d < 0.5+PlayerRadius*0.5 {
				s.hurt("Завалило породой")
			}
		}
		if c.Warn > 0 || c.Active > 0 {
			alive = append(alive, c)
		}
	}
	s.Caves = alive
}

// --- шипы -------------------------------------------------------------------

// SpikePhase возвращает фазу шипов на текущем шаге.
func (s *State) SpikePhase(t SpikeTrap) int { return (s.Tick + t.Phase) % SpikePeriod }

func (s *State) updateSpikes() {
	pc := Pt{int(s.P.X), int(s.P.Y)}
	for _, t := range s.Lv.Spikes {
		if t.P == pc && s.SpikePhase(t) >= SpikeUpFrom {
			s.hurt("Напоролся на шипы")
		}
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
