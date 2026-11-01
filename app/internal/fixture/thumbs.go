package fixture

import (
	"fmt"
	"hash/crc32"
	"image/color"
	"net/http"
	"net/url"
	"strings"
)

// FakeThumbs imita o libretro-database (/metadat/...) e o servidor de imagens
// (/thumbs/<console>/Named_Boxarts/<nome>.png) pros jogos do Env.
type FakeThumbs struct {
	Requests []string
}

func crcOf(s string) string { return fmt.Sprintf("%08X", crc32.ChecksumIEEE([]byte(s))) }

// Cada jogo do Env cai num caminho de identificação diferente.
var fakeDATs = map[string]string{
	"no-intro/Nintendo - Super Nintendo Entertainment System.dat": `clrmamepro (
	name "Nintendo - Super Nintendo Entertainment System"
)

game (
	name "Super Mario World (USA)"
	region "USA"
	rom ( name "Super Mario World (USA).sfc" size 8 crc ` + crcOf("SNES-SMW") + ` md5 00 sha1 00 )
)

game (
	name "Super Mario World (Japan)"
	region "Japan"
	rom ( name "Super Mario World (Japan).sfc" size 8 crc 11111111 )
)

game (
	name "Mega Man X (USA) (Beta)"
	rom ( name "Mega Man X (USA) (Beta).sfc" size 3 crc 22222222 )
)

game (
	name "Mega Man X (USA)"
	rom ( name "Mega Man X (USA).sfc" size 3 crc 33333333 )
)
`,
	"no-intro/Sega - Mega Drive - Genesis.dat": `game (
	name "Sonic The Hedgehog 2 (World) (Rev A)"
	rom ( name "Sonic The Hedgehog 2 (World) (Rev A).md" size 2 crc ` + crcOf("MD") + ` )
)
`,
	"no-intro/Nintendo - Nintendo Entertainment System.dat": `game (
	name "Super Mario Bros. 3 (USA)"
	rom ( name "Super Mario Bros. 3 (USA).nes" size 3 crc ` + crcOf("NES") + ` )
)
`,
	"redump/Sony - PlayStation 2.dat": `game (
	name "FIFA Soccer 07 (USA) (En,Es)"
	serial "SLUS-21431"
	rom ( name "FIFA Soccer 07 (USA) (En,Es).iso" size 1 crc 44444444 serial "SLUS-21431" )
)
`,
	"redump/Sony - PlayStation Portable.dat": `game (
	name "Lara Croft Tomb Raider - Legend (USA) (En,Fr,De,Es,It)"
	serial "ULUS-10084"
	rom ( name "Lara Croft Tomb Raider - Legend (USA) (En,Fr,De,Es,It).iso" size 1 crc 55555555 )
)
`,
	"redump/Sony - PlayStation.dat": `game (
	name "Crash Bandicoot (USA)"
	serial "SCUS-94900"
	rom ( name "Crash Bandicoot (USA).cue" size 1 crc 66666666 )
)
`,
	"fbneo-split/FBNeo - Arcade Games.dat": `game (
	name "Ms. Pac-Man"
	releaseyear "1981"
	rom ( name mspacman.zip size 17270 crc 4AC55D24 )
)
`,
	"no-intro/Nintendo - Nintendo 64.dat":    "",
	"no-intro/Nintendo - Game Boy Color.dat": "",
	"no-intro/Nintendo - Nintendo DS.dat":    "",
}

// Imagens que existem (as outras dão 404, como no servidor real).
var fakeImages = map[string]bool{
	"Super Mario World (USA)": true, "Mega Man X (USA)": true, "Sonic The Hedgehog 2 (World) (Rev A)": true,
	"Super Mario Bros. 3 (USA)": true, "FIFA Soccer 07 (USA) (En,Es)": true, "Crash Bandicoot (USA)": true,
	"Lara Croft Tomb Raider - Legend (USA) (En,Fr,De,Es,It)": true, "Ms. Pac-Man": true,
}

func (f *FakeThumbs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, _ := url.PathUnescape(r.URL.EscapedPath())
	f.Requests = append(f.Requests, path)
	if dat, ok := strings.CutPrefix(path, "/metadat/"); ok {
		body, found := fakeDATs[dat]
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
		return
	}
	if rest, ok := strings.CutPrefix(path, "/thumbs/"); ok {
		parts := strings.Split(rest, "/")
		if len(parts) == 3 {
			name := strings.TrimSuffix(parts[2], ".png")
			if fakeImages[name] && !(name == "Ms. Pac-Man" && parts[1] == "Named_Titles") {
				w.Header().Set("Content-Type", "image/png")
				w.Write(CoverPNG(color.RGBA{R: 40, G: 180, B: uint8(len(name) * 5), A: 255}))
				return
			}
		}
	}
	http.NotFound(w, r)
}
