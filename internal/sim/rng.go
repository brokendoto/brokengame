package sim

// RNG — детерминированный генератор псевдослучайных чисел (SplitMix64).
//
// Вся случайность в симуляции идёт только через него, поэтому одинаковое
// зерно + одинаковые нажатия клавиш всегда дают одинаковую партию.
// На этом держатся «призраки», таблицы рекордов и проверка повторов.
type RNG struct{ s uint64 }

// NewRNG создаёт генератор из зерна.
func NewRNG(seed uint64) RNG { return RNG{s: seed} }

// Next возвращает следующее 64-битное число.
func (r *RNG) Next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Intn возвращает число в диапазоне [0, n).
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.Next() % uint64(n))
}

// Float возвращает число в диапазоне [0, 1).
func (r *RNG) Float() float64 { return float64(r.Next()>>11) / (1 << 53) }

// Hash смешивает строку в 64-битное зерно (FNV-1a + перемешивание SplitMix).
func Hash(s string) uint64 {
	h := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	r := NewRNG(h)
	return r.Next()
}
