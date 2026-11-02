//go:build !windows

package core

import (
	"errors"
	"path/filepath"
	"syscall"
)

// diskUsage: tamanho total e livre (pra você) do disco da pasta; -1 se não
// deu pra saber. Sobe até achar uma pasta que existe.
func diskUsage(dir string) (total, free int64) {
	for d := dir; ; d = filepath.Dir(d) {
		var st syscall.Statfs_t
		if syscall.Statfs(d, &st) == nil {
			return int64(st.Blocks) * int64(st.Bsize), int64(st.Bavail) * int64(st.Bsize)
		}
		if d == filepath.Dir(d) {
			return -1, -1
		}
	}
}

func freeSpace(dir string) int64 { _, free := diskUsage(dir); return free }

// sameDisk: as duas pastas (ou a primeira que existe acima delas) estão no
// mesmo sistema de arquivos? Na dúvida, não.
func sameDisk(a, b string) bool {
	da, oka := device(a)
	db, okb := device(b)
	return oka && okb && da == db
}

func device(dir string) (uint64, bool) {
	for d := dir; ; d = filepath.Dir(d) {
		var st syscall.Stat_t
		if syscall.Stat(d, &st) == nil {
			return uint64(st.Dev), true
		}
		if d == filepath.Dir(d) {
			return 0, false
		}
	}
}

// crossDevice: o rename falhou porque origem e destino estão em discos diferentes.
func crossDevice(err error) bool { return errors.Is(err, syscall.EXDEV) }
