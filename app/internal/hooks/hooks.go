// Package hooks roda scripts do usuário em eventos do app (abrir e fechar
// jogo), no mesmo esquema do ES-DE: executáveis em <dir>/<evento>/, em ordem
// alfabética, em segundo plano. É assim que o backup dos saves do solomon
// dispara ao fechar um jogo, sem o app saber nada de servidor.
package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Eventos suportados.
const (
	GameStart = "game-start"
	GameEnd   = "game-end"
)

// Events lista os eventos na ordem da tela.
var Events = []string{GameStart, GameEnd}

// Runner acha e roda os scripts.
type Runner struct {
	Dir     string        // <dir>/<evento>/<script>
	Log     string        // arquivo de log (saída e erros)
	Timeout time.Duration // por script (padrão 10 min)

	mu sync.Mutex // serializa a escrita no log
	wg sync.WaitGroup
}

// Prepare cria as pastas dos eventos, pra pessoa saber onde pôr os scripts.
func (r *Runner) Prepare() error {
	for _, ev := range Events {
		if err := os.MkdirAll(filepath.Join(r.Dir, ev), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Scripts lista os executáveis de cada evento.
func (r *Runner) Scripts() map[string][]string {
	out := map[string][]string{}
	for _, ev := range Events {
		out[ev] = []string{}
		entries, err := os.ReadDir(filepath.Join(r.Dir, ev))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			info, err := e.Info()
			if err == nil && info.Mode()&0o111 != 0 {
				out[ev] = append(out[ev], e.Name())
			}
		}
		sort.Strings(out[ev])
	}
	return out
}

func (r *Runner) logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(r.Log), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(r.Log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
}

// Run roda os scripts do evento em segundo plano e volta na hora.
func (r *Runner) Run(event string, args ...string) {
	scripts := r.Scripts()[event]
	if len(scripts) == 0 {
		return
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for _, name := range scripts {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			cmd := exec.CommandContext(ctx, filepath.Join(r.Dir, event, name), args...)
			cmd.Dir = filepath.Join(r.Dir, event)
			cmd.WaitDelay = time.Second
			out, err := cmd.CombinedOutput()
			cancel()
			status := "ok"
			switch {
			case ctx.Err() == context.DeadlineExceeded:
				status = fmt.Sprintf("passou de %s e foi interrompido", timeout)
			case err != nil:
				status = "falhou: " + err.Error()
			}
			r.logf("[%s] %s %s", event, name, status)
			if text := strings.TrimSpace(string(out)); text != "" {
				r.logf("[%s] %s saída:\n%s", event, name, text)
			}
		}
	}()
}

// Wait espera os scripts em andamento (testes e saída do app).
func (r *Runner) Wait() { r.wg.Wait() }
