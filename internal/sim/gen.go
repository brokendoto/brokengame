package sim

// Генератор уровней. Уровень полностью определяется зерном трассы, поэтому
// карту не нужно хранить или передавать: достаточно ключа трассы.

// Generate строит уровень для трассы.
func Generate(t Track) *Level {
	level := t.Level
	if t.Daily != "" {
		level = DailyLevel
	}
	cfg := ConfigFor(t.World, level, t.Diff)
	rng := NewRNG(t.Seed())
	for attempt := 0; attempt < 400; attempt++ {
		if lv := tryGenerate(&rng, t, cfg); lv != nil {
			return lv
		}
	}
	// Запасной вариант, если зерно неудачное: поле без стен всегда строится.
	cfg.WallFill = 0
	for {
		if lv := tryGenerate(&rng, t, cfg); lv != nil {
			return lv
		}
	}
}

func cheb(a, b Pt) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func shuffle(rng *RNG, p []Pt) {
	for i := len(p) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
}

func tryGenerate(rng *RNG, t Track, cfg LevelConfig) *Level {
	lv := &Level{Track: t, Cfg: cfg, Lives: Difficulties[t.Diff].Lives}
	lv.Start = Pt{rng.Intn(2), 1 + rng.Intn(H-2)}

	// 1. Скальные блоки.
	target := int(cfg.WallFill * W * H)
	walls := 0
	for tries := 0; walls < target && tries < 600; tries++ {
		w, h := 1+rng.Intn(3), 1+rng.Intn(3)
		switch rng.Intn(4) {
		case 0:
			w, h = 3+rng.Intn(4), 1
		case 1:
			w, h = 1, 3+rng.Intn(3)
		}
		x, y := rng.Intn(W-w+1), rng.Intn(H-h+1)
		if x <= lv.Start.X+2 && x+w-1 >= lv.Start.X-2 && y <= lv.Start.Y+2 && y+h-1 >= lv.Start.Y-2 {
			continue
		}
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				if lv.Grid[yy][xx] == Floor {
					lv.Grid[yy][xx] = Wall
					walls++
				}
			}
		}
	}

	// 2. Всё, куда нельзя дойти от старта, засыпаем породой.
	var dist DistField
	lv.BFS(lv.Start, &dist)
	reach := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if lv.Grid[y][x] == Wall {
				continue
			}
			if dist[y][x] < 0 {
				lv.Grid[y][x] = Wall
			} else {
				reach++
			}
		}
	}
	if reach < W*H*55/100 {
		return nil
	}

	// 3. Выход — самая дальняя по пути клетка у правого края.
	best := int16(-1)
	for y := 0; y < H; y++ {
		for x := W - 3; x < W; x++ {
			if dist[y][x] > best {
				best, lv.Exit = dist[y][x], Pt{x, y}
			}
		}
	}
	if best < 20 {
		return nil
	}

	// Кандидаты на предметы: подальше от старта.
	var cand []Pt
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			p := Pt{x, y}
			if dist[y][x] >= 3 && p != lv.Exit {
				cand = append(cand, p)
			}
		}
	}
	shuffle(rng, cand)

	// 4. Кристаллы, не ближе 2 клеток друг к другу.
	used := map[Pt]bool{lv.Exit: true, lv.Start: true}
	for _, p := range cand {
		if len(lv.Items) == cfg.Items {
			break
		}
		ok := cheb(p, lv.Exit) >= 2
		for _, q := range lv.Items {
			if cheb(p, q) < 2 {
				ok = false
				break
			}
		}
		if ok {
			lv.Items = append(lv.Items, p)
			used[p] = true
		}
	}
	if len(lv.Items) < cfg.Items {
		return nil
	}

	// 5. Зоны мира: грязь/лужи замедляют, туман путает управление, лёд скользкий.
	zone := WorldRules[t.World].Zone
	for i := 0; i < cfg.ZoneBlobs; i++ {
		c := cand[rng.Intn(len(cand))]
		r := 1 + rng.Intn(2)
		if zone != Mud {
			r++
		}
		for y := c.Y - r; y <= c.Y+r; y++ {
			for x := c.X - r; x <= c.X+r; x++ {
				dx, dy := x-c.X, y-c.Y
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				if dx+dy <= r && lv.CellAt(x, y) == Floor && dist[y][x] >= 2 {
					lv.Grid[y][x] = zone
				}
			}
		}
	}

	// 6. Шипы — не на предметах, не рядом друг с другом и с выходом.
	shuffle(rng, cand)
	for _, p := range cand {
		if len(lv.Spikes) == cfg.Spikes {
			break
		}
		if used[p] || dist[p.Y][p.X] < 4 || lv.Grid[p.Y][p.X] != Floor || cheb(p, lv.Exit) < 2 {
			continue
		}
		ok := true
		for _, s := range lv.Spikes {
			if cheb(p, s.P) < 2 {
				ok = false
				break
			}
		}
		if ok {
			lv.Spikes = append(lv.Spikes, SpikeTrap{P: p, Phase: rng.Intn(SpikePeriod)})
			used[p] = true
		}
	}

	// 7. Преследователи появляются далеко от старта.
	for _, p := range cand {
		if len(lv.Chasers) == cfg.Chasers {
			break
		}
		if dist[p.Y][p.X] < 10 || used[p] {
			continue
		}
		kind := ChaserBasic
		if len(lv.Chasers) >= cfg.Chasers-cfg.Special {
			kind = WorldRules[t.World].Special
		}
		lv.Chasers = append(lv.Chasers, ChaserSpawn{P: p, Kind: kind})
		used[p] = true
	}
	if len(lv.Chasers) < cfg.Chasers {
		return nil
	}

	// 8. Чёрные дыры — подальше от старта, выхода и кристаллов.
	for _, p := range cand {
		if len(lv.Holes) == cfg.BlackHoles {
			break
		}
		if used[p] || dist[p.Y][p.X] < 7 || cheb(p, lv.Exit) < 4 || lv.Grid[p.Y][p.X] == Wall {
			continue
		}
		ok := true
		for _, it := range lv.Items {
			if cheb(p, it) < 2 {
				ok = false
			}
		}
		for _, h := range lv.Holes {
			if cheb(p, h) < 6 {
				ok = false
			}
		}
		if ok {
			lv.Holes = append(lv.Holes, p)
			used[p] = true
		}
	}

	// 9. Проклятые предметы — рядом с настоящими, чтобы приходилось обходить.
	for _, it := range lv.Items {
		if len(lv.Cursed) == cfg.Cursed {
			break
		}
		for _, dv := range dirs {
			p := Pt{it.X + dv.X*2, it.Y + dv.Y*2}
			q := Pt{it.X + dv.X, it.Y + dv.Y}
			if lv.CellAt(p.X, p.Y) != Wall && dist[p.Y][p.X] >= 3 && !used[p] && !used[q] && p != lv.Exit {
				lv.Cursed = append(lv.Cursed, p)
				used[p] = true
				break
			}
		}
	}

	// 10. Исчезающие бонусные предметы.
	for _, p := range cand {
		if len(lv.Golden) == cfg.Golden {
			break
		}
		if !used[p] && dist[p.Y][p.X] >= 5 && lv.Grid[p.Y][p.X] != Wall {
			lv.Golden = append(lv.Golden, p)
			used[p] = true
		}
	}

	// 11. Аптечка.
	if cfg.Heart {
		for _, p := range cand {
			if !used[p] && dist[p.Y][p.X] >= 6 {
				lv.Heart, lv.HasHeart = p, true
				break
			}
		}
	}

	lv.computePar()
	return lv
}

// computePar считает эталонное время: жадный обход «к ближайшему кристаллу»
// по кратчайшим путям (BFS), затем к выходу. Это верхняя оценка оптимального
// маршрута (задача коммивояжёра), её достаточно для медалей.
func (lv *Level) computePar() {
	cur := lv.Start
	left := append([]Pt(nil), lv.Items...)
	total := 0
	var d DistField
	for len(left) > 0 {
		lv.BFS(cur, &d)
		bi := 0
		for i, p := range left {
			if d[p.Y][p.X] < d[left[bi].Y][left[bi].X] {
				bi = i
			}
		}
		total += int(d[left[bi].Y][left[bi].X])
		cur = left[bi]
		left = append(left[:bi], left[bi+1:]...)
	}
	lv.BFS(cur, &d)
	total += int(d[lv.Exit.Y][lv.Exit.X])

	lv.ParTicks = int(float64(total)/PlayerSpeed*TPS*1.1) + TPS/2
	limit := float64(lv.ParTicks) * 2.6
	if limit < 35*TPS {
		limit = 35 * TPS
	}
	limit *= Difficulties[lv.Track.Diff].TimeMul
	lv.TimeLimit = (int(limit)/TPS + 1) * TPS
}
