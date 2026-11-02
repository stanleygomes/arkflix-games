package core_test

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestCheevos(t *testing.T) {
	fake := fixture.NewFakeRA()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	t.Setenv("ARKFLIX_RA_BASE", srv.URL)
	h := setup(t)
	smw := h.game(t, "Super Mario World")

	if st := h.svc.CheevosStatus(); st.Connected {
		t.Fatalf("começa sem conta: %+v", st)
	}
	if g, err := h.svc.GameCheevos(smw.ID); err != nil || g.Connected || g.Reason == "" {
		t.Fatalf("sem conta: %+v %v", g, err)
	}
	if _, err := h.svc.CheevosLogin("stanley", "errada"); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	st, err := h.svc.CheevosLogin(" stanley ", "senha123")
	if err != nil || !st.Connected || st.User != "Stanley" || st.Hardcore {
		t.Fatalf("login: %+v %v", st, err)
	}
	for _, name := range []string{"cheevos.json", "retroarch-cheevos.cfg"} {
		p := filepath.Join(h.env.Data, name)
		raw, _ := os.ReadFile(p)
		if strings.Contains(string(raw), "senha123") || !strings.Contains(string(raw), fixture.FakeRAToken) {
			t.Fatalf("%s guarda o token, nunca a senha: %s", name, raw)
		}
		if info, err := os.Stat(p); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s legível por outros: %v", name, err)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(h.env.Data, "retroarch-cheevos.cfg"))
	for _, want := range []string{`cheevos_enable = "true"`, `cheevos_username = "Stanley"`, `cheevos_hardcore_mode_enable = "false"`} {
		if !strings.Contains(string(cfg), want) {
			t.Fatalf("config do RetroArch sem %s:\n%s", want, cfg)
		}
	}

	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	if !strings.Contains(h.env.Calls(), "retroarch-cheevos.cfg") {
		t.Fatalf("RetroArch devia receber a conta: %s", h.env.Calls())
	}

	g, err := h.svc.GameCheevos(smw.ID)
	if err != nil || !g.Supported || g.RAGameID != fixture.FakeRAGame || g.Title != "Super Mario World" {
		t.Fatalf("conquistas: %+v %v", g, err)
	}
	if g.Total != 3 || g.Unlocked != 1 || g.Points != 40 || g.PointsUnlocked != 5 {
		t.Fatalf("placar: %+v", g)
	}
	if !g.List[0].Unlocked || !strings.HasSuffix(g.List[0].BadgeURL, "/Badge/00101.png") ||
		g.List[1].Unlocked || !strings.HasSuffix(g.List[1].BadgeURL, "_lock.png") {
		t.Fatalf("lista: %+v", g.List)
	}
	calls := len(fake.Calls())
	if _, err := h.svc.GameCheevos(smw.ID); err != nil || len(fake.Calls()) != calls {
		t.Fatalf("segunda consulta vem do cache: %v %d → %d", err, calls, len(fake.Calls()))
	}
	if _, err := os.Stat(filepath.Join(h.env.Data, "cache", "cheevos", "hashes.json")); err != nil {
		t.Fatal("hash da ROM fica em cache no disco")
	}
	for _, r := range fake.Calls() {
		if r != "login2" && r != "gameid" && r != "achievementsets" && r != "unlocks" {
			t.Fatalf("chamada que mexe no perfil do RA: %s", r)
		}
	}

	if g, err := h.svc.GameCheevos(h.game(t, "Mega Man X").ID); err != nil || g.Supported || !strings.Contains(g.Reason, "não reconhece") {
		t.Fatalf("jogo que o RA não conhece: %+v %v", g, err)
	}
	if g, err := h.svc.GameCheevos(h.game(t, "Crash Bandicoot").ID); err != nil || g.Supported || g.Reason == "" {
		t.Fatalf("disco ilegível: %+v %v", g, err)
	}

	// cada perfil tem a sua conta
	bia, _ := h.svc.CreateProfile("Bia", "cat", profiles.Colors[0])
	h.svc.SelectProfile(bia.ID, "")
	if st := h.svc.CheevosStatus(); st.Connected {
		t.Fatal("conta é do perfil, não do PC")
	}
	if g, _ := h.svc.GameCheevos(smw.ID); g.Connected {
		t.Fatal("a Bia não vê as conquistas do Stanley")
	}

	// infantil não entra nem sai
	kid, _ := h.svc.CreateProfile("Nina", "mushroom", profiles.Colors[6])
	h.svc.SelectProfile(profiles.Main, "")
	h.svc.SetProfileKids(kid.ID, true)
	h.svc.SelectProfile(kid.ID, "")
	if _, err := h.svc.CheevosLogin("stanley", "senha123"); !errors.Is(err, core.ErrKids) {
		t.Fatalf("infantil não loga: %v", err)
	}
	if err := h.svc.CheevosLogout(); !errors.Is(err, core.ErrKids) {
		t.Fatalf("infantil não desloga: %v", err)
	}

	h.svc.SelectProfile(profiles.Main, "")
	if err := h.svc.CheevosLogout(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cheevos.json", "retroarch-cheevos.cfg"} {
		if _, err := os.Stat(filepath.Join(h.env.Data, name)); !os.IsNotExist(err) {
			t.Fatalf("sair apaga %s", name)
		}
	}
}
