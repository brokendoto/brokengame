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
	EvAbility // N — какая способность
	EvRockWarn
	EvRockBreak
	EvCaveWarn
	EvCaveFall
	EvHeal
	EvWin
	EvLose
	EvCursed  // взят проклятый предмет
	EvGolden  // взят бонусный предмет
	EvVanish  // бонусный предмет исчез
	EvBlinkIn // призрак начинает телепорт
	EvBlinkOut
	EvHole     // затянуло в чёрную дыру
	EvKnockout // дуэль: игрок выбит (N — номер игрока)
	EvRespawn  // дуэль: игрок вернулся на старт
	EvStun     // дуэль: соперник оглушён толчком или способностью
	EvDrop     // дуэль: на поле упал выбитый предмет (N — чей)
	EvLoss     // дуэль: игрок N потерял P предметов
	EvBank     // дуэль «Банк»: игрок N сдал P предметов
)

// Event — событие шага.
type Event struct {
	Kind EventKind
	X, Y float64
	N    int
	P    int // чей это игрок (в дуэли 0 или 1, иначе 0)
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
	MagnetT      int
	AbilityCD    int
	VX, VY       float64 // скорость (важна на льду)
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
	// Для призрака: таймер телепорта и куда он прыгнет.
	BlinkT, BlinkWarn int
	BX, BY            int
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

// Item — предмет на поле.
type Item struct {
	P     Pt
	Taken bool
}

// GoldenLife — сколько шагов живёт бонусный предмет (первый; следующие дольше).
const GoldenLife = 20 * TPS

// Штраф за проклятый предмет.
const CursedPenalty = 5 * TPS

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
	Cursed     []Item
	Golden     []Item
	Penalty    int // штрафные шаги (проклятые предметы)
	FreezeT    int // стоп-время
	HazTick    int // часы ловушек (стоят во время стоп-времени)
	Status     Status
	LoseReason string
	FinishTick int
	Events     []Event

	nextRock, nextCave int
	distP, distAhead   DistField
	duel               *Duel // не nil в режиме дуэли
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
	for _, p := range lv.Cursed {
		s.Cursed = append(s.Cursed, Item{P: p})
	}
	for _, p := range lv.Golden {
		s.Golden = append(s.Golden, Item{P: p})
	}
	for i, c := range lv.Chasers {
		s.Chasers = append(s.Chasers, Chaser{
			X: float64(c.P.X) + 0.5, Y: float64(c.P.Y) + 0.5,
			Kind: c.Kind, TX: c.P.X, TY: c.P.Y,
			Spawn:  90 + i*45,
			BlinkT: 240 + i*70,
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
	t := s.Lv.TimeLimit - s.Elapsed()
	if t < 0 {
		return 0
	}
	return t
}

// Elapsed — прошедшее время с учётом штрафов.
func (s *State) Elapsed() int { return s.Tick + s.Penalty }

// GoldenLeft — сколько шагов осталось жить i-му бонусному предмету.
func (s *State) GoldenLeft(i int) int { return GoldenLife + i*5*TPS - s.Tick }

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
	e := Event{Kind: k, X: x, Y: y, N: n}
	if s.duel != nil && s.duel.cur >= 0 {
		e.P = s.duel.cur
	}
	s.Events = append(s.Events, e)
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
	s.updateHoles()
	if s.Status != Playing {
		return
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
	if s.ComboTimer > 0 {
		s.ComboTimer--
		if s.ComboTimer == 0 {
			s.Combo = 0
		}
	}
	if s.Status == Playing && s.Elapsed() >= s.Lv.TimeLimit {
		s.lose("Время вышло")
	}
}

func (s *State) lose(reason string) {
	s.Status = Lost
	s.LoseReason = reason
	s.FinishTick = s.Elapsed()
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
	cell := s.Lv.CellAt(int(p.X), int(p.Y))
	if cell == Fog {
		// Туман: управление наоборот.
		dx, dy = -dx, -dy
	}
	if dx != 0 || dy != 0 {
		p.FX, p.FY = dx, dy
	}

	if pressed&InDash != 0 && p.DashCD == 0 && p.DashT == 0 {
		p.DashT, p.DashCD = DashTicks, DashCooldown
		p.DashDX, p.DashDY = p.FX, p.FY
		s.emit(EvDash, p.X, p.Y, 0)
	}
	if pressed&InAbility != 0 && p.AbilityCD == 0 {
		s.useAbility()
	}

	var vx, vy float64
	if p.DashT > 0 {
		vx, vy = p.DashDX*DashSpeed, p.DashDY*DashSpeed
		p.DashT--
		if p.DashT == 0 {
			vx, vy = p.DashDX*PlayerSpeed, p.DashDY*PlayerSpeed
		}
	} else {
		sp := PlayerSpeed
		if cell == Mud {
			sp *= MudFactor
		}
		vx, vy = dx*sp, dy*sp
		if cell == Ice {
			// Лёд/невесомость: скорость меняется плавно, героя заносит.
			vx = p.VX + (vx-p.VX)*IceGrip
			vy = p.VY + (vy-p.VY)*IceGrip
		}
	}
	p.VX, p.VY = vx, vy
	px, py := s.holePull()
	ox, oy := p.X, p.Y
	s.movePlayer(vx/TPS+px, vy/TPS+py)
	// Упёрся в стену — эта составляющая скорости гасится.
	if math.Abs(p.X-ox) < math.Abs(vx/TPS)*0.5 {
		p.VX = 0
	}
	if math.Abs(p.Y-oy) < math.Abs(vy/TPS)*0.5 {
		p.VY = 0
	}
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
	if p.MagnetT > 0 {
		p.MagnetT--
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
		if s.duel != nil {
			// В дуэли проигравших нет: игрок выбит и вернётся на старт.
			s.emit(EvKnockout, p.X, p.Y, s.duel.cur)
			return false
		}
		s.lose(reason)
	}
	return false
}

// --- предметы и выход -------------------------------------------------------

func (s *State) updatePickups() {
	p := &s.P
	radius := PickupRadius
	if p.MagnetT > 0 {
		radius = MagnetRadius
	}
	for i := range s.Cursed {
		it := &s.Cursed[i]
		if !it.Taken && math.Hypot(p.X-float64(it.P.X)-0.5, p.Y-float64(it.P.Y)-0.5) < PickupRadius {
			it.Taken = true
			s.Penalty += CursedPenalty
			s.Combo, s.ComboTimer = 0, 0
			s.emit(EvCursed, float64(it.P.X)+0.5, float64(it.P.Y)+0.5, 0)
		}
	}
	for i := range s.Golden {
		it := &s.Golden[i]
		if it.Taken {
			continue
		}
		gx, gy := float64(it.P.X)+0.5, float64(it.P.Y)+0.5
		if s.GoldenLeft(i) <= 0 {
			it.Taken = true
			s.emit(EvVanish, gx, gy, 0)
			continue
		}
		if math.Hypot(p.X-gx, p.Y-gy) < radius {
			it.Taken = true
			s.Combo++
			s.ComboTimer = ComboWindow
			if s.Combo > s.BestCombo {
				s.BestCombo = s.Combo
			}
			s.ItemScore += int(GoldenScore * comboMul(s.Combo))
			s.emit(EvGolden, gx, gy, s.Combo)
		}
	}
	for i := range s.Items {
		it := &s.Items[i]
		if it.Taken {
			continue
		}
		ix, iy := float64(it.P.X)+0.5, float64(it.P.Y)+0.5
		if math.Hypot(p.X-ix, p.Y-iy) < radius {
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
	if s.duel == nil && !s.ExitOpen && s.Collected == len(s.Items) {
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
	if s.ExitOpen && s.duel == nil {
		ex, ey := float64(s.Lv.Exit.X)+0.5, float64(s.Lv.Exit.Y)+0.5
		if math.Hypot(p.X-ex, p.Y-ey) < 0.5 {
			s.Status = Won
			s.FinishTick = s.Elapsed()
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
		s.focus(c.X, c.Y) // в дуэли враг охотится на ближайшего
		speed := s.Lv.Cfg.EnemySpeed
		switch c.Kind {
		case ChaserCunning:
			speed *= 0.92
		case ChaserZigzag:
			speed *= 1.2
		case ChaserPhantom:
			speed *= 0.8
			if s.updatePhantom(c) {
				continue
			}
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

		s.eachPlayer(func() {
			if c.Stun == 0 && math.Hypot(s.P.X-c.X, s.P.Y-c.Y) < PlayerRadius+ChaserRadius {
				if s.hurt(s.Lv.Rules().ChaserHit) {
					c.Stun = 90
				} else if s.P.Invuln == InvulnTicks {
					c.Stun = 45
				}
			}
		})
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
		if c.Kind == ChaserZigzag && s.RNG.Intn(100) < 40 {
			// Зигзаг: на развилке иногда сворачивает в случайную сторону.
			dv := dirs[s.RNG.Intn(4)]
			if x, y := c.TX+dv.X, c.TY+dv.Y; !s.Lv.Solid(x, y) {
				nx, ny = x, y
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
			if !r.Dead {
				s.eachPlayer(func() {
					if math.Hypot(s.P.X-r.X, s.P.Y-r.Y) < PlayerRadius+RockRadius {
						s.hurt(s.Lv.Rules().RockHit)
					}
				})
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
			s.focusAny()
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
			s.eachPlayer(func() {
				var d float64
				if c.Horizontal {
					d = math.Abs(s.P.Y - (float64(c.Line) + 0.5))
				} else {
					d = math.Abs(s.P.X - (float64(c.Line) + 0.5))
				}
				if d < 0.5+PlayerRadius*0.5 {
					s.hurt(s.Lv.Rules().BeamHit)
				}
			})
		}
		if c.Warn > 0 || c.Active > 0 {
			alive = append(alive, c)
		}
	}
	s.Caves = alive
}

// --- шипы -------------------------------------------------------------------

// SpikePhase возвращает фазу шипов на текущем шаге.
func (s *State) SpikePhase(t SpikeTrap) int { return (s.HazTick + t.Phase) % SpikePeriod }

func (s *State) updateSpikes() {
	s.eachPlayer(func() {
		pc := Pt{int(s.P.X), int(s.P.Y)}
		for _, t := range s.Lv.Spikes {
			if t.P == pc && s.SpikePhase(t) >= SpikeUpFrom {
				s.hurt(s.Lv.Rules().SpikeHit)
			}
		}
	})
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

// --- способности ---------------------------------------------------------------

func (s *State) useAbility() {
	p := &s.P
	ab := s.Lv.Rules().Ability
	info := Abilities[ab]
	p.AbilityCD = info.Cooldown
	switch ab {
	case AbShield:
		p.ShieldT = info.Duration
	case AbMagnet:
		p.MagnetT = info.Duration
	case AbTimeStop:
		s.FreezeT = info.Duration
	case AbBlink:
		// Самая дальняя свободная точка на 3 клетки вперёд (стены не мешают).
		for k := BlinkDistance; k >= 0.5; k -= 0.25 {
			x, y := p.X+p.FX*k, p.Y+p.FY*k
			if !s.blocked(x, y) && x > 0 && y > 0 && x < W && y < H {
				s.emit(EvBlinkOut, p.X, p.Y, 0)
				p.X, p.Y = x, y
				p.Invuln = max(p.Invuln, 12)
				break
			}
		}
	case AbPulse:
		for i := range s.Chasers {
			c := &s.Chasers[i]
			if c.Spawn == 0 && math.Hypot(c.X-p.X, c.Y-p.Y) < PulseRadius {
				c.Stun = 150
				c.BlinkWarn = 0
			}
		}
		for i := range s.Rocks {
			r := &s.Rocks[i]
			if r.Warn == 0 && math.Hypot(r.X-p.X, r.Y-p.Y) < PulseRadius {
				r.Dead = true
				s.emit(EvRockBreak, r.X, r.Y, 0)
			}
		}
	}
	s.emit(EvAbility, p.X, p.Y, int(ab))
}

// --- призрак (телепорт за спину) ------------------------------------------------

// updatePhantom ведёт таймер телепорта. Возвращает true, если в этот шаг
// призрак не двигается обычным образом (готовится к прыжку).
func (s *State) updatePhantom(c *Chaser) bool {
	if c.BlinkWarn > 0 {
		c.BlinkWarn--
		if c.BlinkWarn == 0 {
			s.emit(EvBlinkOut, c.X, c.Y, 0)
			c.X, c.Y = float64(c.BX)+0.5, float64(c.BY)+0.5
			c.TX, c.TY = c.BX, c.BY
			c.Stun = 20 // короткая пауза после прыжка — шанс увернуться
			s.emit(EvBlinkIn, c.X, c.Y, 0)
		}
		return true
	}
	c.BlinkT--
	if c.BlinkT > 0 {
		return false
	}
	c.BlinkT = PhantomEvery
	// Клетка в 2–3 шагах позади игрока.
	for k := 3.0; k >= 2; k-- {
		x, y := int(s.P.X-s.P.FX*k), int(s.P.Y-s.P.FY*k)
		if !s.Lv.Solid(x, y) && s.distP[y][x] >= 0 {
			c.BX, c.BY = x, y
			c.BlinkWarn = PhantomWarn
			s.emit(EvBlinkIn, float64(x)+0.5, float64(y)+0.5, 1)
			return true
		}
	}
	return false
}

// --- чёрные дыры ---------------------------------------------------------------

// holePull — смещение игрока за шаг от притяжения чёрных дыр.
func (s *State) holePull() (float64, float64) {
	var fx, fy float64
	for _, h := range s.Lv.Holes {
		dx, dy := float64(h.X)+0.5-s.P.X, float64(h.Y)+0.5-s.P.Y
		d := math.Hypot(dx, dy)
		if d < HoleRadius && d > 1e-6 {
			f := HolePull * (1 - d/HoleRadius) / TPS
			fx += dx / d * f
			fy += dy / d * f
		}
	}
	return fx, fy
}

func (s *State) updateHoles() {
	p := &s.P
	for _, h := range s.Lv.Holes {
		hx, hy := float64(h.X)+0.5, float64(h.Y)+0.5
		if math.Hypot(p.X-hx, p.Y-hy) < HoleCore && p.Invuln == 0 && p.DashT == 0 {
			// Щит от чёрной дыры не спасает: героя выбрасывает на старт.
			saved := p.ShieldT
			p.ShieldT = 0
			s.hurt("Затянуло в чёрную дыру")
			p.ShieldT = saved
			s.emit(EvHole, hx, hy, 0)
			p.X, p.Y = float64(s.Lv.Start.X)+0.5, float64(s.Lv.Start.Y)+0.5
			p.VX, p.VY = 0, 0
			return
		}
	}
}
