package modd

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/cortesi/termlog"
)

// MinRestart is the minimum amount of time between daemon restarts
const MinRestart = 1 * time.Second

func getShell() string {
	sh := os.Getenv("SHELL")
	if sh == "" {
		if _, err := os.Stat("/bin/sh"); err == nil {
			sh = "/bin/sh"
		}
	}
	return sh
}

func logOutput(fp io.ReadCloser, out func(string, ...interface{})) {
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out(string(line))
	}
}

// RunProc runs a process to completion, sending output to log
func RunProc(cmd string, log termlog.Logger) error {
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
	go logOutput(stde, log.Warn)
	go logOutput(stdo, log.Say)
	err = c.Start()
	if err != nil {
		return err
	}
	err = c.Wait()
	if err != nil {
		log.Shout("%s", c.ProcessState.String())
		return err
	}
	// FIXME: rusage stats here
	log.NoticeAs("cmdstats", "run time: %s", c.ProcessState.UserTime())
	return nil
}

// RunProcs runs all commands in sequence. Stops if any command returns an error.
func RunProcs(cmds []string, log termlog.TermLog) error {
	for _, cmd := range cmds {
		err := RunProc(
			cmd,
			log.Stream(cmd),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

type daemon struct {
	commandString string
	log           termlog.Logger
	cmd           *exec.Cmd
	stop          bool
}

func (d *daemon) Run() {
	var lastStart time.Time
	for d.stop != true {
		since := time.Now().Sub(lastStart)
		if since < MinRestart {
			time.Sleep(MinRestart - since)
		}
		lastStart = time.Now()
		sh := getShell()
		c := exec.Command(sh, "-c", d.commandString)
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
		go logOutput(stde, d.log.Warn)
		go logOutput(stdo, d.log.Say)
		err = c.Start()
		if err != nil {
			d.log.Shout("%s", err)
			continue
		}
		d.cmd = c
		err = c.Wait()
		if err != nil {
			d.log.Shout("%s", c.ProcessState.String())
			continue
		}
	}
}

func (d *daemon) Restart() {
	if d.cmd != nil {
		d.cmd.Process.Signal(syscall.SIGHUP)
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

const lineLimit = 80
const postamble = "..."

// niceName tries to produce a nicer process name. We condense whitespace to
// make commands split over multiple lines with indentation more legible, and
// limit the line length to 80 characters.
func niceName(in string) string {
	in = ws.ReplaceAllString(in, " ")
	if len(in) > lineLimit-len(postamble) {
		post := termlog.DefaultPalette.Say.SprintFunc()(postamble)
		return in[:lineLimit-len(postamble)] + post
	}
	return in
}

// Start starts set of daemons, each specified by a command
func (dp *DaemonPen) Start(commands []string, log termlog.TermLog) {
	dp.Lock()
	defer dp.Unlock()
	d := make([]daemon, len(commands))
	for i, c := range commands {
		d[i] = daemon{
			commandString: c,
			log:           log.Stream(niceName(c)),
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
