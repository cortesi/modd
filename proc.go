package modd

import (
	"bufio"
	"os"
	"os/exec"

	"github.com/cortesi/termlog"
)

func getShell() string {
	sh := os.Getenv("SHELL")
	if sh == "" {
		if _, err := os.Stat("/bin/sh"); err == nil {
			sh = "/bin/sh"
		}
	}
	return sh
}

// RunProc runs a process to completion, sending output to log
func RunProc(cmd string, log termlog.Logger) error {
	log.Notice("prep: %s", cmd)
	sh := getShell()
	c := exec.Command(sh, "-c", cmd)
	stdo, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	stde, err := c.StderrPipe()
	if err != nil {
		return err
	}
	go func() {
		r := bufio.NewReader(stde)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				return
			}
			log.Warn(string(line))
		}
	}()
	go func() {
		r := bufio.NewReader(stdo)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				return
			}
			log.Say(string(line))
		}
	}()
	err = c.Start()
	if err != nil {
		return err
	}
	err = c.Wait()
	if err != nil {
		return err
	}
	// FIXME: rusage stats here
	log.NoticeAs("cmdstats", "run time: %s", c.ProcessState.UserTime())
	return nil
}

// RunProcs runs all commands in sequence. Stops if any command returns an error.
func RunProcs(cmds []string, log termlog.Logger) error {
	for _, cmd := range cmds {
		err := RunProc(cmd, log)
		if err != nil {
			return err
		}
	}
	return nil
}
