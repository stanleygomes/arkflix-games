package romm_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/romm"
)

func TestNormalizeBase(t *testing.T) {
	ok := map[string]string{
		"romm.exemplo.org":                 "https://romm.exemplo.org",
		"https://romm.casa.lan/":           "https://romm.casa.lan",
		"http://127.0.0.1:8080/fake-romm/": "http://127.0.0.1:8080/fake-romm",
		"http://localhost:3000":            "http://localhost:3000",
	}
	for in, want := range ok {
		if got, err := romm.NormalizeBase(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "http://romm.casa.lan", "ftp://x", "https://"} {
		if _, err := romm.NormalizeBase(bad); err == nil {
			t.Errorf("devia recusar %q", bad)
		}
	}
}

func TestLoginCatalogDownloadAndSaves(t *testing.T) {
	fake := fixture.NewFakeRomM()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()

	if _, err := romm.Login(ctx, srv.URL, "stanley", "errada", "teste"); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	acc, err := romm.Login(ctx, srv.URL, "stanley", "senha123", "arkflix-games (pc)")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Token != "token-do-app" || acc.TokenID != 7 || fake.Tokens()[0] != "arkflix-games (pc)" {
		t.Fatalf("token de dispositivo: %+v %v", acc, fake.Tokens())
	}
	c := &romm.Client{Base: acc.Base, Token: acc.Token}
	if me, err := c.Me(ctx); err != nil || me != "stanley" {
		t.Fatalf("me: %q %v", me, err)
	}
	if _, err := (&romm.Client{Base: acc.Base, Token: "velho"}).Me(ctx); err != romm.ErrAuth {
		t.Fatalf("token revogado: %v", err)
	}

	roms, err := c.Roms(ctx)
	if err != nil || len(roms) != 4 {
		t.Fatalf("catálogo: %d %v", len(roms), err)
	}
	dkc := roms[0]
	if dkc.Year != 1992 || dkc.Players != "1-2" || dkc.Genres[0] != "Plataforma" || !strings.HasPrefix(dkc.Cover, "/assets/") {
		t.Fatalf("metadados: %+v", dkc)
	}

	dir := t.TempDir()
	var last int64
	path, err := c.Download(ctx, dkc, filepath.Join(dir, "snes"), func(done, total int64) { last = done })
	if err != nil || filepath.Base(path) != "Donkey Kong Country (U).zip" || last != dkc.Size {
		t.Fatalf("download simples: %s %v (%d de %d)", path, err, last, dkc.Size)
	}
	if _, err := c.Download(ctx, dkc, filepath.Join(dir, "snes"), nil); err == nil {
		t.Fatal("não sobrescreve jogo que já existe")
	}
	spyro, err := c.Rom(ctx, 2)
	if err != nil || len(spyro.Files) != 2 {
		t.Fatalf("detalhe: %+v %v", spyro, err)
	}
	path, err = c.Download(ctx, spyro, filepath.Join(dir, "psx"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(path, "Spyro the Dragon (USA).bin")); string(data) != "PSX-DISCO" {
		t.Fatalf("disco em pasta: %s", data)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, "*", "*.part")); len(leftovers) != 0 {
		t.Fatalf("sobrou .part: %v", leftovers)
	}

	bad := spyro
	bad.Files = []romm.File{{ID: 21, Name: "x.bin", Path: "roms/psx/Spyro the Dragon (USA)/../../../fora"}}
	bad.FsName = "Outro"
	if _, err := c.Download(ctx, bad, filepath.Join(dir, "psx"), nil); err == nil {
		t.Fatal("caminho do servidor não pode sair da pasta do jogo")
	}

	save := filepath.Join(dir, "Super Mario World (U) [!].srm")
	os.WriteFile(save, []byte("bateria"), 0o644)
	for range 2 {
		if err := c.Upload(ctx, romm.Saves, 3, "snes9x", save, ""); err != nil {
			t.Fatal(err)
		}
	}
	list, err := c.Assets(ctx, romm.Saves, 3)
	if err != nil || len(list) != 1 || list[0].FileName != "Super Mario World (U) [!].srm" {
		t.Fatalf("reenviar substitui pelo nome: %+v %v", list, err)
	}
	back := filepath.Join(dir, "baixado", "smw.srm")
	if err := c.Fetch(ctx, romm.Saves, list[0], back); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(back)
	if data, _ := os.ReadFile(back); string(data) != "bateria" || !info.ModTime().Equal(list[0].Updated.Truncate(time.Second)) && info.ModTime().Sub(list[0].Updated).Abs() > time.Second {
		t.Fatalf("baixar save: %s %v", data, info.ModTime())
	}
	cover := filepath.Join(dir, "capa.png")
	if err := c.Cover(ctx, dkc.Cover, cover); err != nil {
		t.Fatal(err)
	}
	if err := c.Cover(ctx, "https://outro.site/x.png", cover); err == nil {
		t.Fatal("capa só do próprio servidor")
	}
}

func TestDownloadResumeMultiAndLimit(t *testing.T) {
	fake := fixture.NewFakeRomM()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()
	c := &romm.Client{Base: srv.URL, Token: "token-do-app"}
	spyro, err := c.Rom(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	// pausado no meio do .bin: o .cue completo é pulado e o .bin continua com Range
	dir := filepath.Join(t.TempDir(), "psx")
	part := filepath.Join(dir, "Spyro the Dragon (USA).part")
	os.MkdirAll(part, 0o755)
	os.WriteFile(filepath.Join(part, "Spyro the Dragon (USA).cue"), []byte("FILE \"Spyro the Dragon (USA).bin\" BINARY\n"), 0o644)
	os.WriteFile(filepath.Join(part, "Spyro the Dragon (USA).bin"), []byte("PSX-"), 0o644)
	var last int64
	path, err := c.Download(ctx, spyro, dir, func(done, total int64) { last = done })
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(path, "Spyro the Dragon (USA).bin")); string(data) != "PSX-DISCO" {
		t.Fatalf("retomada: %q", data)
	}
	if r := fake.Ranges(); len(r) != 1 || r[0] != "bytes=4-" {
		t.Fatalf("só o .bin pede o resto: %v", r)
	}
	if last != spyro.Size {
		t.Fatalf("progresso conta o que já tinha: %d de %d", last, spyro.Size)
	}

	// limite de velocidade: 96 KB a 64 KB/s levam ~1,5 s
	big := make([]byte, 96<<10)
	for i := range big {
		big[i] = byte(i * 7 % 251)
	}
	fake.SetRom(1, big)
	dkc, _ := c.Rom(ctx, 1)
	c.Limit = func() int64 { return 64 << 10 }
	start := time.Now()
	if _, err := c.Download(ctx, dkc, filepath.Join(t.TempDir(), "snes"), nil); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took < 1200*time.Millisecond || took > 4*time.Second {
		t.Fatalf("96 KB a 64 KB/s em %v", took)
	}
	c.Limit = func() int64 { return 0 }
	start = time.Now()
	if _, err := c.Download(ctx, dkc, filepath.Join(t.TempDir(), "snes"), nil); err != nil || time.Since(start) > time.Second {
		t.Fatalf("sem limite: %v %v", time.Since(start), err)
	}
}
