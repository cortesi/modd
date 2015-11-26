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
func RunProc(cmd string, log termlog.Logger) (bool, error) {
	log.Notice("prep: %s", cmd)
	sh := getShell()
	c := exec.Command(sh, "-c", cmd)
	stdo, err := c.StdoutPipe()
	if err != nil {
		return false, err
	}
	stde, err := c.StderrPipe()
	if err != nil {
		return false, err
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
		return false, err
	}
	err = c.Wait()
	if err != nil {
		return false, err
	}
	// FIXME: rusage stats here
	log.NoticeAs("cmdstats", "%s, %s", c.ProcessState.UserTime(), c.ProcessState.String())
	return c.ProcessState.Success(), nil
}

// RunProcs runs all commands in sequence. Stops if any command returns an error.
func RunProcs(cmds []string, log termlog.Logger) (bool, error) {
	for _, cmd := range cmds {
		success, err := RunProc(cmd, log)
		if !success || err != nil {
			return success, err
		}
	}
	return true, nil
}
