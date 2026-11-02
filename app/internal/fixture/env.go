package fixture

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// Clip é um vídeo webm de 2 s (96x64, sem som) pros testes de vídeo.
//
//go:embed testdata/clip.webm
var Clip []byte

// Env é uma instalação falsa completa: pasta pessoal, duas pastas de jogos,
// mídia do ES-DE, saves dos três emuladores e lançadores que só registram a chamada.
type Env struct {
	Home     string
	Local    string // pasta de jogos principal (recebe importação)
	Extra    string // segunda pasta de jogos
	ESDE     string
	Bin      string
	Cores    string
	Data     string
	Inbox    string // arquivos soltos pra testar importação
	Settings settings.Settings
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func put(path string, data []byte) {
	must(os.MkdirAll(filepath.Dir(path), 0o755))
	must(os.WriteFile(path, data, 0o644))
}

// ZipBytes monta um zip em memória.
func ZipBytes(files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		fw, _ := w.Create(name)
		fw.Write([]byte(body))
	}
	w.Close()
	return buf.Bytes()
}

// CoverPNG é uma imagem de uma cor só.
func CoverPNG(c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 60, 80))
	for y := range 80 {
		for x := range 60 {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// P2S monta um save state do PCSX2 (zip) com miniatura.
func P2S() []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("Screenshot.png")
	f.Write(CoverPNG(color.RGBA{R: 30, G: 60, B: 200, A: 255}))
	f, _ = w.Create("eeMemory.bin")
	f.Write([]byte("estado"))
	w.Close()
	return buf.Bytes()
}

// fakeEmulator registra a chamada em calls.log e demora ARKFLIX_FAKE_SLEEP s.
const fakeEmulator = `#!/bin/sh
dir=$(dirname "$0")
printf '%s|%s\n' "$(basename "$0")" "$*" >> "$dir/calls.log"
sleep "${ARKFLIX_FAKE_SLEEP:-0.3}"
exit "${ARKFLIX_FAKE_EXIT:-0}"
`

const gamelist = `<?xml version="1.0"?>
<gameList>
  <game>
    <path>./Super Mario World (U) [!].zip</path>
    <name>Super Mario World</name>
    <desc>Mario e Yoshi atravessam a Dinosaur Land pra salvar a princesa.</desc>
    <releasedate>19901121T000000</releasedate>
    <genre>Plataforma</genre>
    <players>1-2</players>
  </game>
  <game>
    <path>./Mega Man X (U) (V1.1).zip</path>
    <name>Mega Man X</name>
    <desc>X enfrenta os Mavericks de Sigma.</desc>
    <releasedate>19931217T000000</releasedate>
    <genre>Ação, Plataforma</genre>
    <players>1</players>
  </game>
</gameList>
`

// NewEnv cria tudo dentro de root.
func NewEnv(root string) Env {
	e := Env{
		Home:  filepath.Join(root, "home"),
		Bin:   filepath.Join(root, "bin"),
		Cores: filepath.Join(root, "cores"),
		Inbox: filepath.Join(root, "inbox"),
	}
	e.Local = filepath.Join(e.Home, "Games", "roms")
	e.Extra = filepath.Join(e.Home, "Jogos antigos")
	e.ESDE = filepath.Join(e.Home, "ES-DE")
	e.Data = filepath.Join(e.Home, ".config", "arkflix-games")
	e.Settings = settings.Defaults(e.Home)
	e.Settings.LibraryDirs = append(e.Settings.LibraryDirs, settings.LibraryDir{Path: e.Extra, Label: "Jogos antigos"})

	// perfil principal já configurado (a primeira vez tem teste próprio)
	put(filepath.Join(e.Data, "profiles.json"), []byte(`[{"id":"main","name":"Jogador 1","avatar":"mario","color":"#3a86d8","created":1}]`))
	put(filepath.Join(e.Local, "snes", "Super Mario World (U) [!].zip"), ZipBytes(map[string]string{"smw.sfc": "SNES-SMW"}))
	put(filepath.Join(e.Local, "snes", "Mega Man X (U) (V1.1).zip"), ZipBytes(map[string]string{"mmx.sfc": "MMX"}))
	put(filepath.Join(e.Local, "genesis", "Sonic The Hedgehog 2 (World) (Rev A).zip"), ZipBytes(map[string]string{"sonic2.md": "MD"}))
	put(filepath.Join(e.Local, "genesis", "covers", "Sonic The Hedgehog 2 (World) (Rev A).png"), CoverPNG(color.RGBA{R: 30, G: 80, B: 220, A: 255}))
	put(filepath.Join(e.Local, "psx", "Crash Bandicoot (USA)", "Crash Bandicoot (USA).cue"), []byte("FILE \"Crash Bandicoot (USA).bin\" BINARY\n"))
	put(filepath.Join(e.Local, "psx", "Crash Bandicoot (USA)", "Crash Bandicoot (USA).bin"), []byte("PSX"))
	put(filepath.Join(e.Local, "psp", "Lara Croft Tomb Raider - Legend (USA).iso"), PSPISO("ULUS10084", "Tomb Raider: Legend"))
	put(filepath.Join(e.Local, "ps2", "FIFA Soccer 07 (USA).iso"), PS2ISO("SLUS-21431"))
	put(filepath.Join(e.Extra, "nes", "Super Mario Bros. 3.zip"), ZipBytes(map[string]string{"smb3.nes": "NES"}))
	put(filepath.Join(e.Extra, "arcade", "mspacman.zip"), ZipBytes(map[string]string{"pacman.6e": "x"}))
	put(filepath.Join(e.Extra, "n64", "Super Mario 64.zip"), ZipBytes(map[string]string{"sm64.z64": "N64"}))

	put(filepath.Join(e.ESDE, "gamelists", "snes", "gamelist.xml"), []byte(gamelist))
	put(filepath.Join(e.ESDE, "downloaded_media", "snes", "covers", "Super Mario World (U) [!].png"), CoverPNG(color.RGBA{R: 220, G: 40, B: 40, A: 255}))
	put(filepath.Join(e.ESDE, "downloaded_media", "snes", "manuals", "Super Mario World (U) [!].pdf"), []byte("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF\n"))
	put(filepath.Join(e.Local, "snes", "screenshots", "Super Mario World (U) [!].png"), CoverPNG(color.RGBA{R: 60, G: 160, B: 60, A: 255}))
	put(filepath.Join(e.Local, "genesis", "videos", "Sonic The Hedgehog 2 (World) (Rev A).webm"), Clip)

	put(filepath.Join(e.Inbox, "Tetris DX.gbc"), []byte("GBC"))
	put(filepath.Join(e.Inbox, "Metal Slug X (USA).zip"), ZipBytes(map[string]string{
		"Metal Slug X (USA).cue": "FILE \"C:\\roms\\Metal Slug X (USA).bin\" BINARY\n",
		"Metal Slug X (USA).bin": "PSX",
	}))
	put(filepath.Join(e.Inbox, "Crazy Taxi.gdi"), []byte("dreamcast"))

	s := e.Settings.Saves
	old := time.Now().Add(-48 * time.Hour)
	thumb := CoverPNG(color.RGBA{R: 200, G: 60, B: 60, A: 255})
	put(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!].state1"), []byte("s1"))
	put(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!].state1.png"), thumb)
	put(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!].state.auto"), []byte("auto"))
	put(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!].state.auto.png"), thumb)
	os.Chtimes(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!].state1"), old, old)
	put(filepath.Join(s.RetroArchStates, "Super Mario World (U) [!] Hack.state"), []byte("outro jogo"))
	put(filepath.Join(s.RetroArchSaves, "Super Mario World (U) [!].srm"), []byte("bateria"))
	put(filepath.Join(s.PPSSPPStates, "ULUS10084_1.00_0.ppst"), []byte("psp"))
	put(filepath.Join(s.PPSSPPStates, "ULUS10084_1.00_0.jpg"), []byte("jpg"))
	put(filepath.Join(s.PPSSPPSaveData, "ULUS10084DATA00", "DATA.BIN"), []byte("savedata"))
	put(filepath.Join(s.PCSX2States, "SLUS-21431 (1A2B3C4D).01.p2s"), P2S())
	put(filepath.Join(s.PCSX2Memcards, "Mcd001.ps2"), make([]byte, 1024))
	// BIOS falsas (tamanho certo, conteúdo não): PS1 e PS2 jogam; os testes
	// apagam pra ver o aviso de que falta
	put(PS1BiosPath(e.Settings), make([]byte, 512<<10))
	put(PS2BiosPath(e.Settings), PS2Bios())

	must(os.MkdirAll(e.Bin, 0o755))
	for _, name := range []string{"retroarch", "ppsspp", "pcsx2"} {
		must(os.WriteFile(filepath.Join(e.Bin, name), []byte(fakeEmulator), 0o755))
	}
	for _, sys := range catalog.Systems {
		if sys.Core != "" {
			put(filepath.Join(e.Cores, sys.Core+"_libretro.so"), nil)
		}
	}
	// um core alternativo (bsnes) e um shader de CRT pra configuração por jogo
	put(filepath.Join(e.Cores, "bsnes_libretro.so"), nil)
	put(filepath.Join(filepath.Dir(s.RetroArchSaves), "shaders", "shaders_slang", "crt", "crt-easymode.slangp"), []byte("shaders = 1\n"))
	// script de fim de jogo (como o backup do solomon): registra os argumentos
	hook := filepath.Join(e.Data, "scripts", "game-end", "10-registra.sh")
	log := filepath.Join(root, "hook.log")
	put(hook, []byte("#!/bin/sh\nprintf '%s|' \"$@\" >> \""+log+"\"\necho >> \""+log+"\"\n"))
	must(os.Chmod(hook, 0o755))
	must(os.MkdirAll(e.Data, 0o700))
	return e
}

// PS1BiosPath: a BIOS americana do PS1 na pasta system do RetroArch.
func PS1BiosPath(s settings.Settings) string {
	return filepath.Join(filepath.Dir(s.Saves.RetroArchSaves), "system", "scph5501.bin")
}

// PS2BiosPath: a BIOS do PS2 na pasta bios do PCSX2.
func PS2BiosPath(s settings.Settings) string {
	return filepath.Join(filepath.Dir(s.Saves.PCSX2Memcards), "bios", "SCPH-39001.bin")
}

// PS2Bios é uma BIOS de PS2 de mentira: 4 MB com a entrada RESET do ROMDIR.
func PS2Bios() []byte {
	b := make([]byte, 4<<20)
	copy(b[0x2700:], "RESET")
	copy(b[0x2710:], "ROMDIR")
	return b
}

// Calls lê o registro dos lançadores falsos.
func (e Env) Calls() string {
	data, _ := os.ReadFile(filepath.Join(e.Bin, "calls.log"))
	return string(data)
}
