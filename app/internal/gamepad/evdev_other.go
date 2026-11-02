//go:build !linux

package gamepad

// Native: fora do Linux quem lê os controles é a interface (Gamepad API do
// WebView); aqui o gerente só repassa eventos injetados (testes).
const Native = false

func (m *Manager) scan() {}
