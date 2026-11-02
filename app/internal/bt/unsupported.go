package bt

import "context"

// Unsupported é o Backend de quem não pareia pelo app (Windows, macOS, Linux
// sem BlueZ). Só abre as configurações do sistema.
type Unsupported struct{ Reason string }

func (u Unsupported) Adapter() Adapter                    { return Adapter{Reason: u.Reason} }
func (Unsupported) Devices() ([]Device, error)            { return []Device{}, nil }
func (Unsupported) SetPowered(bool) error                 { return ErrUnsupported }
func (Unsupported) StartDiscovery() error                 { return ErrUnsupported }
func (Unsupported) StopDiscovery() error                  { return nil }
func (Unsupported) Pair(context.Context, string) error    { return ErrUnsupported }
func (Unsupported) SetTrusted(string, bool) error         { return ErrUnsupported }
func (Unsupported) Connect(context.Context, string) error { return ErrUnsupported }
func (Unsupported) Disconnect(string) error               { return ErrUnsupported }
func (Unsupported) Remove(string) error                   { return ErrUnsupported }
func (Unsupported) OpenSettings() error                   { return openSettings() }

// macSettingsURL: o Ventura (13) trocou Preferências por Ajustes do Sistema e
// o id do painel de Bluetooth mudou junto.
func macSettingsURL(major int) string {
	if major >= 13 {
		return "x-apple.systempreferences:com.apple.BluetoothSettings"
	}
	return "x-apple.systempreferences:com.apple.preferences.Bluetooth"
}
