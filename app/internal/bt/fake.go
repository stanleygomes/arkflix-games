package bt

import (
	"context"
	"errors"
	"sync"
)

// Endereços dos aparelhos do Fake (testes de ponta a ponta usam estes).
const (
	FakeXbox       = "9C:AA:1B:00:00:01" // aparece na busca, não pareado
	FakeEightBitDo = "E4:17:D8:00:00:02" // já pareado e conectado
)

// Fake é um Bluetooth de mentira, determinístico, pro modo de testes: nunca
// toca no rádio de verdade.
type Fake struct {
	mu       sync.Mutex
	powered  bool
	scanning bool
	seen     bool // a busca já achou o controle não pareado
	devs     map[string]*Device
}

// NewFake liga o rádio falso com o 8BitDo pareado; o Xbox só aparece na busca.
func NewFake() *Fake {
	return &Fake{powered: true, devs: map[string]*Device{
		FakeEightBitDo: {Address: FakeEightBitDo, Name: "8BitDo Pro 2", Paired: true, Trusted: true, Connected: true, Gamepad: true, Battery: 80},
	}}
}

func (f *Fake) Adapter() Adapter {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Adapter{Supported: true, Powered: f.powered, Discovering: f.scanning}
}

func (f *Fake) Devices() ([]Device, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []Device{}
	if !f.powered {
		return out, nil
	}
	for _, d := range f.devs {
		out = append(out, *d)
	}
	Sort(out)
	return out, nil
}

func (f *Fake) SetPowered(on bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.powered = on
	if !on {
		f.scanning = false
	}
	return nil
}

func (f *Fake) StartDiscovery() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.powered {
		return errOff
	}
	f.scanning = true
	if _, ok := f.devs[FakeXbox]; !ok {
		f.devs[FakeXbox] = &Device{Address: FakeXbox, Name: "Xbox Wireless Controller", Gamepad: true, Battery: -1}
	}
	return nil
}

func (f *Fake) StopDiscovery() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scanning = false
	return nil
}

// errOff: mesmo texto do BlueZ traduzido (rádio desligado).
var errOff = errors.New("o Bluetooth está desligado")

// dev acha o aparelho com o rádio ligado (chamar com f.mu travado).
func (f *Fake) dev(addr string) (*Device, error) {
	if !f.powered {
		return nil, errOff
	}
	d, ok := f.devs[addr]
	if !ok {
		return nil, errNotFound
	}
	return d, nil
}

func (f *Fake) Pair(_ context.Context, addr string) error {
	return f.change(addr, func(d *Device) { d.Paired = true })
}

func (f *Fake) SetTrusted(addr string, on bool) error {
	return f.change(addr, func(d *Device) { d.Trusted = on })
}

func (f *Fake) Connect(_ context.Context, addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.dev(addr)
	if err != nil {
		return err
	}
	if !d.Paired {
		return errors.New("pareie o controle antes de conectar")
	}
	d.Connected = true
	if d.Battery < 0 {
		d.Battery = 100
	}
	return nil
}

func (f *Fake) Disconnect(addr string) error {
	return f.change(addr, func(d *Device) { d.Connected = false })
}

// Remove esquece o aparelho; o Xbox volta a aparecer na próxima busca.
func (f *Fake) Remove(addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, err := f.dev(addr); err != nil {
		return err
	}
	delete(f.devs, addr)
	return nil
}

func (f *Fake) OpenSettings() error { return nil }

func (f *Fake) change(addr string, fn func(*Device)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.dev(addr)
	if err != nil {
		return err
	}
	fn(d)
	return nil
}

var errNotFound = errors.New("aparelho não encontrado: busque de novo com o controle em modo de pareamento")
