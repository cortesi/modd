package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/cortesi/termlog"
)

var ValidShells = map[string]bool{
	"bash":       true,
	"modd":       true,
	"powershell": true,
	"sh":         true,
}

var shellTesting bool

var Default = "modd"

type Executor struct {
	Shell   string
	Command string
	Dir     string

	cmd  *exec.Cmd
	stdo io.ReadCloser
	stde io.ReadCloser
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
	}
	return v, nil
}

func NewExecutor(shell string, command string, dir string) (*Executor, error) {
	_, err := makeCommand(shell, command, dir)
	if err != nil {
		return nil, err
	}
	return &Executor{
		Shell:   shell,
		Command: command,
		Dir:     dir,
	}, nil
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
	e.stdo = stdo
	e.stde = stde

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
	return e.sendSignal(sig)
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

// CheckShell checks that a shell is supported, and returns the correct command name
func CheckShell(shell string) (string, error) {
	if _, ok := ValidShells[shell]; !ok {
		return "", fmt.Errorf("unsupported shell: %q", shell)
	}
	switch shell {
	case "powershell":
		if _, err := exec.LookPath("powershell"); err == nil {
			return "powershell", nil
		} else if _, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh", nil
		} else {
			return "", fmt.Errorf("powershell/pwsh not on path")
		}
	case "modd":
		// When testing, we're running under a special compiled test executable,
		// so we look for an instance of modd on our path.
		if shellTesting {
			return exec.LookPath("modd")
		}
		return os.Executable()
	default:
		return exec.LookPath(shell)
	}
}

func makeCommand(shell string, command string, dir string) (*exec.Cmd, error) {
	shcmd, err := CheckShell(shell)
	if err != nil {
		return nil, err
	}
	var cmd *exec.Cmd
	switch shell {
	case "bash", "sh":
		cmd = exec.Command(shcmd, "-c", command)
	case "modd":
		cmd = exec.Command(shcmd, "--exec", command)
	case "powershell":
		cmd = exec.Command(shcmd, "-Command", command)
	}
	cmd.Dir = dir
	prepCmd(cmd)
	return cmd, nil
}
