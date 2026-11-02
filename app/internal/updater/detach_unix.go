//go:build !windows

package updater

import (
	"os/exec"
	"syscall"
)

// detach: sessão própria, pra não morrer junto com o app.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
