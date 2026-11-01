// Package stats agrega o histórico de sessões: tempo, partidas, por jogo, por
// console, por dia da semana e por dia. Função pura sobre os dados.
package stats

import (
	"sort"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/history"
)

// Game é o que o cálculo precisa saber de cada jogo da biblioteca.
type Game struct {
	ID     string
	Name   string
	System string
}

// GameStat é um jogo no ranking.
type GameStat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	System  string `json:"system"`
	Seconds int64  `json:"seconds"`
	Plays   int    `json:"plays"`
	Last    int64  `json:"last"`
}

// SystemStat é um console.
type SystemStat struct {
	System  string `json:"system"`
	Seconds int64  `json:"seconds"`
	Plays   int    `json:"plays"`
	Games   int    `json:"games"`  // jogos na biblioteca
	Played  int    `json:"played"` // jogos jogados no período
}

// DayStat é um dia (AAAA-MM-DD).
type DayStat struct {
	Date    string `json:"date"`
	Seconds int64  `json:"seconds"`
}

// Stats é o resumo do período.
type Stats struct {
	Period       string       `json:"period"`
	TotalSeconds int64        `json:"totalSeconds"`
	Plays        int          `json:"plays"`
	Sessions     int          `json:"sessions"`
	GamesPlayed  int          `json:"gamesPlayed"`
	LibraryGames int          `json:"libraryGames"`
	ByGame       []GameStat   `json:"byGame"`
	BySystem     []SystemStat `json:"bySystem"`
	ByWeekday    [7]int64     `json:"byWeekday"` // domingo = 0
	ByDay        []DayStat    `json:"byDay"`     // últimos 30 dias, do mais antigo pro mais novo
	BestDay      DayStat      `json:"bestDay"`
}

// Since devolve o início do período ("7d", "30d", "all").
func Since(period string, now time.Time) int64 {
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7).Unix()
	case "30d":
		return now.AddDate(0, 0, -30).Unix()
	}
	return 0
}

// Compute agrega as sessões do período. Sessão com erro do emulador não
// conta; tempo conta toda sessão sem erro, partida só as que valem (Log.Counts).
func Compute(log *history.Log, sessions []history.Session, games []Game, period string, now time.Time, loc *time.Location) Stats {
	if loc == nil {
		loc = time.Local
	}
	since := Since(period, now)
	names := map[string]Game{}
	librarySystems := map[string]int{}
	for _, g := range games {
		names[g.ID] = g
		librarySystems[g.System]++
	}
	byGame := map[string]*GameStat{}
	bySystem := map[string]*SystemStat{}
	playedBySystem := map[string]map[string]bool{}
	byDay := map[string]int64{}
	out := Stats{Period: period, LibraryGames: len(games), ByGame: []GameStat{}, BySystem: []SystemStat{}}
	for _, s := range sessions {
		if s.Start < since || s.Code != 0 {
			continue
		}
		out.Sessions++
		out.TotalSeconds += s.Seconds
		counts := log.Counts(s)
		if counts {
			out.Plays++
		}
		gs, ok := byGame[s.Game]
		if !ok {
			name, system := s.Name, s.System
			if g, found := names[s.Game]; found {
				name, system = g.Name, g.System
			}
			gs = &GameStat{ID: s.Game, Name: name, System: system}
			byGame[s.Game] = gs
		}
		gs.Seconds += s.Seconds
		if counts {
			gs.Plays++
		}
		gs.Last = max(gs.Last, s.Start)
		ss, ok := bySystem[gs.System]
		if !ok {
			ss = &SystemStat{System: gs.System}
			bySystem[gs.System] = ss
			playedBySystem[gs.System] = map[string]bool{}
		}
		ss.Seconds += s.Seconds
		if counts {
			ss.Plays++
		}
		playedBySystem[gs.System][s.Game] = true
		t := time.Unix(s.Start, 0).In(loc)
		out.ByWeekday[t.Weekday()] += s.Seconds
		byDay[t.Format("2006-01-02")] += s.Seconds
	}
	for _, gs := range byGame {
		out.ByGame = append(out.ByGame, *gs)
	}
	sort.Slice(out.ByGame, func(i, j int) bool {
		if out.ByGame[i].Seconds != out.ByGame[j].Seconds {
			return out.ByGame[i].Seconds > out.ByGame[j].Seconds
		}
		return out.ByGame[i].Name < out.ByGame[j].Name
	})
	out.GamesPlayed = len(byGame)
	for system, n := range librarySystems {
		if _, ok := bySystem[system]; !ok {
			bySystem[system] = &SystemStat{System: system}
			playedBySystem[system] = map[string]bool{}
		}
		bySystem[system].Games = n
	}
	for system, ss := range bySystem {
		ss.Played = len(playedBySystem[system])
		out.BySystem = append(out.BySystem, *ss)
	}
	sort.Slice(out.BySystem, func(i, j int) bool {
		if out.BySystem[i].Seconds != out.BySystem[j].Seconds {
			return out.BySystem[i].Seconds > out.BySystem[j].Seconds
		}
		return out.BySystem[i].System < out.BySystem[j].System
	})
	today := now.In(loc)
	for i := 29; i >= 0; i-- {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		out.ByDay = append(out.ByDay, DayStat{Date: d, Seconds: byDay[d]})
	}
	for d, sec := range byDay {
		if sec > out.BestDay.Seconds || (sec == out.BestDay.Seconds && d > out.BestDay.Date) {
			out.BestDay = DayStat{Date: d, Seconds: sec}
		}
	}
	return out
}
