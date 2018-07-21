// +build windows

package shell

import (
	"os"
	"os/exec"
)

func prepCmd(cmd *exec.Cmd) {
}

func sendSignal(cmd *exec.Cmd, sig os.Signal) error {
	return cmd.Process.Signal(sig)
}
