package controls

import "testing"

func TestDefaultsHaveNoGapsOrDuelClashes(t *testing.T) {
	all := Defaults()
	for l := 0; l < LayoutCount; l++ {
		if e := all[l].Empty(); len(e) > 0 {
			t.Errorf("%s: нет клавиш для %v", LayoutNames[l], e)
		}
	}
	used := map[string]bool{}
	for _, l := range []int{Duel1, Duel2} {
		for a := Action(0); a < ActionCount; a++ {
			for _, k := range all[l][a] {
				if k == "" {
					continue
				}
				if used[k] {
					t.Errorf("клавиша %s у обоих игроков дуэли", k)
				}
				used[k] = true
			}
		}
	}
}

func TestSetMovesKeyAway(t *testing.T) {
	all := Defaults()
	// Space был рывком; назначаем его на способность — с рывка он снимается.
	if taken := Set(&all, Solo, Ability, 1, "Space"); taken != "Рывок" {
		t.Fatalf("ожидалось «Рывок», получено %q", taken)
	}
	if all[Solo][Dash][0] != "" || all[Solo][Ability][1] != "Space" {
		t.Fatalf("неверная раскладка: %+v", all[Solo])
	}
	// В дуэли клавиша второго игрока уходит от него к первому.
	Set(&all, Duel1, Up, 1, "ArrowUp")
	if all[Duel2][Up][0] != "" || all[Duel1][Up][1] != "ArrowUp" {
		t.Fatal("клавиша не перешла от второго игрока к первому")
	}
	// Одиночная раскладка от дуэли не зависит.
	if all[Solo][Up][1] != "ArrowUp" {
		t.Fatal("одиночная раскладка изменилась")
	}
}

func TestMapRoundTrip(t *testing.T) {
	all := Defaults()
	if m := ToMap(all); len(m) != 0 {
		t.Fatalf("раскладки по умолчанию не должны сохраняться: %v", m)
	}
	Set(&all, Duel2, Dash, 0, "Numpad0")
	Set(&all, Solo, Left, 1, "")
	back := FromMap(ToMap(all), func(string) bool { return true })
	if back != all {
		t.Fatalf("после сохранения раскладки отличаются:\n%v\n%v", back, all)
	}
	// Неизвестные клавиши отбрасываются.
	m := map[string][]string{"solo.up": {"Nope", "W"}}
	got := FromMap(m, func(k string) bool { return k != "Nope" })
	if got[Solo][Up] != [Slots]string{"", "W"} || got[Solo][Dash] != [Slots]string{} {
		t.Fatalf("неожиданно: %+v", got[Solo])
	}
}
