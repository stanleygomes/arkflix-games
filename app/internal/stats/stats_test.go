package stats_test

import (
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/stats"
)

func TestCompute(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 11, 1, 20, 0, 0, 0, loc) // domingo
	day := func(daysAgo, hour int) int64 {
		return now.AddDate(0, 0, -daysAgo).Add(time.Duration(hour-20) * time.Hour).Unix()
	}
	sessions := []history.Session{
		{Game: "smw", Start: day(0, 10), Seconds: 3600},
		{Game: "smw", Start: day(1, 10), Seconds: 1800},
		{Game: "smw", Start: day(1, 11), Seconds: 5}, // abriu e fechou: tempo conta, partida não
		{Game: "sonic", Start: day(3, 10), Seconds: 600},
		{Game: "sonic", Start: day(4, 10), Seconds: 9999, Code: 1}, // erro: ignorada
		{Game: "fifa", Start: day(40, 10), Seconds: 7200},          // fora de 30 dias
		{Game: "apagado", Name: "Jogo Apagado", System: "nes", Start: day(2, 10), Seconds: 60},
	}
	games := []stats.Game{
		{ID: "smw", Name: "Super Mario World", System: "snes"},
		{ID: "mmx", Name: "Mega Man X", System: "snes"},
		{ID: "sonic", Name: "Sonic 2", System: "genesis"},
		{ID: "fifa", Name: "FIFA 07", System: "ps2"},
	}
	log := &history.Log{}

	all := stats.Compute(log, sessions, games, "all", now, loc)
	if all.TotalSeconds != 3600+1800+5+600+7200+60 || all.Sessions != 6 || all.Plays != 5 {
		t.Fatalf("total: %+v", all)
	}
	if all.ByGame[0].ID != "fifa" || all.GamesPlayed != 4 || all.LibraryGames != 4 {
		t.Fatalf("ranking: %+v", all.ByGame)
	}

	m := stats.Compute(log, sessions, games, "30d", now, loc)
	if m.TotalSeconds != 3600+1800+5+600+60 || m.Plays != 4 {
		t.Fatalf("30 dias: %+v", m)
	}
	if m.ByGame[0].Name != "Super Mario World" || m.ByGame[0].Plays != 2 || m.ByGame[0].Seconds != 5405 {
		t.Fatalf("jogo 1: %+v", m.ByGame[0])
	}
	if m.ByGame[2].Name != "Jogo Apagado" {
		t.Fatalf("jogo apagado usa o nome da sessão: %+v", m.ByGame)
	}
	var snes, ps2 stats.SystemStat
	for _, s := range m.BySystem {
		switch s.System {
		case "snes":
			snes = s
		case "ps2":
			ps2 = s
		}
	}
	if snes.Games != 2 || snes.Played != 1 || snes.Seconds != 5405 {
		t.Fatalf("snes: %+v", snes)
	}
	if ps2.Games != 1 || ps2.Played != 0 || ps2.Seconds != 0 {
		t.Fatalf("console sem jogo no período aparece zerado: %+v", ps2)
	}
	if m.ByWeekday[time.Sunday] != 3600 || m.ByWeekday[time.Saturday] != 1805 {
		t.Fatalf("dia da semana: %v", m.ByWeekday)
	}
	if len(m.ByDay) != 30 || m.ByDay[29].Date != "2026-11-01" || m.ByDay[29].Seconds != 3600 {
		t.Fatalf("por dia: %+v", m.ByDay[29])
	}
	if m.BestDay.Date != "2026-11-01" {
		t.Fatalf("melhor dia: %+v", m.BestDay)
	}

	w := stats.Compute(log, sessions, games, "7d", now, loc)
	if w.GamesPlayed != 3 {
		t.Fatalf("7 dias: %+v", w)
	}
	empty := stats.Compute(log, nil, games, "7d", now, loc)
	if empty.TotalSeconds != 0 || len(empty.ByGame) != 0 || len(empty.BySystem) != 3 {
		t.Fatalf("sem sessões: %+v", empty)
	}
}
