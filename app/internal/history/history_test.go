package history_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/history"
)

func TestAppendAndPlays(t *testing.T) {
	l := &history.Log{Path: filepath.Join(t.TempDir(), "sessions.jsonl")}
	all, err := l.All()
	if err != nil || len(all) != 0 {
		t.Fatalf("sem arquivo: %v %v", all, err)
	}
	sessions := []history.Session{
		{Game: "smw", Seconds: 600, Mode: "start"},
		{Game: "smw", Seconds: 3, Mode: "start"},           // abriu e fechou: não conta
		{Game: "smw", Seconds: 900, Mode: "auto", Code: 1}, // emulador quebrou: não conta
		{Game: "sonic", Seconds: 10, Mode: "slot"},
	}
	for _, s := range sessions {
		if err := l.Append(s); err != nil {
			t.Fatal(err)
		}
	}
	all, _ = l.All()
	if len(all) != 4 {
		t.Fatalf("sessões: %d", len(all))
	}
	plays := l.Plays(all)
	if plays["smw"] != 1 || plays["sonic"] != 1 {
		t.Fatalf("partidas: %v", plays)
	}
	info, _ := os.Stat(l.Path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("histórico devia ser 600: %v", info.Mode().Perm())
	}
}

func TestCorruptLineSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.jsonl")
	os.WriteFile(path, []byte(`{"game":"a","seconds":60}`+"\n"+`{"game":"b","sec`+"\n"), 0o600)
	l := &history.Log{Path: path}
	l.Append(history.Session{Game: "c", Seconds: 60})
	all, err := l.All()
	if err != nil || len(all) != 2 || all[1].Game != "c" {
		t.Fatalf("linha corrompida devia ser pulada: %+v %v", all, err)
	}
}

func TestMinPlayConfigurable(t *testing.T) {
	l := &history.Log{MinPlay: -1}
	if !l.Counts(history.Session{Game: "x", Seconds: 0}) {
		t.Fatal("MinPlay negativo = conta tudo (modo de teste)")
	}
	l = &history.Log{MinPlay: time.Minute}
	if l.Counts(history.Session{Game: "x", Seconds: 59}) {
		t.Fatal("59 s não devia contar com mínimo de 1 min")
	}
}

func TestConcurrentAppend(t *testing.T) {
	l := &history.Log{Path: filepath.Join(t.TempDir(), "s.jsonl")}
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Append(history.Session{Game: "g", Seconds: 30})
		}()
	}
	wg.Wait()
	all, _ := l.All()
	if len(all) != 50 {
		t.Fatalf("escritas concorrentes perderam linha: %d", len(all))
	}
}

func TestPlayedBetween(t *testing.T) {
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local)
	sessions := []history.Session{
		{Game: "a", Start: day.Add(-10 * time.Minute).Unix(), Seconds: 30 * 60}, // 20 min depois da meia-noite
		{Game: "b", Start: day.Add(9 * time.Hour).Unix(), Seconds: 15 * 60},
		{Game: "c", Start: day.Add(-48 * time.Hour).Unix(), Seconds: 3600}, // outro dia
	}
	if got := history.PlayedBetween(sessions, day, day.Add(24*time.Hour)); got != 35*time.Minute {
		t.Fatalf("jogado no dia: %s", got)
	}
}
