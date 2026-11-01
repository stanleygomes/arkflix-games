package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestProfilesIsolation(t *testing.T) {
	h := setup(t)
	smw := h.game(t, "Super Mario World")
	if _, err := h.svc.SetFavorite(smw.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.CreateList("Do principal"); err != nil {
		t.Fatal(err)
	}
	st, _ := h.svc.Profiles()
	if len(st.Profiles) != 1 || st.Current != profiles.Main || st.Choose {
		t.Fatalf("migração: um perfil, sem tela de escolha: %+v", st)
	}

	bia, err := h.svc.CreateProfile("Bia", "cat", profiles.Colors[2])
	if err != nil {
		t.Fatal(err)
	}
	if st, _ := h.svc.Profiles(); !st.Choose {
		t.Fatal("com dois perfis, a escolha aparece ao abrir")
	}
	if _, err := h.svc.SetProfilePIN(bia.ID, "", "4321"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SelectProfile(bia.ID, "1111"); err != profiles.ErrPIN {
		t.Fatalf("PIN errado: %v", err)
	}
	boot, err := h.svc.SelectProfile(bia.ID, "4321")
	if err != nil {
		t.Fatal(err)
	}
	if boot.Profile.Current != bia.ID || boot.Profile.Choose || len(boot.Lists) != 0 {
		t.Fatalf("perfil novo começa limpo: %+v", boot.Profile)
	}
	for _, g := range boot.Games {
		shared := g.System == "psp" || g.System == "ps2" // emuladores sem pasta por perfil
		if g.Favorite || (g.HasSaves && !shared) || g.Plays != 0 {
			t.Fatalf("perfil novo não vê favoritos nem saves do principal: %+v", g)
		}
	}
	if l, _ := h.svc.Saves(smw.ID); len(l.States) != 0 {
		t.Fatalf("saves do principal vazaram: %+v", l.States)
	}

	// acessibilidade é do perfil; o resto das configurações é da casa
	s := h.svc.GetSettings()
	s.A11y.FontScale, s.Fullscreen = 1.5, false
	if _, err := h.svc.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	h.svc.SetFavorite(h.game(t, "Mega Man X").ID, true)
	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	calls := h.env.Calls()
	if !strings.Contains(calls, "retroarch-saves.cfg") {
		t.Fatalf("RetroArch do perfil usa pastas de save próprias: %s", calls)
	}
	cfg, _ := os.ReadFile(filepath.Join(h.env.Data, "profiles", bia.ID, "retroarch-saves.cfg"))
	if !strings.Contains(string(cfg), filepath.Join("profiles", bia.ID, "saves", "retroarch", "states")) {
		t.Fatalf("cfg de saves: %s", cfg)
	}
	if err := h.svc.DeleteProfile(bia.ID, "4321"); err == nil {
		t.Fatal("não apaga o perfil em uso")
	}

	if _, err := h.svc.SelectProfile(profiles.Main, ""); err != nil {
		t.Fatal(err)
	}
	if g := h.game(t, "Super Mario World"); !g.Favorite || !g.HasSaves || g.Plays != 0 {
		t.Fatalf("principal mantém o dele e não ganha a partida da Bia: %+v", g)
	}
	if h.game(t, "Mega Man X").Favorite {
		t.Fatal("favorito da Bia vazou pro principal")
	}
	main := h.svc.GetSettings()
	if main.A11y.FontScale != 1 || main.Fullscreen {
		t.Fatalf("fonte é por perfil, tela cheia é da casa: %+v", main)
	}
	if lists, _ := h.svc.Lists(); len(lists) != 1 {
		t.Fatalf("listas por perfil: %+v", lists)
	}
	if err := h.svc.DeleteProfile(bia.ID, "0000"); err != profiles.ErrPIN {
		t.Fatalf("apagar pede o PIN: %v", err)
	}
	if err := h.svc.DeleteProfile(bia.ID, "4321"); err != nil {
		t.Fatal(err)
	}
	if st, _ := h.svc.Profiles(); len(st.Profiles) != 1 {
		t.Fatalf("depois de apagar: %+v", st)
	}
}

func TestKidsProfile(t *testing.T) {
	h := setup(t)
	smw := h.game(t, "Super Mario World")
	mmx := h.game(t, "Mega Man X")
	if _, err := h.svc.SaveMeta(smw.ID, meta.Override{KidsOK: true}); err != nil {
		t.Fatal(err)
	}
	kid, _ := h.svc.CreateProfile("Nina", "mushroom", profiles.Colors[6])
	if _, err := h.svc.SetProfileKids(profiles.Main, true); err == nil {
		t.Fatal("principal não vira infantil")
	}
	if v, err := h.svc.SetProfileKids(kid.ID, true); err != nil || !v.Kids {
		t.Fatalf("marcar infantil: %+v %v", v, err)
	}
	if _, err := h.svc.SetProfilePIN(profiles.Main, "", "9090"); err != nil {
		t.Fatal(err)
	}
	boot, err := h.svc.SelectProfile(kid.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(boot.Games) != 1 || boot.Games[0].ID != smw.ID {
		t.Fatalf("só o jogo liberado aparece: %d jogos", len(boot.Games))
	}
	if err := h.svc.Play(mmx.ID, ""); err == nil {
		t.Fatal("jogo não liberado não abre nem pela API")
	}
	if _, err := h.svc.Saves(mmx.ID); err == nil {
		t.Fatal("saves de jogo não liberado")
	}
	blocked := map[string]error{}
	_, blocked["SaveSettings"] = h.svc.SaveSettings(h.svc.GetSettings())
	_, blocked["ResetSettings"] = h.svc.ResetSettings()
	_, blocked["SaveInput"] = h.svc.ResetInput()
	_, blocked["Import"] = h.svc.Import(nil)
	_, blocked["PickFiles"] = h.svc.PickFiles()
	_, blocked["SaveMeta"] = h.svc.SaveMeta(smw.ID, meta.Override{})
	_, blocked["DeleteState"] = h.svc.DeleteState(smw.ID, "x")
	_, blocked["CreateProfile"] = h.svc.CreateProfile("Hack", "ghost", profiles.Colors[0])
	_, blocked["SetProfilePIN"] = h.svc.SetProfilePIN(profiles.Main, "9090", "")
	_, blocked["SetProfileKids"] = h.svc.SetProfileKids(kid.ID, false)
	blocked["DeleteProfile"] = h.svc.DeleteProfile(kid.ID, "")
	blocked["UpdateCovers"] = h.svc.UpdateCovers(false)
	for name, err := range blocked {
		if err != core.ErrKids {
			t.Errorf("%s devia ser bloqueado no perfil infantil: %v", name, err)
		}
	}
	if len(h.svc.AnalyzeImport([]string{h.env.Inbox})) != 0 {
		t.Error("análise de importação no perfil infantil")
	}
	if _, err := h.svc.SetFavorite(smw.ID, true); err != nil {
		t.Fatalf("favoritar pode: %v", err)
	}
	if _, err := h.svc.SelectProfile(profiles.Main, ""); err != profiles.ErrPIN {
		t.Fatalf("sair do infantil pede o PIN do adulto: %v", err)
	}
	if _, err := h.svc.SelectProfile(profiles.Main, "9090"); err != nil {
		t.Fatal(err)
	}
	if v, _ := h.svc.ResetMeta(smw.ID); !v.KidsOK {
		t.Fatal("restaurar metadados não tira a liberação")
	}
}
