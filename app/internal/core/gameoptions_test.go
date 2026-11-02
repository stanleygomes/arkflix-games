package core_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestHidden(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Mega Man X")
	v, err := h.svc.SetHidden(g.ID, true)
	if err != nil || !v.Hidden {
		t.Fatalf("ocultar: %+v %v", v, err)
	}
	if !h.game(t, "Mega Man X").Hidden {
		t.Fatal("oculto não persistiu")
	}
	// por perfil: outro perfil vê o jogo normal
	other, _ := h.svc.CreateProfile("Bia", "mushroom", profiles.Colors[2])
	if _, err := h.svc.SelectProfile(other.ID, ""); err != nil {
		t.Fatal(err)
	}
	if h.game(t, "Mega Man X").Hidden {
		t.Fatal("oculto vazou pra outro perfil")
	}
	if _, err := h.svc.SelectProfile(profiles.Main, ""); err != nil {
		t.Fatal(err)
	}
	if v, err := h.svc.SetHidden(g.ID, false); err != nil || v.Hidden {
		t.Fatalf("mostrar de novo: %+v %v", v, err)
	}
}

func TestDeleteGame(t *testing.T) {
	h := setup(t)
	crash := h.game(t, "Crash Bandicoot")
	info, err := h.svc.GameSize(crash.ID)
	if err != nil || info.Size != crash.Size || info.Files != 1 || info.Server {
		t.Fatalf("tamanho: %+v %v (jogo %d)", info, err, crash.Size)
	}
	if err := h.svc.DeleteGame(crash.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h.env.Local, "psx", "Crash Bandicoot (USA)")); !os.IsNotExist(err) {
		t.Fatal("pasta do jogo continua lá")
	}
	if _, err := os.Stat(filepath.Join(h.env.Local, "psx")); err != nil {
		t.Fatal("a pasta do console não pode sumir")
	}
	b, _ := h.svc.Bootstrap()
	for _, g := range b.Games {
		if g.ID == crash.ID {
			t.Fatal("jogo apagado continua na biblioteca")
		}
	}

	// saves ficam
	smw := h.game(t, "Super Mario World")
	if err := h.svc.DeleteGame(smw.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h.env.Settings.Saves.RetroArchSaves, "Super Mario World (U) [!].srm")); err != nil {
		t.Fatal("o save foi junto com o jogo")
	}
	if _, err := os.Stat(filepath.Join(h.env.Local, "snes", "Mega Man X (U) (V1.1).zip")); err != nil {
		t.Fatal("apagou o jogo errado")
	}

	// pasta só de leitura: recusa
	smb := h.game(t, "Super Mario Bros. 3")
	if _, err := h.svc.GameSize(smb.ID); err == nil {
		t.Fatal("jogo de pasta só de leitura não tem o que apagar")
	}
	if err := h.svc.DeleteGame(smb.ID); err == nil {
		t.Fatal("apagou de pasta só de leitura")
	}
	if _, err := os.Stat(filepath.Join(h.env.Extra, "nes", "Super Mario Bros. 3.zip")); err != nil {
		t.Fatal("arquivo de pasta só de leitura sumiu")
	}
}

func TestDeleteLooseCueAndM3U(t *testing.T) {
	h := setup(t)
	dir := filepath.Join(h.env.Local, "psx")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Spyro (USA).cue", "FILE \"Spyro (USA) (Track 1).bin\" BINARY\nFILE \"Spyro (USA) (Track 2).bin\" BINARY\n")
	write("Spyro (USA) (Track 1).bin", "1234")
	write("Spyro (USA) (Track 2).bin", "56")
	write("FF7 (USA).m3u", "FF7 (USA) (Disc 1).cue\n# comentário\n../../fora.cue\n")
	write("FF7 (USA) (Disc 1).cue", "FILE \"FF7 (USA) (Disc 1).bin\" BINARY\n")
	write("FF7 (USA) (Disc 1).bin", "disco")
	fora := filepath.Join(h.env.Home, "fora.cue")
	if err := os.WriteFile(fora, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.svc.Refresh()

	spyro := h.game(t, "Spyro")
	info, err := h.svc.GameSize(spyro.ID)
	if err != nil || info.Files != 3 {
		t.Fatalf("cue + 2 trilhas: %+v %v", info, err)
	}
	if err := h.svc.DeleteGame(spyro.ID); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"Spyro (USA).cue", "Spyro (USA) (Track 1).bin", "Spyro (USA) (Track 2).bin"} {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Fatalf("%s continua lá", n)
		}
	}

	var ff7 core.GameView
	b, _ := h.svc.Bootstrap()
	for _, g := range b.Games {
		if g.FileName == "FF7 (USA).m3u" {
			ff7 = g
		}
	}
	if ff7.ID == "" {
		t.Fatal("m3u não entrou na biblioteca")
	}
	if err := h.svc.DeleteGame(ff7.ID); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"FF7 (USA).m3u", "FF7 (USA) (Disc 1).cue", "FF7 (USA) (Disc 1).bin"} {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Fatalf("%s continua lá", n)
		}
	}
	if _, err := os.Stat(fora); err != nil {
		t.Fatal("apagou arquivo fora da pasta do console")
	}
	if _, err := os.Stat(filepath.Join(dir, "Crash Bandicoot (USA)")); err != nil {
		t.Fatal("apagou outro jogo")
	}
}

func TestDeleteRunningGame(t *testing.T) {
	t.Setenv("ARKFLIX_FAKE_SLEEP", "1")
	h := setup(t)
	g := h.game(t, "Mega Man X")
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DeleteGame(g.ID); err == nil {
		t.Fatal("apagou jogo aberto")
	}
	h.rec.waitGame(t, "exited")
}

func TestDeleteGameBackToServer(t *testing.T) {
	h := setup(t)
	srv := httptest.NewServer(fixture.NewFakeRomM())
	defer srv.Close()
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	h.rec.waitTask(t, "remote")
	smw := h.game(t, "Super Mario World")
	if info, err := h.svc.GameSize(smw.ID); err != nil || !info.Server {
		t.Fatalf("jogo do servidor: %+v %v", info, err)
	}
	if err := h.svc.DeleteGame(smw.ID); err != nil {
		t.Fatal(err)
	}
	if g := h.game(t, "Super Mario World"); !g.Remote || g.ID != smw.ID {
		t.Fatalf("volta como remoto: %+v", g)
	}
}
