//go:build !windows

package core

import (
	"path/filepath"
	"syscall"
)

// freeSpace: bytes livres no disco da pasta (-1 se não deu pra saber).
func freeSpace(dir string) int64 {
	for d := dir; ; d = filepath.Dir(d) {
		var st syscall.Statfs_t
		if syscall.Statfs(d, &st) == nil {
			return int64(st.Bavail) * int64(st.Bsize)
		}
		if d == filepath.Dir(d) {
			return -1
		}
	}
}
