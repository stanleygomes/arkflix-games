package hooks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/hooks"
)

func script(t *testing.T, dir, event, name, body string, exec bool) {
	t.Helper()
	path := filepath.Join(dir, event, name)
	os.MkdirAll(filepath.Dir(path), 0o755)
	mode := os.FileMode(0o644)
	if exec {
		mode = 0o755
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), mode); err != nil {
		t.Fatal(err)
	}
}

func TestRunOrderArgsAndLog(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	r := &hooks.Runner{Dir: filepath.Join(dir, "scripts"), Log: filepath.Join(dir, "logs", "scripts.log")}
	if err := r.Prepare(); err != nil {
		t.Fatal(err)
	}
	script(t, r.Dir, hooks.GameEnd, "20-segundo.sh", `echo "2 $*" >> `+out, true)
	script(t, r.Dir, hooks.GameEnd, "10-primeiro.sh", `echo "1 $*" >> `+out+`; echo oi do script`, true)
	script(t, r.Dir, hooks.GameEnd, "30-sem-permissao.sh", `echo nunca >> `+out, false)
	script(t, r.Dir, hooks.GameEnd, "40-falha.sh", `exit 7`, true)

	if got := r.Scripts()[hooks.GameEnd]; len(got) != 3 {
		t.Fatalf("scripts executáveis: %v", got)
	}
	r.Run(hooks.GameEnd, "/roms/smw.zip", "Super Mario World", "snes", "120")
	r.Wait()
	data, _ := os.ReadFile(out)
	want := "1 /roms/smw.zip Super Mario World snes 120\n2 /roms/smw.zip Super Mario World snes 120\n"
	if string(data) != want {
		t.Fatalf("ordem/argumentos:\n%s\nwant:\n%s", data, want)
	}
	log, _ := os.ReadFile(r.Log)
	for _, s := range []string{"10-primeiro.sh ok", "oi do script", "40-falha.sh falhou"} {
		if !strings.Contains(string(log), s) {
			t.Errorf("log sem %q:\n%s", s, log)
		}
	}
}

func TestTimeoutDoesNotHang(t *testing.T) {
	dir := t.TempDir()
	r := &hooks.Runner{Dir: dir, Log: filepath.Join(dir, "log"), Timeout: 300 * time.Millisecond}
	script(t, dir, hooks.GameStart, "lento.sh", "sleep 30", true)
	start := time.Now()
	r.Run(hooks.GameStart)
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("Run devia voltar na hora")
	}
	r.Wait()
	if time.Since(start) > 5*time.Second {
		t.Fatal("timeout não interrompeu o script")
	}
	log, _ := os.ReadFile(r.Log)
	if !strings.Contains(string(log), "interrompido") {
		t.Fatalf("log devia registrar o timeout: %s", log)
	}
}

func TestNoScripts(t *testing.T) {
	r := &hooks.Runner{Dir: filepath.Join(t.TempDir(), "nao-existe")}
	r.Run(hooks.GameEnd, "x")
	r.Wait()
	if got := r.Scripts(); len(got[hooks.GameEnd]) != 0 {
		t.Fatal("pasta ausente devia dar lista vazia")
	}
}

func TestWindowsScripts(t *testing.T) {
	for name, want := range map[string]bool{"backup.ps1": true, "a.BAT": true, "x.cmd": true, "y.exe": true, "leia.txt": false, "z.sh": false} {
		if hooks.Runnable("windows", name, 0o644) != want {
			t.Errorf("windows %s: %v", name, !want)
		}
	}
	if !hooks.Runnable("linux", "z.sh", 0o755) || hooks.Runnable("darwin", "z.sh", 0o644) {
		t.Error("linux/mac: bit de execução")
	}
	if got := hooks.Command("windows", `C:\s\b.ps1`, []string{"rom"}); got[0] != "powershell" || got[len(got)-1] != "rom" {
		t.Errorf("ps1: %v", got)
	}
	if got := hooks.Command("windows", `C:\s\b.bat`, nil); got[0] != "cmd" || got[1] != "/C" {
		t.Errorf("bat: %v", got)
	}
	if got := hooks.Command("linux", "/s/b.sh", []string{"x"}); len(got) != 2 || got[0] != "/s/b.sh" {
		t.Errorf("linux: %v", got)
	}
}
