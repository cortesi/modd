package modd

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/modd/watch"
	"github.com/cortesi/termlog"
)

const moddVar = "@mods"

// MinRestart is the minimum amount of time between daemon restarts
const MinRestart = 1 * time.Second

const lineLimit = 80

// shortCommand shortens a command to a name we can use in a notification
// header.
func shortCommand(command string) string {
	ret := command
	parts := strings.Split(command, "\n")
	for _, i := range parts {
		i = strings.TrimLeft(i, " \t#")
		i = strings.TrimRight(i, " \t\\")
		if i != "" {
			ret = i
			break
		}
	}
	return ret
}

// niceHeader tries to produce a nicer process name. We condense whitespace to
// make commands split over multiple lines with indentation more legible, and
// limit the line length to 80 characters.
func niceHeader(preamble string, command string) string {
	pre := termlog.DefaultPalette.Timestamp.SprintFunc()(preamble)
	command = termlog.DefaultPalette.Header.SprintFunc()(shortCommand(command))
	return pre + command
}

func getShell() string {
	return "bash"
}

func logOutput(wg *sync.WaitGroup, fp io.ReadCloser, out func(string, ...interface{})) {
	defer wg.Done()
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out(string(line))
	}
}

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
	wg := sync.WaitGroup{}
	wg.Add(2)
	buff := new(bytes.Buffer)
	mut := sync.Mutex{}
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
	err = c.Start()
	if err != nil {
		return err
	}
	err = c.Wait()
	wg.Wait()
	if err != nil {
		log.Shout("%s", c.ProcessState.String())
		return ProcError{err.Error(), buff.String()}
	}
	log.Notice(">> done (%s)", c.ProcessState.UserTime())
	return nil
}

// RunPreps runs all commands in sequence. Stops if any command returns an error.
func RunPreps(b conf.Block, vars map[string]string, mod *watch.Mod, log termlog.TermLog) error {
	vcmd := varcmd.VarCmd{Block: &b, Mod: mod, Vars: vars}
	for _, p := range b.Preps {
		cmd, err := vcmd.Render(p.Command)
		if err != nil {
			return err
		}
		err = RunProc(cmd, log.Stream(niceHeader("prep: ", cmd)))
		if err != nil {
			return err
		}
	}
	return nil
}

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

// DaemonPen is a group of daemons, managed as a unit.
type DaemonPen struct {
	daemons *[]daemon
	sync.Mutex
}

var ws = regexp.MustCompile(`\s\s+`)

// Start starts set of daemons, each specified by a command
func (dp *DaemonPen) Start(daemons []conf.Daemon, vars map[string]string, log termlog.TermLog) {
	dp.Lock()
	defer dp.Unlock()
	d := make([]daemon, len(daemons))
	for i, dmn := range daemons {
		vcmd := varcmd.VarCmd{Block: nil, Mod: nil, Vars: vars}
		finalcmd, err := vcmd.Render(dmn.Command)
		if err != nil {
			log.Shout("%s", err)
			continue
		}
		dmn.Command = finalcmd
		d[i] = daemon{
			conf: dmn,
			log: log.Stream(
				niceHeader("daemon: ", dmn.Command),
			),
		}
		go d[i].Run()
	}
	dp.daemons = &d
}

// Restart all daemons in the pen
func (dp *DaemonPen) Restart() {
	dp.Lock()
	defer dp.Unlock()
	if dp.daemons != nil {
		for _, d := range *dp.daemons {
			d.Restart()
		}
	}
}

// Shutdown all daemons in the pen
func (dp *DaemonPen) Shutdown(sig os.Signal) {
	dp.Lock()
	defer dp.Unlock()
	if dp.daemons != nil {
		for _, d := range *dp.daemons {
			d.Shutdown(sig)
		}
	}
}
