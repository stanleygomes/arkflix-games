package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/bios"
	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/romm"
)

// BiosInfo é a BIOS de um console: o que achou, o que falta e onde pôr.
type BiosInfo struct {
	System   string      `json:"system"`
	Name     string      `json:"name"`
	Required bool        `json:"required"` // sem ela o jogo não abre
	OK       bool        `json:"ok"`
	Found    []string    `json:"found"`
	Missing  []bios.File `json:"missing"`
	Warnings []string    `json:"warnings"`
	// Fallback: joga mesmo sem a BIOS (PS1 com o core que imita ela)
	Fallback string `json:"fallback,omitempty"`
	Dir      string `json:"dir"`
}

// biosDir: as pastas ficam ao lado das de save (a mesma raiz de config do
// emulador em cada sistema): system do RetroArch, bios do PCSX2.
func (s *Service) biosDir(where string) string {
	d := s.opt.Store.Settings().Saves
	if where == bios.PCSX2Bios {
		return filepath.Join(filepath.Dir(d.PCSX2Memcards), "bios")
	}
	return filepath.Join(filepath.Dir(d.RetroArchSaves), "system")
}

func (s *Service) biosInfo(r bios.Rule) BiosInfo {
	dir := s.biosDir(r.Where)
	res := s.bios.Check(r, dir)
	name := r.System
	if sys, ok := catalog.Find(r.System); ok {
		name = sys.Name
	}
	info := BiosInfo{System: r.System, Name: name, Required: r.Required, OK: res.OK, Found: nonNil(res.Found),
		Missing: nonNil(res.Missing), Warnings: nonNil(res.Warnings), Dir: dir}
	if !res.OK && r.System == "psx" && s.coreInstalled(psxNoBios) {
		info.Fallback = "PCSX ReARMed"
	}
	return info
}

func nonNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

// BiosStatus confere a BIOS de cada console que usa uma. Barato: só relê o
// arquivo que mudou.
func (s *Service) BiosStatus() []BiosInfo {
	out := make([]BiosInfo, 0, len(bios.Rules))
	for _, r := range bios.Rules {
		out = append(out, s.biosInfo(r))
	}
	return out
}

// biosError: erro claro quando falta a BIOS obrigatória do console.
func (s *Service) biosError(system string) error {
	r, ok := bios.Find(system)
	if !ok || !r.Required {
		return nil
	}
	info := s.biosInfo(r)
	if info.OK {
		return nil
	}
	return fmt.Errorf("Falta a BIOS do %s (%s): coloque em %s ou suba no seu RomM (console › Firmware), que o app baixa sozinho", info.Name, r.Hint, info.Dir)
}

// OpenBiosFolder abre a pasta da BIOS do console no gerenciador de arquivos
// (cria se não existir).
func (s *Service) OpenBiosFolder(system string) error {
	if err := s.adult(); err != nil {
		return err
	}
	r, ok := bios.Find(strings.ToLower(system))
	if !ok {
		return fmt.Errorf("%s não usa BIOS", system)
	}
	dir := s.biosDir(r.Where)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return s.opt.Window.OpenFile(dir)
}

// syncBios baixa do RomM as BIOS que faltam neste PC (a pessoa sobe uma vez
// no servidor e todo aparelho recebe). Devolve quantas baixou.
func (s *Service) syncBios(ctx context.Context) (int, error) {
	c, _ := s.remote.client()
	if c == nil {
		return 0, nil
	}
	var want []bios.Rule
	for _, r := range bios.Rules {
		if info := s.biosInfo(r); !info.OK {
			want = append(want, r)
		}
	}
	if len(want) == 0 {
		return 0, nil
	}
	list, err := c.Firmwares(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range want {
		dir := s.biosDir(r.Where)
		for _, f := range list {
			if f.Platform != r.System || !biosWanted(r, f) {
				continue
			}
			dest := filepath.Join(dir, f.FileName)
			if _, err := os.Stat(dest); err == nil {
				continue // nunca troca um arquivo que já está aqui
			}
			if err := c.FetchFirmware(ctx, f, dest); err != nil {
				return n, err
			}
			if r.Mode == bios.PS2 && !bios.IsPS2(dest) {
				os.Remove(dest) // não era BIOS de PS2 (outro arquivo do console)
				continue
			}
			n++
		}
	}
	return n, nil
}

// biosWanted: o arquivo do servidor é uma das BIOS da regra?
func biosWanted(r bios.Rule, f romm.Firmware) bool {
	if r.Mode == bios.PS2 {
		return f.Size >= 4<<20 && f.Size <= 8<<20
	}
	for _, x := range r.Files {
		if strings.EqualFold(x.Name, f.FileName) {
			return true
		}
	}
	return false
}

// biosAuthWarned: token antigo sem permissão de ler BIOS avisa uma vez só.
var biosAuthWarned sync.Once

// syncBiosLater roda em segundo plano (ao conectar e ao atualizar a lista do
// servidor) e só aparece quando baixa alguma coisa.
func (s *Service) syncBiosLater() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		n, err := s.syncBios(ctx)
		switch {
		case errors.Is(err, romm.ErrAuth):
			biosAuthWarned.Do(func() {
				s.opt.Emitter.Emit("task", TaskEvent{ID: "bios", Label: "BIOS do servidor", State: "error",
					Message: "o app não tem permissão de ler a BIOS do RomM: entre de novo no servidor (Você › Servidor RomM)"})
			})
		case err != nil:
			log.Printf("bios do servidor: %v", err)
		case n > 0:
			s.opt.Emitter.Emit("task", TaskEvent{ID: "bios", Label: "BIOS do servidor", State: "done", Progress: 1,
				Message: fmt.Sprintf("%d arquivo(s) de BIOS baixado(s): os jogos de PS1/PS2 já abrem", n)})
		}
	}()
}

// biosPlan: o que fazer quando falta BIOS na hora de jogar. Tenta trazer do
// RomM; no PS1, ainda dá pra jogar com o PCSX ReARMed, que imita a BIOS
// (core = o core a usar; vazio = o normal).
func (s *Service) biosPlan(system string) (core string, err error) {
	if err = s.biosError(system); err == nil {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	if n, _ := s.syncBios(ctx); n > 0 {
		err = s.biosError(system)
	}
	cancel()
	if err != nil && system == "psx" && s.coreInstalled(psxNoBios) {
		return psxNoBios, nil
	}
	return "", err
}

// psxNoBios: core de PS1 que roda sem a BIOS da Sony (imitação dela).
const psxNoBios = "pcsx_rearmed"

func (s *Service) coreInstalled(core string) bool {
	spec := s.launchSpec()
	goos := spec.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	_, err := os.Stat(launcher.CorePath(launcher.CoresDir(spec), core, goos))
	return err == nil
}
