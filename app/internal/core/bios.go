package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/bios"
	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
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
	Dir      string      `json:"dir"`
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
	return BiosInfo{System: r.System, Name: name, Required: r.Required, OK: res.OK, Found: nonNil(res.Found),
		Missing: nonNil(res.Missing), Warnings: nonNil(res.Warnings), Dir: dir}
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
	return fmt.Errorf("Falta a BIOS do %s (%s): coloque em %s", info.Name, r.Hint, info.Dir)
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
