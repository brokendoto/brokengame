package sim

import "time"

// Ability — особая способность героя мира (клавиша E).
type Ability int

const (
	AbShield   Ability = iota // шахтёр: щит отбрасывает врагов
	AbMagnet                  // белочка: притягивает предметы издалека
	AbTimeStop                // ронин: враги и ловушки замирают
	AbBlink                   // пилот: телепорт вперёд, даже сквозь стену
	AbPulse                   // космонавт: импульс оглушает всё вокруг
)

// AbilityInfo — параметры способности.
type AbilityInfo struct {
	Name     string
	Hint     string
	Duration int // в шагах, 0 — мгновенная
	Cooldown int
}

var Abilities = [...]AbilityInfo{
	AbShield:   {"Щит", "отбрасывает врагов и блокирует удары", 240, 720},
	AbMagnet:   {"Магнит", "собирает предметы в радиусе 2,5 клетки", 300, 720},
	AbTimeStop: {"Стоп-время", "враги и ловушки замирают на 3 секунды", 180, 840},
	AbBlink:    {"Телепорт", "прыжок на 3 клетки вперёд, даже сквозь стену", 0, 360},
	AbPulse:    {"Импульс", "оглушает врагов и разбивает астероиды рядом", 0, 600},
}

// Rules — особенности мира: какие зоны, враги, ловушки и способность.
type Rules struct {
	Zone       Cell       // Mud — замедление, Fog — управление наоборот, Ice — скольжение
	ZoneName   string     // для подсказок
	Ability    Ability    // способность героя
	Special    ChaserKind // особый враг мира
	Rocks      bool       // катящиеся валуны/брёвна/астероиды
	Beams      bool       // обвалы/лучи по строке или столбцу
	Spikes     bool       // шипы/колючки/мины
	BlackHoles bool       // чёрные дыры
	Cursed     bool       // проклятые предметы (−5 с и сброс серии)
	Golden     bool       // исчезающие бонусные предметы

	ChaserHit, RockHit, BeamHit, SpikeHit string
}

var WorldRules = [WorldCount]Rules{
	Mine: {
		Zone: Mud, ZoneName: "грязь", Ability: AbShield, Special: ChaserCunning,
		Rocks: true, Beams: true, Spikes: true,
		ChaserHit: "Тебя догнал жук", RockHit: "Под валуном", BeamHit: "Завалило породой", SpikeHit: "Напоролся на шипы",
	},
	Forest: {
		Zone: Mud, ZoneName: "лужи", Ability: AbMagnet, Special: ChaserZigzag,
		Rocks: true, Spikes: true, Golden: true,
		ChaserHit: "Лиса оказалась быстрее", RockHit: "Сбило бревном", SpikeHit: "Застряла в колючках",
	},
	Temple: {
		Zone: Fog, ZoneName: "туман", Ability: AbTimeStop, Special: ChaserPhantom,
		Beams: true, Spikes: true, Cursed: true,
		ChaserHit: "Призрак настиг", BeamHit: "Сожжён лучом духов", SpikeHit: "Ловушка храма",
	},
	Front: {
		Zone: Ice, ZoneName: "масло", Ability: AbBlink, Special: ChaserZigzag,
		Rocks: true, Beams: true, Spikes: true,
		ChaserHit: "Дрон нашёл цель", RockHit: "Сбила катящаяся мина", BeamHit: "Под лазером", SpikeHit: "Подорвался на мине",
	},
	Space: {
		Zone: Ice, ZoneName: "невесомость", Ability: AbPulse, Special: ChaserPhantom,
		Rocks: true, Beams: true, BlackHoles: true,
		ChaserHit: "Пришелец поймал", RockHit: "Удар астероида", BeamHit: "Под лазером", SpikeHit: "",
	},
}

// DailyTrack возвращает трассу испытания дня для даты ГГГГ-ММ-ДД.
// Мир меняется каждый день по кругу.
func DailyTrack(date string) Track {
	w := Mine
	if t, err := time.Parse("2006-01-02", date); err == nil {
		w = World((t.Unix() / 86400) % int64(WorldCount))
	}
	return Track{World: w, Level: DailyLevel, Diff: Normal, Daily: date}
}
