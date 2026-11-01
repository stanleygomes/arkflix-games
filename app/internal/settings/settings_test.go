package settings_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

func TestDefaultsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	def := settings.Defaults("/home/x", "/bin", "/cores")
	if len(def.LibraryDirs) != 1 || !def.LibraryDirs[0].Writable || def.ImportDir != "/home/x/Games/roms" {
		t.Fatalf("padrão: %+v", def.LibraryDirs)
	}
	s, err := settings.Open(dir, def)
	if err != nil {
		t.Fatal(err)
	}
	next := s.Settings()
	next.A11y.HighContrast = true
	next.A11y.MusicVolume = 300
	saved, err := s.Save(next)
	if err != nil || saved.A11y.MusicVolume != 100 {
		t.Fatalf("validação do volume: %+v %v", saved.A11y, err)
	}
	again, err := settings.Open(dir, def)
	if err != nil || !again.Settings().A11y.HighContrast {
		t.Fatal("configuração não persistiu")
	}
	info, _ := os.Stat(filepath.Join(dir, "settings.json"))
	if info.Mode().Perm() != 0o600 {
		t.Errorf("settings.json devia ser 600, é %v", info.Mode().Perm())
	}
	reset, _ := again.Reset()
	if reset.A11y.HighContrast {
		t.Error("reset não voltou ao padrão")
	}
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{quebrado"), 0o600)
	if _, err := settings.Open(dir, settings.Defaults("/h", "", "")); err == nil {
		t.Fatal("arquivo corrompido devia dar erro (não sobrescrever calado)")
	}
}

func TestStateConcurrent(t *testing.T) {
	s, _ := settings.Open(t.TempDir(), settings.Defaults("/h", "", ""))
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.RecordPlay("jogo", time.Duration(i)*time.Second, time.Now())
			_ = s.State().PlayTime["jogo"]
		}()
	}
	wg.Wait()
	if got := s.State().PlayTime["jogo"]; got != 190 {
		t.Fatalf("tempo somado: %d, esperado 190", got)
	}
}

func TestScreensaverValidate(t *testing.T) {
	s := settings.Settings{Screensaver: settings.Screensaver{Minutes: 0, Mode: "xpto"}}
	s.Validate()
	if s.Screensaver.Minutes != 5 || s.Screensaver.Mode != "snake" {
		t.Fatalf("corrige valores ruins: %+v", s.Screensaver)
	}
	s.Screensaver = settings.Screensaver{Minutes: 999, Mode: "showcase"}
	s.Validate()
	if s.Screensaver.Minutes != 60 || s.Screensaver.Mode != "showcase" {
		t.Fatalf("limite de 60 min: %+v", s.Screensaver)
	}
	d := settings.Defaults("/h", "", "")
	if !d.Screensaver.Enabled || d.Screensaver.Minutes != 5 {
		t.Fatalf("padrão: %+v", d.Screensaver)
	}
}
