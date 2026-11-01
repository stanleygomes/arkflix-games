// Package history guarda cada sessão de jogo num arquivo JSON Lines que só
// cresce (sessions.jsonl). É a base do contador de partidas e das estatísticas.
package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

// Session é uma vez que o jogo foi aberto e fechado.
type Session struct {
	Game    string `json:"game"`
	System  string `json:"system"`
	Name    string `json:"name"`
	Start   int64  `json:"start"`   // unix
	Seconds int64  `json:"seconds"` // duração
	Mode    string `json:"mode"`    // start, auto (continuar), slot (save state)
	Code    int    `json:"code"`    // código de saída do emulador
}

// Log lê e grava o histórico.
type Log struct {
	Path string
	// MinPlay é a duração mínima pra contar como partida (abriu e fechou por
	// engano não conta). Padrão 10 s.
	MinPlay time.Duration

	mu sync.Mutex
}

func (l *Log) min() int64 {
	if l.MinPlay < 0 {
		return 0
	}
	if l.MinPlay == 0 {
		return 10
	}
	return int64(l.MinPlay.Seconds())
}

// Counts reports whether the session counts as a play.
func (l *Log) Counts(s Session) bool { return s.Seconds >= l.min() && s.Code == 0 }

// Append acrescenta uma sessão.
func (l *Log) Append(s Session) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// All lê todas as sessões. Linha corrompida (desligou no meio da escrita) é pulada.
func (l *Log) All() ([]Session, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.Open(l.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := []Session{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		var s Session
		if json.Unmarshal(sc.Bytes(), &s) == nil && s.Game != "" {
			out = append(out, s)
		}
	}
	return out, sc.Err()
}

// Plays conta as partidas de cada jogo.
func (l *Log) Plays(sessions []Session) map[string]int {
	out := map[string]int{}
	for _, s := range sessions {
		if l.Counts(s) {
			out[s.Game]++
		}
	}
	return out
}
