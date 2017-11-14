package modd

import (
	"fmt"
	"io"
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
	MinRestart = 1 * time.Second
	// MulRestart is the exponential backoff multiplier applied when the daemon exits uncleanly
	MulRestart = 2
	// MaxRestart is the maximum amount of time between daemon restarts
	MaxRestart = 5 * time.Second
)

// A single daemon
type daemon struct {
	conf  conf.Daemon
	indir string

	log     termlog.Stream
	cmd     *exec.Cmd
	stdin   io.Writer
	shell   string
	stop    bool
	started bool
	sync.Mutex
}

func (d *daemon) Run() {
	var lastStart time.Time
	delay := MinRestart
	for d.stop != true {
		d.log.Notice(">> starting...")
		since := time.Now().Sub(lastStart)
		if since < delay {
			time.Sleep(delay - since)
		}
		lastStart = time.Now()

		c, err := shell.Command(d.shell, d.conf.Command)
		if err != nil {
			d.log.Shout("%s", err)
			return
		}
		c.Dir = d.indir
		if d.conf.PipeRestartSignal {
			d.stdin, err = c.StdinPipe()
			if err != nil {
				d.log.Shout("%s", err)
				continue
			}
		}
		stdo, err := c.StdoutPipe()
		if err != nil {
			d.log.Shout("%s", err)
			continue
		}
		stde, err := c.StderrPipe()
		if err != nil {
			d.log.Shout("%s", err)
			continue
		}
		wg := sync.WaitGroup{}
		wg.Add(2)
		go logOutput(&wg, stde, d.log.Warn)
		go logOutput(&wg, stdo, d.log.Say)

		d.Lock()
		err = c.Start()
		if err != nil {
			d.log.Shout("%s", err)
			d.Unlock()
			continue
		}
		d.cmd = c
		d.Unlock()

		wg.Wait()
		err = c.Wait()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				d.log.Warn("exited: %s", c.ProcessState.String())
			} else {
				d.log.Shout("exited: %s", err)
			}
			// unclean restart; increase backoff
			delay *= MulRestart
			if delay > MaxRestart {
				delay = MaxRestart
			}
		} else {
			d.log.Warn("exited: %s", c.ProcessState.String())
			// clean restart; reset backoff
			delay = MinRestart
		}
	}
}

// Go's standard library uses a mix of word tenses for the string
// representation of signals. For our 'pipe signals' feature, the strings we
// write on the pipe should be present-tense because they are conceptually
// requests.
var pipeSignalTenseCorrections = map[string]string{
	"aborted":    "abort",
	"killed":     "kill",
	"terminated": "terminate",
}

// Restart the daemon, or start it if it's not yet running
func (d *daemon) Restart() {
	d.Lock()
	defer d.Unlock()
	if !d.started {
		go d.Run()
		d.started = true
	} else {
		if d.cmd != nil {
			if d.conf.PipeRestartSignal {
				d.log.Notice(">> piping signal %s", d.conf.RestartSignal)
				sigStr := d.conf.RestartSignal.String()
				if s, ok := pipeSignalTenseCorrections[sigStr]; ok {
					sigStr = s
				}
				fmt.Fprintln(d.stdin, sigStr)
			} else {
				d.log.Notice(">> sending signal %s", d.conf.RestartSignal)
				d.cmd.Process.Signal(d.conf.RestartSignal)
			}
		}
	}
}

func (d *daemon) Shutdown(sig os.Signal) {
	d.Lock()
	defer d.Unlock()
	d.stop = true
	if d.cmd != nil {
		d.cmd.Process.Signal(sig)
	}
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
		vcmd := varcmd.VarCmd{Block: nil, Mod: nil, Vars: vars}
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

		d[i] = &daemon{
			conf:  dmn,
			log:   log.Stream(niceHeader("daemon: ", dmn.Command)),
			shell: vars[shellVarName],
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
