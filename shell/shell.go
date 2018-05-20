package shell

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/cortesi/termlog"
	"github.com/google/shlex"
)

var Default = "bash"

type ExitError error

type Executor struct {
	Shell   string
	Command string

	cmd *exec.Cmd
	sync.Mutex
}

func NewExecutor(shell string, command string) (*Executor, error) {
	_, err := makeCommand(shell, command)
	if err != nil {
		return nil, err
	}
	return &Executor{Shell: shell, Command: command}, nil
}

func (e *Executor) start(
	log termlog.Stream, bufferr bool,
) (*exec.Cmd, *bytes.Buffer, *sync.WaitGroup, error) {
	e.Lock()
	defer e.Unlock()

	cmd, err := makeCommand(e.Shell, e.Command)
	if err != nil {
		return nil, nil, nil, err
	}
	e.cmd = cmd

	stdo, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stde, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	buff := new(bytes.Buffer)
	err = cmd.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	buflock := sync.Mutex{}
	go logOutput(
		&wg, stde,
		func(s string, args ...interface{}) {
			log.Warn(s, args...)
			if bufferr {
				buflock.Lock()
				defer buflock.Unlock()
				fmt.Fprintf(buff, "%s\n", args...)
			}
		},
	)
	go logOutput(&wg, stdo, log.Say)
	return cmd, buff, &wg, nil
}

func (e *Executor) running() bool {
	return e.cmd != nil
}

func (e *Executor) Running() bool {
	e.Lock()
	defer e.Unlock()
	return e.running()
}

func (e *Executor) reset() {
	e.Lock()
	defer e.Unlock()
	e.cmd = nil
}

func (e *Executor) Run(log termlog.Stream, bufferr bool) (error, ExitError, string) {
	cmd, buff, wg, err := e.start(log, bufferr)
	if err != nil {
		return err, nil, ""
	}
	eret := cmd.Wait()
	wg.Wait()
	e.reset()
	return nil, eret, buff.String()
}

func (e *Executor) Signal(sig os.Signal) error {
	e.Lock()
	defer e.Unlock()
	if !e.running() {
		return fmt.Errorf("executor not running")
	}
	return e.cmd.Process.Signal(sig)
}

func (e *Executor) Stop() error {
	e.Lock()
	defer e.Unlock()
	if !e.running() {
		return fmt.Errorf("executor not running")
	}
	return e.cmd.Process.Kill()
}

func logOutput(wg *sync.WaitGroup, fp io.ReadCloser, out func(string, ...interface{})) {
	defer wg.Done()
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out("%s", string(line))
	}
}

func makeCommand(shell string, command string) (*exec.Cmd, error) {
	switch shell {
	case "exec":
		ss, err := shlex.Split(command)
		if err != nil {
			return nil, err
		}
		if len(ss) == 0 {
			return nil, errors.New("No command defined")
		}
		return exec.Command(ss[0], ss[1:]...), nil
	case "bash":
		sh, err := getBash()
		if err != nil {
			return nil, fmt.Errorf("Could not find bash or sh")
		}
		return exec.Command(sh, "-c", command), nil
	case "builtin":
		path, err := os.Executable()
		if err != nil {
			return nil, err
		}
		return exec.Command(path, "--exec", command), nil
	default:
		return nil, fmt.Errorf("Unknown shell: %s", shell)
	}
}

func getBash() (string, error) {
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash", nil
	}
	if _, err := exec.LookPath("sh"); err == nil {
		return "sh", nil
	}
	return "", fmt.Errorf("could not find bash or sh on path")
}
