package core

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
)

// gameRun é o jogo aberto: o guia fecha ele ou manda comando pro emulador.
type gameRun struct {
	id       string
	emulator string
	stop     context.CancelFunc
}

// raCommandAddr: porta de comandos do RetroArch (network_cmd_port).
var raCommandAddr = "127.0.0.1:55355"

// quitGrace: quanto o RetroArch tem pra fechar sozinho antes de ser morto.
var quitGrace = 4 * time.Second

// guideCommands: só no Linux. No Windows e no macOS o firewall perguntaria
// sobre a porta na primeira vez; lá ficam os atalhos do controle.
func guideCommands() bool { return runtime.GOOS == "linux" }

// guideCfg liga a porta de comandos do RetroArch (salvar/carregar/fechar pelo
// guia). Só dura enquanto o jogo roda.
func (s *Service) guideCfg(u *user) []string {
	if !guideCommands() {
		return nil
	}
	path := filepath.Join(u.dir, "retroarch-guide.cfg")
	body := "network_cmd_enable = \"true\"\nnetwork_cmd_port = \"55355\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil
	}
	return []string{path}
}

// GuideInfo diz o que o guia pode fazer com o jogo aberto.
type GuideInfo struct {
	Game     string `json:"game"`     // id do jogo aberto ("" = nenhum)
	Commands bool   `json:"commands"` // salvar/carregar estado pelo guia
}

// Guide: estado do jogo aberto pro guia rápido.
func (s *Service) Guide() GuideInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.game == nil {
		return GuideInfo{}
	}
	return GuideInfo{Game: s.game.id, Commands: s.game.emulator == catalog.RetroArch && guideCommands()}
}

// ShowGuide traz o app pra frente por cima do jogo (botão Home do controle).
func (s *Service) ShowGuide() { s.opt.Window.Show() }

// ResumeGame esconde o app e devolve a tela pro jogo.
func (s *Service) ResumeGame() { s.opt.Window.Hide() }

// GameCommand: "save", "load" ou "quit" no jogo aberto.
func (s *Service) GameCommand(cmd string) error {
	s.mu.Lock()
	g := s.game
	s.mu.Unlock()
	if g == nil {
		return errors.New("nenhum jogo aberto")
	}
	if cmd == "quit" {
		grace := quitGrace
		if s.opt.Emulators.Bin != "" {
			grace = 300 * time.Millisecond // emulador falso (testes) não ouve a porta
		}
		if g.emulator == catalog.RetroArch && guideCommands() && raCommand("QUIT") == nil {
			// o RetroArch fecha sozinho e grava o save (matar perderia a bateria
			// do cartucho). Duas vezes: "confirmar ao sair" pede apertar de novo.
			go func() {
				time.Sleep(300 * time.Millisecond)
				raCommand("QUIT")
				time.Sleep(grace)
				time.Sleep(grace)
				g.stop()
			}()
			return nil
		}
		g.stop()
		return nil
	}
	if g.emulator != catalog.RetroArch || !guideCommands() {
		return errors.New("neste emulador, use os atalhos do controle dentro do jogo")
	}
	ra := map[string]string{"save": "SAVE_STATE", "load": "LOAD_STATE"}[cmd]
	if ra == "" {
		return errors.New("comando desconhecido")
	}
	if err := raCommand(ra); err != nil {
		return err
	}
	s.opt.Window.Hide() // volta pro jogo
	return nil
}

func raCommand(cmd string) error {
	conn, err := net.DialTimeout("udp", raCommandAddr, time.Second)
	if err != nil {
		return errors.New("o emulador não respondeu")
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd))
	return err
}
