// +build windows

package shell

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func prepCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func (e *Executor) sendSignal(sig os.Signal) error {
	return exec.Command("taskkill", "/f", "/t", "/pid", strconv.Itoa(e.cmd.Process.Pid)).Run()
}
