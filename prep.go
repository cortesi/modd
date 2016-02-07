package modd

import (
	"bytes"
	"os/exec"
	"sync"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/modd/watch"
	"github.com/cortesi/termlog"
)

// ProcError is a process error, possibly containing command output
type ProcError struct {
	shorttext string
	Output    string
}

func (p ProcError) Error() string {
	return p.shorttext
}

// RunProc runs a process to completion, sending output to log
func RunProc(cmd string, log termlog.Stream) error {
	log.Header()
	sh, error := getShell()
	if error != nil {
		return error
	}
	c := exec.Command(sh, "-c", cmd)
	stdo, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	stde, err := c.StderrPipe()
	if err != nil {
		return err
	}
	buff := new(bytes.Buffer)
	mut := sync.Mutex{}
	err = c.Start()
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go logOutput(
		&wg, stde,
		func(s string, args ...interface{}) {
			log.Warn(s)

			mut.Lock()
			defer mut.Unlock()
			buff.WriteString(s + "\n")
		},
	)
	go logOutput(&wg, stdo, log.Say)
	wg.Wait()
	err = c.Wait()
	if err != nil {
		log.Shout("%s", c.ProcessState.String())
		return ProcError{err.Error(), buff.String()}
	}
	log.Notice(">> done (%s)", c.ProcessState.UserTime())
	return nil
}

// RunPreps runs all commands in sequence. Stops if any command returns an error.
func RunPreps(b conf.Block, vars map[string]string, mod *watch.Mod, log termlog.TermLog, notifiers []notify.Notifier) error {
	vcmd := varcmd.VarCmd{Block: &b, Mod: mod, Vars: vars}
	for _, p := range b.Preps {
		cmd, err := vcmd.Render(p.Command)
		if err != nil {
			return err
		}
		err = RunProc(cmd, log.Stream(niceHeader("prep: ", cmd)))
		if err != nil {
			if pe, ok := err.(ProcError); ok {
				for _, n := range notifiers {
					n.Push("modd error", pe.Output, "")
				}
			} else {
				return err
			}
		}
	}
	return nil
}
