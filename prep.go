package modd

import (
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
func RunProc(cmd string, shellMethod string, dir string, log termlog.Stream) error {
	log.Header()
	ex, err := shell.NewExecutor(shellMethod, cmd, dir)
	if err != nil {
		return err
	}
	start := time.Now()
	err, estate := ex.Run(log, true)
	if err != nil {
		return err
	} else if estate.Error != nil {
		log.Shout("%s", estate.Error)
		return ProcError{estate.Error.Error(), estate.ErrOutput}
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
	sh, err := shell.GetShellName(vars[shellVarName])
	if err != nil {
		return err
	}

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
		err = RunProc(cmd, sh, b.InDir, log.Stream(niceHeader("prep: ", cmd)))
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
