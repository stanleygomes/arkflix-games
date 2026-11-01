package core_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestRemoteRomM(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeRomM()
	srv := httptest.NewServer(fake)
	defer srv.Close()

	if _, err := h.svc.ConnectRemote("http://192.168.0.10", "stanley", "senha123"); err == nil {
		t.Fatal("http fora desta máquina expõe o token")
	}
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "errada"); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	info, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123")
	if err != nil || !info.Connected || !info.Mine {
		t.Fatalf("conectar: %+v %v", info, err)
	}
	if ev := h.rec.waitTask(t, "remote"); ev.State != "done" {
		t.Fatalf("catálogo: %+v", ev)
	}
	raw, _ := os.ReadFile(filepath.Join(h.env.Data, "romm.json"))
	if strings.Contains(string(raw), "senha123") || !strings.Contains(string(raw), "token-do-app") {
		t.Fatalf("guarda o token, nunca a senha: %s", raw)
	}
	if st, _ := os.Stat(filepath.Join(h.env.Data, "romm.json")); st.Mode().Perm() != 0o600 {
		t.Fatalf("romm.json legível por outros: %v", st.Mode())
	}
	if s := h.svc.RemoteStatus(); s.Games != 3 {
		t.Fatalf("3 jogos de consoles conhecidos (pico fica de fora): %+v", s)
	}

	dkc := h.game(t, "Donkey Kong Country")
	if !dkc.Remote || dkc.Cover == "" || dkc.Year != 1992 || !strings.Contains(dkc.Summary, "servidor") {
		t.Fatalf("jogo remoto com metadados e capa do servidor: %+v", dkc)
	}
	smw := h.game(t, "Super Mario World")
	if smw.Remote {
		t.Fatal("jogo que já está aqui não duplica")
	}
	if err := h.svc.Play(dkc.ID, ""); err == nil || !strings.Contains(err.Error(), "baixe") {
		t.Fatalf("remoto não abre: %v", err)
	}
	if l, _ := h.svc.Saves(dkc.ID); len(l.States) != 0 {
		t.Fatal("remoto não tem saves locais")
	}

	if err := h.svc.DownloadGame(dkc.ID); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "download:"+dkc.ID); ev.State != "done" {
		t.Fatalf("download: %+v", ev)
	}
	if g := h.game(t, "Donkey Kong Country"); g.Remote || !g.Local {
		t.Fatalf("baixado vira local: %+v", g)
	}
	if _, err := os.Stat(filepath.Join(h.env.Settings.ImportDir, "snes", "Donkey Kong Country (U).zip")); err != nil {
		t.Fatal("arquivo na pasta de importação")
	}

	// save mais novo no servidor desce antes de jogar e sobe de volta depois
	fake.Put("saves", 3, "Super Mario World (U) [!].srm", []byte("do outro pc"), time.Now().Add(time.Minute))
	fake.Put("states", 3, "Super Mario World (U) [!] [2026-10-31 18-18-19].state", []byte("player web"), time.Now().Add(time.Minute))
	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	srm := filepath.Join(h.env.Settings.Saves.RetroArchSaves, "Super Mario World (U) [!].srm")
	if data, _ := os.ReadFile(srm); string(data) != "do outro pc" {
		t.Fatalf("save do servidor devia substituir o mais velho daqui: %s", data)
	}
	if ev := h.rec.waitTask(t, "sync:"+smw.ID); ev.State != "done" || !slices.Contains(fake.Uploads(), "saves:3:Super Mario World (U) [!].srm") {
		t.Fatalf("envio: %+v %v", ev, fake.Uploads())
	}
	if matches, _ := filepath.Glob(filepath.Join(h.env.Settings.Saves.RetroArchStates, "*2026*")); len(matches) != 0 {
		t.Fatal("state do player web (nome com data) não entra")
	}

	// perfil que não ligou o servidor não sincroniza
	bia, _ := h.svc.CreateProfile("Bia", "cat", profiles.Colors[0])
	h.svc.SelectProfile(bia.ID, "")
	if s := h.svc.RemoteStatus(); s.Mine || s.Owner != "Jogador 1" {
		t.Fatalf("dono: %+v", s)
	}
	before := len(fake.Uploads())
	h.rec = &recorder{}
	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if len(fake.Uploads()) != before {
		t.Fatal("saves da Bia não vão pra conta do servidor")
	}
	h.svc.SelectProfile(profiles.Main, "")

	if err := h.svc.DisconnectRemote(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h.env.Data, "romm.json")); !os.IsNotExist(err) {
		t.Fatal("desconectar apaga o token")
	}
	boot, _ := h.svc.Bootstrap()
	for _, g := range boot.Games {
		if g.Remote {
			t.Fatalf("remoto some ao desconectar: %s", g.Name)
		}
	}
}
