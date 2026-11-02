package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Última sincronização de saves com o servidor, por jogo (sync.json do
// perfil). É o que separa "o servidor está mais novo" de "conflito".
var syncMu sync.Mutex

func (u *user) syncFile() string { return filepath.Join(u.dir, "sync.json") }

func (u *user) syncLog() map[string]int64 {
	out := map[string]int64{}
	if b, err := os.ReadFile(u.syncFile()); err == nil {
		json.Unmarshal(b, &out)
	}
	return out
}

func (u *user) lastSync(game string) int64 {
	syncMu.Lock()
	defer syncMu.Unlock()
	return u.syncLog()[game]
}

func (u *user) setSync(game string, at int64) {
	syncMu.Lock()
	defer syncMu.Unlock()
	log := u.syncLog()
	log[game] = at
	if b, err := json.Marshal(log); err == nil {
		os.WriteFile(u.syncFile(), b, 0o644)
	}
}
