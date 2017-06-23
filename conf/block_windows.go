// +build  windows

package conf

import (
	"fmt"
	"syscall"
)

func (b *Block) addDaemon(command string, options []string) error {
	if b.Daemons == nil {
		b.Daemons = []Daemon{}
	}
	d := Daemon{
		Command:           command,
		RestartSignal:     syscall.SIGHUP,
		PipeRestartSignal: true,
	}
	for _, v := range options {
		switch v {
		case "+sighup":
			d.RestartSignal = syscall.SIGHUP
		case "+sigterm":
			d.RestartSignal = syscall.SIGTERM
		case "+sigint":
			d.RestartSignal = syscall.SIGINT
		case "+sigkill":
			d.RestartSignal = syscall.SIGKILL
			// Although Windows doesn't have signals, Go does recognise the
			// intention of SIGKILL and uses a native API to terminate the
			// target process.
			d.PipeRestartSignal = false
		case "+sigquit":
			d.RestartSignal = syscall.SIGQUIT
		default:
			return fmt.Errorf("unknown option: %s", v)
		}
	}
	b.Daemons = append(b.Daemons, d)
	return nil
}
