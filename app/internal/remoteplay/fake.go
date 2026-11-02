package remoteplay

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FakeIP: o endereço que o Fake mostra pra digitar no Moonlight.
const FakeIP = "192.168.0.42"

// Fake é um Sunshine de mentira pro modo de testes: começa não instalado e
// nunca roda nada de verdade.
type Fake struct {
	mu        sync.Mutex
	installed bool
	running   bool
	onExit    func()
}

func NewFake() *Fake { return &Fake{} }

func (f *Fake) Status(context.Context) Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Status{Supported: true, Platform: "linux", AutoInstall: true, Installed: f.installed,
		Running: f.running, Started: f.running, URL: URL, PairURL: URL + "/pin", LanIP: FakeIP}
}

func (f *Fake) Install(ctx context.Context, progress func(float64)) (string, error) {
	for _, p := range []float64{0.1, 0.5, 0.9} {
		progress(p)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	f.mu.Lock()
	f.installed = true
	f.mu.Unlock()
	return "Sunshine instalado", nil
}

func (f *Fake) Start(onExit func()) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.installed {
		return errors.New("instale o Sunshine primeiro")
	}
	if f.running {
		return errors.New("o Sunshine já está ligado")
	}
	f.running, f.onExit = true, onExit
	return nil
}

func (f *Fake) Stop() error {
	f.mu.Lock()
	if !f.running {
		f.mu.Unlock()
		return errors.New("o Sunshine não foi ligado pelo app: desligue por onde ligou")
	}
	f.running = false
	done := f.onExit
	f.mu.Unlock()
	if done != nil {
		done()
	}
	return nil
}
