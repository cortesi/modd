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

var ValidShells = map[string]bool{
	"builtin": true,
	"bash":    true,
	"exec":    true,
	"sh":      true,
}

var Default = "builtin"

type Executor struct {
	Shell   string
	Command string
	Dir     string

	cmd *exec.Cmd
	sync.Mutex
}

type ExecState struct {
	Error     error
	ErrOutput string
	ProcState string
}

func GetShellName(v string) (string, error) {
	if v == "" {
		return Default, nil
	}
	if _, ok := ValidShells[v]; !ok {
		return "", fmt.Errorf("Unsupported shell: %q", v)
	} else {
		return v, nil
	}
}

func NewExecutor(shell string, command string, dir string) (*Executor, error) {
	_, err := makeCommand(shell, command, dir)
	if err != nil {
		return nil, err
	}
	return &Executor{Shell: shell, Command: command, Dir: dir}, nil
}

func (e *Executor) start(
	log termlog.Stream, bufferr bool,
) (*exec.Cmd, *bytes.Buffer, *sync.WaitGroup, error) {
	e.Lock()
	defer e.Unlock()

	cmd, err := makeCommand(e.Shell, e.Command, e.Dir)
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

func (e *Executor) Run(log termlog.Stream, bufferr bool) (error, *ExecState) {
	if e.cmd != nil {
		return fmt.Errorf("already running"), nil
	}
	cmd, buff, wg, err := e.start(log, bufferr)
	if err != nil {
		return err, nil
	}

	// Order is important here. We MUST wait for the readers to exit before we wait
	// on the command itself.
	wg.Wait()

	eret := cmd.Wait()
	estate := &ExecState{
		Error:     eret,
		ErrOutput: buff.String(),
		ProcState: cmd.ProcessState.String(),
	}
	e.reset()
	return nil, estate
}

func (e *Executor) Signal(sig os.Signal) error {
	e.Lock()
	defer e.Unlock()
	if !e.running() {
		return fmt.Errorf("executor not running")
	}
	return sendSignal(e.cmd, sig)
}

func (e *Executor) Stop() error {
	return e.Signal(os.Kill)
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

func makeCommand(shell string, command string, dir string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch shell {
	case "exec":
		ss, err := shlex.Split(command)
		if err != nil {
			return nil, err
		}
		if len(ss) == 0 {
			return nil, errors.New("No command defined")
		}
		cmd = exec.Command(ss[0], ss[1:]...)
	case "bash", "sh":
		sh, err := getBash()
		if err != nil {
			return nil, fmt.Errorf("Could not find bash or sh")
		}
		cmd = exec.Command(sh, "-c", command)
	case "builtin":
		path, err := os.Executable()
		if err != nil {
			return nil, err
		}
		cmd = exec.Command(path, "--exec", command)
	default:
		return nil, fmt.Errorf("Unknown shell: %s", shell)
	}
	cmd.Dir = dir
	prepCmd(cmd)
	return cmd, nil
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
