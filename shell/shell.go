package shell

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"github.com/google/shlex"
)

// Default Inteface to use if none provided.
var Default = "bash"

// Interface to the shell.
type Interface interface {
	// Name of the shell interface.
	Name() string

	// Put in a exec command line and get a Cmd.
	Command(line string) (*exec.Cmd, error)
}

var (
	shellsLock sync.Mutex
	shells     = make(map[string]Interface)
)

func init() {
	Register(&Raw{})
	Register(&Bash{})
}

// Register a new shell interface.
func Register(i Interface) {
	shellsLock.Lock()
	defer shellsLock.Unlock()

	name := i.Name()
	if _, has := shells[name]; has {
		panic("shell interface " + name + " already exists")
	}
	shells[name] = i
}

// Command returns a *Cmd. If method is empty then the default shell
// interface method is used. The line should contain the exec line.
func Command(method, line string) (*exec.Cmd, error) {
	shellsLock.Lock()
	defer shellsLock.Unlock()

	if len(method) == 0 {
		method = Default
	}

	i, has := shells[method]
	if !has {
		return nil, fmt.Errorf("Shell method %q not found", method)
	}
	return i.Command(line)
}

// No shell, just execute the command raw.
type Raw struct{}

func (r *Raw) Name() string {
	return "raw"
}

func (r *Raw) Command(line string) (*exec.Cmd, error) {
	ss, err := shlex.Split(line)
	if err != nil {
		return nil, err
	}
	if len(ss) == 0 {
		return nil, errors.New("No command defined")
	}
	return exec.Command(ss[0], ss[1:]...), nil
}

// Bash shell command.
type Bash struct{}

func (b *Bash) Name() string {
	return "bash"
}

func (b *Bash) getShell() (string, error) {
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash", nil
	}
	if _, err := exec.LookPath("sh"); err == nil {
		return "sh", nil
	}
	return "", fmt.Errorf("Could not find bash or sh on path.")
}

func (b *Bash) Command(line string) (*exec.Cmd, error) {
	sh, err := b.getShell()
	if err != nil {
		return nil, err
	}
	return exec.Command(sh, "-c", line), nil
}
