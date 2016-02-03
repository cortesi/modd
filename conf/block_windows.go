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
		Command:       command,
		RestartSignal: syscall.SIGHUP,
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
		case "+sigquit":
			d.RestartSignal = syscall.SIGQUIT
		default:
			return fmt.Errorf("unknown option: %s", v)
		}
	}
	b.Daemons = append(b.Daemons, d)
	return nil
}
