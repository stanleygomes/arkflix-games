package fixture

import (
	"crypto/md5"
	"encoding/hex"
	"image/color"
	"net/http"
	"strings"
	"sync"
)

// Conta e jogo do RetroAchievements falso.
const (
	FakeRAUser  = "stanley"
	FakeRAPass  = "senha123"
	FakeRAToken = "token-ra"
	FakeRAGame  = 228 // Super Mario World
)

// FakeRAHash é o hash do Super Mario World do ambiente falso (smw.sfc dentro
// do zip, conteúdo "SNES-SMW", sem cabeçalho de copiadora).
var FakeRAHash = func() string { s := md5.Sum([]byte("SNES-SMW")); return hex.EncodeToString(s[:]) }()

// FakeRA imita o dorequest.php do RetroAchievements (só o que o app usa).
// Prefix é onde ele está montado (pra URL das imagens apontar pra ele mesmo).
type FakeRA struct {
	Prefix string

	mu    sync.Mutex
	calls []string
}

// NewFakeRA monta o servidor falso.
func NewFakeRA() *FakeRA { return &FakeRA{} }

// Calls: valores de "r" recebidos, em ordem.
func (f *FakeRA) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.calls...)
}

func raFail(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, map[string]any{"Success": false, "Status": status, "Code": code, "Error": msg})
}

func (f *FakeRA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/Badge/") {
		c := color.RGBA{R: 240, G: 190, B: 40, A: 255}
		if strings.HasSuffix(r.URL.Path, "_lock.png") {
			c = color.RGBA{R: 90, G: 90, B: 90, A: 255}
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(CoverPNG(c))
		return
	}
	if r.URL.Path != "/dorequest.php" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	q := r.PostForm
	f.mu.Lock()
	f.calls = append(f.calls, q.Get("r"))
	f.mu.Unlock()
	switch q.Get("r") {
	case "login2":
		if q.Get("u") != FakeRAUser || q.Get("p") != FakeRAPass {
			raFail(w, 401, "invalid_credentials", "Invalid user/password combination. Please try again.")
			return
		}
		writeJSON(w, map[string]any{"Success": true, "User": "Stanley", "Token": FakeRAToken, "Score": 1234, "SoftcoreScore": 5, "Messages": 0})
	case "gameid":
		id := 0
		if strings.EqualFold(q.Get("m"), FakeRAHash) {
			id = FakeRAGame
		}
		writeJSON(w, map[string]any{"Success": true, "GameID": id})
	case "achievementsets", "unlocks":
		if !strings.EqualFold(q.Get("u"), FakeRAUser) || q.Get("t") != FakeRAToken {
			raFail(w, 401, "invalid_credentials", "Invalid user/token combination.")
			return
		}
		if q.Get("r") == "unlocks" {
			unlocks := []int{1001}
			if q.Get("h") == "1" {
				unlocks = []int{}
			}
			writeJSON(w, map[string]any{"Success": true, "UserUnlocks": unlocks, "GameID": FakeRAGame, "HardcoreMode": q.Get("h") == "1"})
			return
		}
		writeJSON(w, f.sets("http://"+r.Host+f.Prefix))
	default:
		raFail(w, 422, "invalid_parameter", "Unknown request: "+q.Get("r"))
	}
}

func (f *FakeRA) sets(base string) map[string]any {
	ach := func(id, points, flags int, title, desc, badge string) map[string]any {
		return map[string]any{
			"ID": id, "Title": title, "Description": desc, "Flags": flags, "Points": points,
			"MemAddr": "0xH0001=1", "Author": "fake", "BadgeName": badge, "Created": 1, "Modified": 1,
			"BadgeURL": base + "/Badge/" + badge + ".png", "BadgeLockedURL": base + "/Badge/" + badge + "_lock.png",
		}
	}
	return map[string]any{
		"Success": true, "GameId": FakeRAGame, "Title": "Super Mario World", "ConsoleId": 3,
		"ImageIconUrl": base + "/Images/000001.png", "RichPresenceGameId": FakeRAGame, "RichPresencePatch": "",
		"Sets": []map[string]any{{
			"AchievementSetId": 1, "GameId": FakeRAGame, "Title": nil, "Type": "core",
			"ImageIconUrl": base + "/Images/000001.png", "Leaderboards": []any{},
			"Achievements": []map[string]any{
				ach(1001, 5, 3, "Yoshi!", "Encontre o Yoshi", "00101"),
				ach(1002, 10, 3, "Castelo do Iggy", "Vença o primeiro castelo", "00102"),
				ach(1003, 25, 3, "Estrela", "Termine o jogo", "00103"),
				ach(1099, 1, 5, "Rascunho", "Não promovida", "00199"),
			},
		}},
	}
}
