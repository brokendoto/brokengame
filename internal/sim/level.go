package sim

// Physics constants (в клетках и клетках/сек).
const (
	PlayerRadius  = 0.32
	PlayerSpeed   = 4.6
	MudFactor     = 0.55
	DashSpeed     = 13.0
	DashTicks     = 10
	DashCooldown  = 75
	ShieldTicks   = 240
	ShieldCD      = 720
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
)

// LevelConfig — параметры уровня, зависящие от его номера и сложности.
type LevelConfig struct {
	Items      int
	WallFill   float64
	MudBlobs   int
	Spikes     int
	Chasers    int
	Cunning    int // из них «хитрых», бегущих наперерез
	RockEvery  int // период валунов в шагах, 0 — нет
	CaveEvery  int // период обвалов в шагах, 0 — нет
	Heart      bool
	EnemySpeed float64
}

// ConfigFor возвращает параметры уровня. Сложность растёт плавно:
// новые опасности появляются по одной, чтобы игрок успел их освоить.
func ConfigFor(level int, d Difficulty) LevelConfig {
	if level < 1 {
		level = 1
	}
	c := LevelConfig{
		Items:    6 + level,
		WallFill: 0.10 + 0.012*float64(level),
	}
	if level >= 2 {
		c.MudBlobs = 1 + level/3
		c.Spikes = (level - 1) * 2
	}
	switch {
	case level >= 7:
		c.Chasers, c.Cunning = 4, 1
	case level >= 6:
		c.Chasers, c.Cunning = 3, 1
	case level >= 4:
		c.Chasers = 2
	case level >= 2:
		c.Chasers = 1
	}
	if level >= 3 {
		c.RockEvery = 260 - level*15
	}
	if level >= 5 {
		c.CaveEvery = 440 - level*20
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
	ParTicks  int // эталонное время (жадный обход через BFS)
	TimeLimit int // в шагах
	Lives     int
}

// Solid — непроходима ли клетка (за краем поля — стена).
func (lv *Level) Solid(x, y int) bool {
	if x < 0 || y < 0 || x >= W || y >= H {
		return true
	}
	return lv.Grid[y][x] == Wall
}

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
	for y := range d {
		for x := range d[y] {
			d[y][x] = -1
		}
	}
	if lv.Solid(from.X, from.Y) {
		return
	}
	var q [W * H]Pt
	head, tail := 0, 0
	q[tail] = from
	tail++
	d[from.Y][from.X] = 0
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
