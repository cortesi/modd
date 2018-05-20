package modd

import (
	"fmt"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/shell"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/moddwatch"
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
func RunProc(cmd, shellMethod string, log termlog.Stream) error {
	log.Header()
	ex := shell.GetExecutor(shellMethod)
	if ex == nil {
		return fmt.Errorf("Could not find executor %s", shellMethod)
	}
	start := time.Now()
	err, procerr, errbuf := ex.Run(cmd, log, true)
	if err != nil {
		return err
	} else if procerr != nil {
		log.Shout("%s", procerr)
		return ProcError{err.Error(), errbuf}
	}
	log.Notice(">> done (%s)", time.Since(start))
	return nil
}

// RunPreps runs all commands in sequence. Stops if any command returns an error.
func RunPreps(
	b conf.Block,
	vars map[string]string,
	mod *moddwatch.Mod,
	log termlog.TermLog,
	notifiers []notify.Notifier,
	initial bool,
) error {
	shell := vars[shellVarName]
	var modified []string
	if mod != nil {
		modified = mod.All()
	}
	vcmd := varcmd.VarCmd{Block: &b, Modified: modified, Vars: vars}
	for _, p := range b.Preps {
		cmd, err := vcmd.Render(p.Command)
		if initial && p.Onchange {
			log.Say(niceHeader("skipping prep: ", cmd))
			continue
		}
		if err != nil {
			return err
		}
		err = RunProc(cmd, shell, log.Stream(niceHeader("prep: ", cmd)))
		if err != nil {
			if pe, ok := err.(ProcError); ok {
				for _, n := range notifiers {
					n.Push("modd error", pe.Output, "")
				}
			}
			return err
		}
	}
	return nil
}
