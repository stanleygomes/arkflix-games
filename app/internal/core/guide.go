package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/saves"
)

// gameRun é o jogo aberto: o guia fecha ele ou manda comando pro emulador.
type gameRun struct {
	id        string
	emulator  string
	stop      context.CancelFunc
	started   time.Time
	recording bool
	next      string // retomada rápida: jogo que abre quando este fechar
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
	shots, videos := u.mediaDir("screenshots"), u.mediaDir("videos")
	os.MkdirAll(shots, 0o755)
	os.MkdirAll(videos, 0o755)
	// capturas e gravações vão pra pasta do perfil (a página do jogo mostra)
	body := fmt.Sprintf("screenshot_directory = %q\nrecording_output_directory = %q\nscreenshots_in_content_dir = \"false\"\nsort_screenshots_by_content_enable = \"false\"\nauto_screenshot_filename = \"true\"\n", shots, videos)
	if guideCommands() {
		body += "network_cmd_enable = \"true\"\nnetwork_cmd_port = \"55355\"\n"
	}
	path := filepath.Join(u.dir, "retroarch-guide.cfg")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil
	}
	return []string{path}
}

// GuideInfo diz o que o guia pode fazer com o jogo aberto.
type GuideInfo struct {
	Game      string `json:"game"`      // id do jogo aberto ("" = nenhum)
	Commands  bool   `json:"commands"`  // salvar/carregar/capturar pelo guia
	Started   int64  `json:"started"`   // quando o jogo abriu (unix)
	Recording bool   `json:"recording"` // gravando vídeo
	Note      string `json:"note"`      // anotação do jogo
}

// Guide: estado do jogo aberto pro guia rápido.
func (s *Service) Guide() GuideInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.game
	if g == nil {
		return GuideInfo{}
	}
	return GuideInfo{Game: g.id, Commands: g.emulator == catalog.RetroArch && guideCommands(), Started: g.started.Unix(),
		Recording: g.recording, Note: s.cur().note(g.id)}
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
	ra := map[string]string{"save": "SAVE_STATE", "load": "LOAD_STATE", "fps": "FPS_TOGGLE", "screenshot": "SCREENSHOT", "record": "RECORDING_TOGGLE"}[cmd]
	if ra == "" {
		return errors.New("comando desconhecido")
	}
	if err := raCommand(ra); err != nil {
		return err
	}
	if cmd == "record" {
		s.mu.Lock()
		g.recording = !g.recording
		s.mu.Unlock()
	}
	if cmd != "screenshot" { // a captura é do quadro do jogo: o guia pode ficar
		s.opt.Window.Hide() // volta pro jogo
	}
	return nil
}

// SwitchGame é a retomada rápida: fecha o jogo aberto (que grava o ponto
// automático) e abre outro de onde ele parou.
func (s *Service) SwitchGame(id string) error {
	e, _, err := s.find(id)
	if err != nil {
		return err
	}
	if e.remote {
		return errors.New("este jogo está no servidor: baixe antes de jogar")
	}
	s.mu.Lock()
	g := s.game
	if g != nil && g.id != id {
		g.next = id
	}
	s.mu.Unlock()
	if g == nil {
		return s.Play(id, s.resumeState(id))
	}
	if g.id == id {
		s.opt.Window.Hide()
		return nil
	}
	return s.GameCommand("quit")
}

// resumeState: "auto" quando o jogo tem ponto automático (só RetroArch).
func (s *Service) resumeState(id string) string {
	e, sys, err := s.find(id)
	if err != nil || sys.Emulator != catalog.RetroArch {
		return ""
	}
	for _, st := range s.opt.Saves.List(saves.Game{System: sys, Launch: e.game.Launch}).States {
		if st.Slot == -1 {
			return "auto"
		}
	}
	return ""
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
