package modd

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/cortesi/termlog"
	"github.com/fatih/color"
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
	log.Say("%s %s", color.BlueString("running prep:"), cmd)
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
func RunProcs(cmds []string, log termlog.Logger) error {
	for _, cmd := range cmds {
		err := RunProc(cmd, log)
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
}

func (d *daemon) Run() {
	for {
		d.log.Say("%s %s", color.BlueString("starting daemon:"), d.commandString)
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

// DaemonPen is a group of daemons, managed as a unit.
type DaemonPen struct {
	daemons *[]daemon
}

// Start starts set of daemons, each specified by a command
func (dp *DaemonPen) Start(commands []string, log termlog.Logger) {
	d := make([]daemon, len(commands))
	for i, c := range commands {
		d[i] = daemon{
			commandString: c,
			log:           log,
		}
		go d[i].Run()
	}
	dp.daemons = &d
}

// Restart all daemons in the pen
func (dp *DaemonPen) Restart() {
	if dp.daemons != nil {
		for _, d := range *dp.daemons {
			d.Restart()
		}
	}
}
