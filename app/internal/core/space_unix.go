//go:build !windows

package core

import (
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
