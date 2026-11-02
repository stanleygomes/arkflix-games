//go:build !windows

package launcher

import (
	"os"
	"syscall"
)

func terminate(p *os.Process) error { return p.Signal(syscall.SIGTERM) }
