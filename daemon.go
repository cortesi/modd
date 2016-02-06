package modd

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/termlog"
)

// MinRestart is the minimum amount of time between daemon restarts
const MinRestart = 1 * time.Second

// A single daemon
type daemon struct {
	conf conf.Daemon
	log  termlog.Stream
	cmd  *exec.Cmd
	stop bool
}

func (d *daemon) Run() {
	var lastStart time.Time
	for d.stop != true {
		d.log.Notice(">> starting...")
		since := time.Now().Sub(lastStart)
		if since < MinRestart {
			time.Sleep(MinRestart - since)
		}
		lastStart = time.Now()
		sh := getShell()

		c := exec.Command(sh, "-c", d.conf.Command)
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
		err = c.Start()
		if err != nil {
			d.log.Shout("%s", err)
			continue
		}
		d.cmd = c
		err = c.Wait()
		wg.Wait()
		if err != nil {
			d.log.Shout("%s", c.ProcessState.String())
			continue
		}
	}
}

func (d *daemon) Restart() {
	if d.cmd != nil {
		d.log.Notice(">> sending signal %s", d.conf.RestartSignal)
		d.cmd.Process.Signal(d.conf.RestartSignal)
	}
}

func (d *daemon) Shutdown(sig os.Signal) {
	d.stop = true
	if d.cmd != nil {
		d.cmd.Process.Signal(sig)
		d.cmd.Wait()
	}
}

// DaemonPen is a group of daemons in a single block, managed as a unit.
type DaemonPen struct {
	daemons []daemon
	sync.Mutex
}

// NewDaemonPen creates a new DaemonPen
func NewDaemonPen(block conf.Block, vars map[string]string, log termlog.TermLog) (*DaemonPen, error) {
	d := make([]daemon, len(block.Daemons))
	for i, dmn := range block.Daemons {
		vcmd := varcmd.VarCmd{Block: nil, Mod: nil, Vars: vars}
		finalcmd, err := vcmd.Render(dmn.Command)
		if err != nil {
			return nil, err
		}
		dmn.Command = finalcmd
		d[i] = daemon{
			conf: dmn,
			log:  log.Stream(niceHeader("daemon: ", dmn.Command)),
		}
	}
	return &DaemonPen{daemons: d}, nil
}

// Start starts set of daemons, each specified by a command
func (dp *DaemonPen) Start() {
	dp.Lock()
	defer dp.Unlock()
	for i := range dp.daemons {
		go dp.daemons[i].Run()
	}
}

// Restart all daemons in the pen
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

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		go func() {
			d.Shutdown(<-c)
			os.Exit(0)
		}()
	}
	return &DaemonWorld{daemonPens}, nil
}

// Start all daemon pens
func (dw *DaemonWorld) Start() {
	for _, dp := range dw.DaemonPens {
		dp.Start()
	}
}
