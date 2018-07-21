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

func sendSignal(cmd *exec.Cmd, sig os.Signal) error {
	return syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal))
}
