// +build !windows

package shell

import (
	"os"
	"os/exec"
	"syscall"
)

func prepCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (e *Executor) sendSignal(sig os.Signal) error {
	return syscall.Kill(-e.cmd.Process.Pid, sig.(syscall.Signal))
}
