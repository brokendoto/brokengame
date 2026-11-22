package sim

// Physics constants (в клетках и клетках/сек).
const (
	PlayerRadius  = 0.32
	PlayerSpeed   = 4.6
	MudFactor     = 0.55
	DashSpeed     = 13.0
	DashTicks     = 10
	DashCooldown  = 75
	InvulnTicks   = 90
	ComboWindow   = 150
	PickupRadius  = 0.55
	ChaserRadius  = 0.30
	RockRadius    = 0.38
	SpikePeriod   = 180
	SpikeWarnFrom = 110 // с этой фазы шипы «шевелятся»
	SpikeUpFrom   = 140 // с этой фазы шипы опасны
	CaveWarnTicks = 75
	CaveHitTicks  = 18
	RockWarnTicks = 50
	IceGrip       = 0.06 // доля изменения скорости за шаг на льду
	MagnetRadius  = 2.5
	BlinkDistance = 3.0
	PulseRadius   = 3.0
	GoldenScore   = 250
	PhantomEvery  = 300
	PhantomWarn   = 45
	HoleRadius    = 3.5
	HolePull      = 3.2
	HoleCore      = 0.45
)

// LevelConfig — параметры уровня, зависящие от мира, номера и сложности.
type LevelConfig struct {
	Items      int
	WallFill   float64
	ZoneBlobs  int
	Spikes     int
	Chasers    int
	Special    int // из них особых врагов мира (хитрый, зигзаг, призрак)
	RockEvery  int // период валунов в шагах, 0 — нет
	CaveEvery  int // период обвалов/лучей в шагах, 0 — нет
	Heart      bool
	EnemySpeed float64
	BlackHoles int
	Cursed     int
	Golden     int
}

// ConfigFor возвращает параметры уровня. Сложность растёт плавно:
// новые опасности появляются по одной, чтобы игрок успел их освоить.
func ConfigFor(w World, level int, d Difficulty) LevelConfig {
	if level < 1 {
		level = 1
	}
	r := WorldRules[w]
	c := LevelConfig{
		Items:    6 + level,
		WallFill: 0.10 + 0.012*float64(level),
	}
	if level >= 2 {
		c.ZoneBlobs = 1 + level/3
		if r.Spikes {
			c.Spikes = (level - 1) * 2
		}
	}
	switch {
	case level >= 7:
		c.Chasers, c.Special = 4, 1
	case level >= 6:
		c.Chasers, c.Special = 3, 1
	case level >= 4:
		c.Chasers = 2
	case level >= 2:
		c.Chasers = 1
	}
	if w != Mine && level >= 4 {
		// В новых мирах особый враг появляется раньше — он и есть изюминка мира.
		c.Special = 1 + level/7
	}
	if r.Rocks && level >= 3 {
		c.RockEvery = 260 - level*15
	}
	beamFrom := 5
	if !r.Rocks {
		beamFrom = 3
	}
	if r.Beams && level >= beamFrom {
		c.CaveEvery = 440 - level*20
	}
	if r.BlackHoles && level >= 3 {
		c.BlackHoles = 1 + level/6
	}
	if r.Cursed && level >= 2 {
		c.Cursed = 1 + level/3
	}
	if r.Golden {
		c.Golden = 2 + level/4
	}
	c.Heart = level >= 4
	c.EnemySpeed = (2.3 + 0.1*float64(level)) * Difficulties[d].EnemySpeed
	return c
}

// SpikeTrap — шипы на клетке с собственной фазой.
type SpikeTrap struct {
	P     Pt
	Phase int
}

// ChaserKind — вид преследователя.
type ChaserKind int

const (
	ChaserBasic   ChaserKind = iota // бежит к игроку кратчайшим путём
	ChaserCunning                   // бежит наперерез, туда, куда смотрит игрок
	ChaserZigzag                    // быстрый, но петляет на развилках
	ChaserPhantom                   // время от времени телепортируется за спину
)

// ChaserSpawn — где и кто появляется.
type ChaserSpawn struct {
	P    Pt
	Kind ChaserKind
}

// Level — неизменяемое описание уровня, полученное генератором.
type Level struct {
	Track     Track
	Cfg       LevelConfig
	Grid      [H][W]Cell
	Start     Pt
	Exit      Pt
	Items     []Pt
	Spikes    []SpikeTrap
	Chasers   []ChaserSpawn
	Heart     Pt
	HasHeart  bool
	Holes     []Pt // чёрные дыры
	Cursed    []Pt // проклятые предметы
	Golden    []Pt // исчезающие бонусные предметы
	ParTicks  int  // эталонное время (жадный обход через BFS)
	TimeLimit int  // в шагах
	Lives     int
}

// Solid — непроходима ли клетка (за краем поля — стена).
func (lv *Level) Solid(x, y int) bool {
	if x < 0 || y < 0 || x >= W || y >= H {
		return true
	}
	return lv.Grid[y][x] == Wall
}

// Rules — правила мира уровня.
func (lv *Level) Rules() *Rules { return &WorldRules[lv.Track.World] }

// CellAt возвращает тип клетки (за краем — стена).
func (lv *Level) CellAt(x, y int) Cell {
	if x < 0 || y < 0 || x >= W || y >= H {
		return Wall
	}
	return lv.Grid[y][x]
}

// DistField — расстояния в шагах от клетки-источника до всех клеток (-1 — недостижимо).
type DistField [H][W]int16

var dirs = [4]Pt{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

// BFS заполняет поле расстояний от from поиском в ширину.
func (lv *Level) BFS(from Pt, d *DistField) {
	src := [1]Pt{from}
	lv.BFSMulti(src[:], d)
}

// BFSMulti — поиск в ширину сразу от нескольких клеток: в каждой клетке
// оказывается расстояние до ближайшей из них (нужно для дуэли).
func (lv *Level) BFSMulti(from []Pt, d *DistField) {
	for y := range d {
		for x := range d[y] {
			d[y][x] = -1
		}
	}
	var q [W * H]Pt
	head, tail := 0, 0
	for _, f := range from {
		if lv.Solid(f.X, f.Y) || d[f.Y][f.X] == 0 {
			continue
		}
		q[tail] = f
		tail++
		d[f.Y][f.X] = 0
	}
	for head < tail {
		c := q[head]
		head++
		for _, dv := range dirs {
			n := Pt{c.X + dv.X, c.Y + dv.Y}
			if lv.Solid(n.X, n.Y) || d[n.Y][n.X] >= 0 {
				continue
			}
			d[n.Y][n.X] = d[c.Y][c.X] + 1
			q[tail] = n
			tail++
		}
	}
}

// Medal возвращает медаль за прохождение за ticks шагов.
func (lv *Level) Medal(ticks int) Medal {
	switch {
	case ticks <= lv.GoldTicks():
		return Gold
	case ticks <= lv.SilverTicks():
		return Silver
	default:
		return Bronze
	}
}

// GoldTicks — время на золото.
func (lv *Level) GoldTicks() int { return lv.ParTicks * 115 / 100 }

// SilverTicks — время на серебро.
func (lv *Level) SilverTicks() int { return lv.ParTicks * 160 / 100 }
