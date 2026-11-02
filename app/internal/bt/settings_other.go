//go:build !linux

package bt

import (
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// New: fora do Linux o app não pareia (as APIs nativas ficam de fora); a tela
// mostra o motivo e o botão das configurações do sistema.
func New() Backend {
	switch runtime.GOOS {
	case "windows":
		return Unsupported{Reason: "No Windows, pareie o controle em Configurações > Bluetooth. O botão abaixo abre essa tela."}
	case "darwin":
		return Unsupported{Reason: "No macOS, pareie o controle em Ajustes do Sistema > Bluetooth. O botão abaixo abre essa tela."}
	}
	return Unsupported{Reason: "Parear pelo app ainda não funciona neste sistema: use as configurações de Bluetooth dele."}
}

func openSettings() error {
	switch runtime.GOOS {
	case "windows":
		// rundll32 não pisca janela de console como o "cmd /c start"
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", "ms-settings:bluetooth").Start()
	case "darwin":
		return exec.Command("open", macSettingsURL(macMajor())).Start()
	}
	return errors.New("não sei abrir as configurações de Bluetooth deste sistema")
}

// macMajor: versão principal do macOS (0 se não deu pra saber).
func macMajor() int {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return 0
	}
	major, _, _ := strings.Cut(strings.TrimSpace(string(out)), ".")
	n, _ := strconv.Atoi(major)
	return n
}
