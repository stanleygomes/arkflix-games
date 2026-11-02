package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
)

// BTInfo é a tela de Bluetooth: o rádio e os aparelhos (controles primeiro).
type BTInfo struct {
	Supported bool        `json:"supported"`
	Powered   bool        `json:"powered"`
	Scanning  bool        `json:"scanning"`
	Reason    string      `json:"reason,omitempty"` // por que não dá pra parear pelo app
	Devices   []bt.Device `json:"devices"`
}

// btState: a busca que o app pediu e a última lista mandada pra interface.
type btState struct {
	mu      sync.Mutex
	until   time.Time // fim da busca (zero: parada)
	looping bool      // o laço da busca está rodando
	last    string    // última BTInfo emitida (só avisa quando muda)
}

const (
	btScanDefault = 20 * time.Second
	btScanMax     = 60 * time.Second
	btPairTimeout = 60 * time.Second
	btConnTimeout = 30 * time.Second
)

var errBTOff = errors.New("ligue o Bluetooth primeiro")

// BluetoothStatus diz se dá pra parear pelo app e lista os aparelhos.
func (s *Service) BluetoothStatus() BTInfo {
	a := s.opt.Bluetooth.Adapter()
	info := BTInfo{Supported: a.Supported, Powered: a.Powered, Reason: a.Reason, Devices: []bt.Device{}}
	info.Scanning = a.Powered && (a.Discovering || s.btScanning())
	if a.Supported && a.Powered {
		if ds, err := s.opt.Bluetooth.Devices(); err == nil {
			info.Devices = ds
		}
	}
	return info
}

func (s *Service) btScanning() bool {
	s.bts.mu.Lock()
	defer s.bts.mu.Unlock()
	return !s.bts.until.IsZero()
}

// emitBT manda o evento "bluetooth" (sem force, só se algo mudou).
func (s *Service) emitBT(force bool) {
	info := s.BluetoothStatus()
	raw, _ := json.Marshal(info)
	s.bts.mu.Lock()
	changed := string(raw) != s.bts.last
	s.bts.last = string(raw)
	s.bts.mu.Unlock()
	if force || changed {
		s.opt.Emitter.Emit("bluetooth", info)
	}
}

// ready: o rádio existe e está ligado.
func (s *Service) btReady() error {
	a := s.opt.Bluetooth.Adapter()
	if !a.Supported {
		if a.Reason != "" {
			return errors.New(a.Reason)
		}
		return bt.ErrUnsupported
	}
	if !a.Powered {
		return errBTOff
	}
	return nil
}

// BluetoothScan busca aparelhos por alguns segundos (0 = 20 s, máx. 60) em
// segundo plano; a lista chega pelo evento "bluetooth". Chamar de novo estende.
func (s *Service) BluetoothScan(seconds int) error {
	if err := s.btReady(); err != nil {
		return err
	}
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		d = btScanDefault
	}
	d = min(d, btScanMax)
	if err := s.opt.Bluetooth.StartDiscovery(); err != nil {
		return err
	}
	s.bts.mu.Lock()
	s.bts.until = time.Now().Add(d)
	start := !s.bts.looping
	s.bts.looping = true
	s.bts.mu.Unlock()
	if start {
		go s.btScanLoop()
	}
	s.emitBT(true)
	return nil
}

// btScanLoop confere a lista a cada segundo e para a busca no fim do prazo.
func (s *Service) btScanLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		s.bts.mu.Lock()
		done := s.bts.until.IsZero() || time.Now().After(s.bts.until)
		if done {
			s.bts.until, s.bts.looping = time.Time{}, false
		}
		s.bts.mu.Unlock()
		if done {
			s.opt.Bluetooth.StopDiscovery()
			s.emitBT(true)
			return
		}
		s.emitBT(false)
	}
}

// btStopScan encerra a busca: parear com ela ligada falha em muito adaptador.
func (s *Service) btStopScan() {
	s.bts.mu.Lock()
	s.bts.until = time.Time{}
	s.bts.mu.Unlock()
	s.opt.Bluetooth.StopDiscovery()
}

// BluetoothPower liga ou desliga o rádio (vale pro console todo).
func (s *Service) BluetoothPower(on bool) error {
	if err := s.adult(); err != nil {
		return err
	}
	if !s.opt.Bluetooth.Adapter().Supported {
		return s.btReady()
	}
	if !on {
		s.btStopScan()
	}
	err := s.opt.Bluetooth.SetPowered(on)
	s.emitBT(true)
	return err
}

// btDevice valida o endereço e acha o aparelho na lista.
func (s *Service) btDevice(address string) (bt.Device, error) {
	addr, err := bt.ParseAddress(address)
	if err != nil {
		return bt.Device{}, err
	}
	if err := s.btReady(); err != nil {
		return bt.Device{}, err
	}
	ds, err := s.opt.Bluetooth.Devices()
	if err != nil {
		return bt.Device{}, err
	}
	for _, d := range ds {
		if d.Address == addr {
			return d, nil
		}
	}
	return bt.Device{}, errors.New("aparelho não encontrado: busque de novo com o controle em modo de pareamento")
}

// BluetoothPair pareia, confia (o controle volta sozinho depois) e conecta,
// em segundo plano: progresso pela tarefa "bt:<endereço>" e evento "bluetooth".
func (s *Service) BluetoothPair(address string) error {
	if err := s.adult(); err != nil {
		return err
	}
	d, err := s.btDevice(address)
	if err != nil {
		return err
	}
	if !s.runTask("bt:"+d.Address, "Parear "+d.Name, func(progress func(float64)) (string, error) {
		return s.btPair(d, progress)
	}) {
		return errors.New("já estou pareando esse controle")
	}
	return nil
}

func (s *Service) btPair(d bt.Device, progress func(float64)) (string, error) {
	b := s.opt.Bluetooth
	defer s.emitBT(true)
	s.btStopScan()
	progress(0.1)
	ctx, cancel := context.WithTimeout(context.Background(), btPairTimeout)
	defer cancel()
	if err := b.Pair(ctx, d.Address); err != nil {
		return "", err
	}
	s.emitBT(true)
	progress(0.6)
	if err := b.SetTrusted(d.Address, true); err != nil {
		return "", err
	}
	progress(0.8)
	cctx, ccancel := context.WithTimeout(context.Background(), btConnTimeout)
	defer ccancel()
	if err := b.Connect(cctx, d.Address); err != nil {
		// pareado já vale: muito controle só conecta ao apertar o botão
		return d.Name + " pareado. Se não conectar, aperte o botão central do controle.", nil
	}
	return d.Name + " pareado e conectado", nil
}

// BluetoothForget esquece o aparelho (pra parear de novo do zero).
func (s *Service) BluetoothForget(address string) error {
	if err := s.adult(); err != nil {
		return err
	}
	return s.btDo(address, s.opt.Bluetooth.Remove)
}

// BluetoothConnect conecta um aparelho já pareado.
func (s *Service) BluetoothConnect(address string) error {
	return s.btDo(address, func(addr string) error {
		ctx, cancel := context.WithTimeout(context.Background(), btConnTimeout)
		defer cancel()
		return s.opt.Bluetooth.Connect(ctx, addr)
	})
}

// BluetoothDisconnect desconecta sem esquecer (o controle volta ao ligar).
func (s *Service) BluetoothDisconnect(address string) error {
	return s.btDo(address, s.opt.Bluetooth.Disconnect)
}

func (s *Service) btDo(address string, fn func(addr string) error) error {
	d, err := s.btDevice(address)
	if err != nil {
		return err
	}
	err = fn(d.Address)
	s.emitBT(true)
	return err
}

// OpenBluetoothSettings abre a tela de Bluetooth do sistema (plano B pra
// Windows, macOS e controles que o app não pareia).
func (s *Service) OpenBluetoothSettings() error {
	if err := s.adult(); err != nil {
		return err
	}
	return s.opt.Bluetooth.OpenSettings()
}
