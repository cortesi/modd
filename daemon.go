package modd

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/shell"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/termlog"
)

const (
	// MinRestart is the minimum amount of time between daemon restarts
	MinRestart = 500 * time.Millisecond
	// MulRestart is the exponential backoff multiplier applied when the daemon exits uncleanly
	MulRestart = 2
	// MaxRestart is the maximum amount of time between daemon restarts
	MaxRestart = 8 * time.Second
)

// A single daemon
type daemon struct {
	conf  conf.Daemon
	indir string

	running bool
	ex      *shell.Executor
	log     termlog.Stream
	shell   string
	stop    bool
	sync.Mutex
}

func (d *daemon) Run() {
	ex, err := shell.NewExecutor(d.shell, d.conf.Command, d.indir)
	if err != nil {
		d.log.Shout("Could not create executor: %s", err)
	}
	d.ex = ex

	var lastStart time.Time
	delay := MinRestart
	for d.stop != true {
		if !lastStart.IsZero() {
			d.log.Notice(">> sleeping... %#v", delay)
			time.Sleep(delay)
		}
		d.log.Notice(">> starting...")
		lastStart = time.Now()
		err, pstate := ex.Run(d.log, false)

		if err != nil {
			d.log.Shout("execution error: %s", err)
		} else if pstate.Error != nil {
			if _, ok := pstate.Error.(*exec.ExitError); ok {
				d.log.Warn("exited: %s", pstate.ProcState)
			} else {
				d.log.Shout("exited: %s", err)
			}
		} else {
			d.log.Warn("exited: %s", pstate.ProcState)
		}

		// If we exited cleanly, or the process ran for > MaxRestart, we reset
		// the delay timer
		if time.Now().Sub(lastStart) > MaxRestart {
			delay = MinRestart
		} else {
			delay *= MulRestart
			if delay > MaxRestart {
				delay = MaxRestart
			}
		}
	}
}

// Restart the daemon, or start it if it's not yet running
func (d *daemon) Restart() {
	d.Lock()
	defer d.Unlock()
	if !d.running {
		d.running = true
		go d.Run()
	} else {
		d.log.Notice(">> sending signal %s", d.conf.RestartSignal)
		err := d.ex.Signal(d.conf.RestartSignal)
		if err != nil {
			d.log.Warn(
				"failed to send %s signal to %s: %v", d.conf.RestartSignal, d.conf.Command, err,
			)
		}
	}
}

func (d *daemon) Shutdown(sig os.Signal) error {
	d.log.Notice(">> stopping")
	d.stop = true
	if d.ex != nil {
		return d.ex.Stop()
	}
	return nil
}

// DaemonPen is a group of daemons in a single block, managed as a unit.
type DaemonPen struct {
	daemons []*daemon
	sync.Mutex
}

// NewDaemonPen creates a new DaemonPen
func NewDaemonPen(block conf.Block, vars map[string]string, log termlog.TermLog) (*DaemonPen, error) {
	d := make([]*daemon, len(block.Daemons))
	for i, dmn := range block.Daemons {
		vcmd := varcmd.VarCmd{Block: nil, Modified: nil, Vars: vars}
		finalcmd, err := vcmd.Render(dmn.Command)
		if err != nil {
			return nil, err
		}
		dmn.Command = finalcmd
		var indir string
		if block.InDir != "" {
			indir = block.InDir
		} else {
			indir, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
		sh, err := shell.GetShellName(vars[shellVarName])
		if err != nil {
			return nil, err
		}

		d[i] = &daemon{
			conf:  dmn,
			log:   log.Stream(niceHeader("daemon: ", dmn.Command)),
			shell: sh,
			indir: indir,
		}
	}
	return &DaemonPen{daemons: d}, nil
}

// Restart all daemons in the pen, or start them if they're not running yet.
func (dp *DaemonPen) Restart() {
	dp.Lock()
	defer dp.Unlock()
	if dp.daemons != nil {
		for _, d := range dp.daemons {
			d.Restart()
		}
	}
}

// Shutdown all daemons in the pen
func (dp *DaemonPen) Shutdown(sig os.Signal) {
	dp.Lock()
	defer dp.Unlock()
	if dp.daemons != nil {
		for _, d := range dp.daemons {
			d.Shutdown(sig)
		}
	}
}

// DaemonWorld represents the entire world of daemons
type DaemonWorld struct {
	DaemonPens []*DaemonPen
}

// NewDaemonWorld creates a DaemonWorld
func NewDaemonWorld(cnf *conf.Config, log termlog.TermLog) (*DaemonWorld, error) {
	daemonPens := make([]*DaemonPen, len(cnf.Blocks))
	for i, b := range cnf.Blocks {
		d, err := NewDaemonPen(b, cnf.GetVariables(), log)
		if err != nil {
			return nil, err
		}
		daemonPens[i] = d

	}
	return &DaemonWorld{daemonPens}, nil
}

// Shutdown all daemons with signal s
func (dw *DaemonWorld) Shutdown(s os.Signal) {
	for _, dp := range dw.DaemonPens {
		dp.Shutdown(s)
	}
}
