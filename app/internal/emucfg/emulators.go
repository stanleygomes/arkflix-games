package emucfg

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotSetUp: o emulador nunca foi aberto (sem config pra ajustar ainda). O
// primeiro jogo abre com o padrão dele; do segundo em diante, já ajustado.
var ErrNotSetUp = errors.New("emulador ainda não configurado")

// códigos de tecla do PPSSPP (Android): dispositivo 1 = teclado, 10 = controle
const (
	ppssppEsc = "1-111"
	ppssppF10 = "1-140"
)

// padCode: o primeiro código de controle (10-*) já mapeado pra um botão do PSP.
func padCode(values []string) string {
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); strings.HasPrefix(p, "10-") && !strings.Contains(p, ":") {
				return p
			}
		}
	}
	return ""
}

// PPSSPP: Esc / Select+△ = menu de pausa; F10 / Select+Start = sai; e "sair"
// no menu de pausa fecha o emulador (volta pro app) em vez de ir pro menu dele.
// Os botões do controle vêm do mapeamento que a pessoa já tem no PPSSPP.
func PPSSPP(dir string) error {
	controls := filepath.Join(dir, "controls.ini")
	general := filepath.Join(dir, "ppsspp.ini")
	for _, f := range []string{controls, general} {
		if _, err := os.Stat(f); err != nil {
			return ErrNotSetUp
		}
	}
	c, err := ReadIni(controls)
	if err != nil {
		return err
	}
	const sec = "ControlMapping"
	sel, start, tri := padCode(c.Get(sec, "Select")), padCode(c.Get(sec, "Start")), padCode(c.Get(sec, "Triangle"))
	pause, exit := ppssppEsc, ppssppF10
	if sel != "" && tri != "" {
		pause += "," + sel + ":" + tri // ":" = botões juntos
	}
	if sel != "" && start != "" {
		exit += "," + sel + ":" + start
	}
	c.Set(sec, "Pause", pause)
	c.Set(sec, "Exit App", exit)
	if err := c.Write(controls); err != nil {
		return err
	}
	g, err := ReadIni(general)
	if err != nil {
		return err
	}
	g.Set("General", "PauseMenuExitsEmulator", "True")
	return g.Write(general)
}

// PCSX2: mesmos atalhos (o app abre com -batch: desligar a máquina fecha o
// emulador) e sem a pergunta "deseja mesmo sair?".
func PCSX2(ini string) error {
	if _, err := os.Stat(ini); err != nil {
		return ErrNotSetUp
	}
	c, err := ReadIni(ini)
	if err != nil {
		return err
	}
	c.Set("Hotkeys", "OpenPauseMenu", "Keyboard/Escape", "SDL-0/Back & SDL-0/Y")
	c.Set("Hotkeys", "ShutdownVM", "Keyboard/F10", "SDL-0/Back & SDL-0/Start")
	c.Set("UI", "ConfirmShutdown", "false")
	return c.Write(ini)
}
