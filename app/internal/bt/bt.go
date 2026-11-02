// Package bt pareia controles Bluetooth sem sair do app. Cada sistema tem um
// Backend: BlueZ por D-Bus no Linux, um falso pros testes e um "sem suporte"
// no Windows/macOS (lá o app só abre as configurações do sistema).
package bt

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
)

// Device é um aparelho que o adaptador conhece (pareado ou visto na busca).
type Device struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	Paired    bool   `json:"paired"`
	Connected bool   `json:"connected"`
	Trusted   bool   `json:"trusted"`
	Gamepad   bool   `json:"gamepad"`
	Battery   int    `json:"battery"` // 0–100; -1: não informado
}

// Adapter é o estado do rádio. Supported=false traz o motivo em Reason.
type Adapter struct {
	Supported   bool
	Powered     bool
	Discovering bool
	Reason      string
}

// Backend fala com o Bluetooth do sistema. Endereços chegam já validados.
type Backend interface {
	Adapter() Adapter
	Devices() ([]Device, error)
	SetPowered(on bool) error
	StartDiscovery() error
	StopDiscovery() error
	Pair(ctx context.Context, addr string) error
	SetTrusted(addr string, on bool) error
	Connect(ctx context.Context, addr string) error
	Disconnect(addr string) error
	Remove(addr string) error
	// OpenSettings abre a tela de Bluetooth do sistema (plano B).
	OpenSettings() error
}

// ErrUnsupported: o sistema não deixa parear por aqui.
var ErrUnsupported = errors.New("Bluetooth não disponível no app: use as configurações do sistema")

var addrRe = regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){5}$`)

// ParseAddress valida o endereço (XX:XX:XX:XX:XX:XX) e devolve em maiúsculas.
// Tudo que vira caminho D-Bus ou argumento passa por aqui antes.
func ParseAddress(s string) (string, error) {
	if !addrRe.MatchString(s) {
		return "", errors.New("endereço Bluetooth inválido")
	}
	return strings.ToUpper(s), nil
}

// nomes de controle conhecidos (quando o aparelho não diz o ícone)
var padNames = []string{
	"xbox", "dualshock", "dualsense", "wireless controller", "pro controller",
	"8bitdo", "joy-con", "gamepad",
}

// IsGamepad decide se o aparelho é controle: pelo ícone que o BlueZ dá
// (input-gaming) ou, sem ele, pelo nome.
func IsGamepad(name, icon string) bool {
	if icon == "input-gaming" {
		return true
	}
	n := strings.ToLower(name)
	for _, p := range padNames {
		if strings.Contains(n, p) {
			return true
		}
	}
	return false
}

// Sort: controles primeiro, depois conectados, pareados e nome.
func Sort(ds []Device) {
	rank := func(d Device) int {
		r := 0
		if !d.Gamepad {
			r += 4
		}
		if !d.Connected {
			r += 2
		}
		if !d.Paired {
			r++
		}
		return r
	}
	sort.SliceStable(ds, func(i, j int) bool {
		if a, b := rank(ds[i]), rank(ds[j]); a != b {
			return a < b
		}
		return strings.ToLower(ds[i].Name) < strings.ToLower(ds[j].Name)
	})
}
