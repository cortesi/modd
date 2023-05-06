package shell

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cortesi/termlog"
)

type cmdTest struct {
	name    string
	cmd     string
	bufferr bool

	shells []string

	logHas  string
	buffHas string
	err     bool
	procerr bool
	kill    bool
}

func testCmd(t *testing.T, shell string, ct cmdTest) {
	if ct.shells != nil {
		issh := func() bool {
			for _, v := range ct.shells {
				if v == shell {
					return true
				}
			}
			return false
		}()
		if !issh {
			t.Skip("skipping")
			return
		}
	}

	lt := termlog.NewLogTest()
	exec, err := NewExecutor(shell, ct.cmd, "")
	if err != nil {
		t.Error(err)
		return
	}
	type result struct {
		err    error
		pstate *ExecState
	}

	ch := make(chan result)
	go func() {
		err, pstate := exec.Run(lt.Log.Stream(""), ct.bufferr)
		ch <- result{err: err, pstate: pstate}
	}()

	// Wait for the first output to make sure process is running
	for {
		time.Sleep(100 * time.Millisecond)
		if lt.String() != "" {
			break
		}
	}

	if ct.kill {
		err := exec.Stop()
		if err != nil {
			t.Errorf("Error stopping: %s", err)
			return
		}
		time.Sleep(1 * time.Second)
	}

	res := <-ch
	if (res.err != nil) != ct.err {
		t.Errorf("Unexpected invocation error: %s", err)
	}
	if (res.pstate.Error != nil) != ct.procerr {
		t.Errorf("Unexpected process error: %s, %s", res.pstate.Error, res.pstate.ErrOutput)
	}
	if ct.buffHas != "" && !strings.Contains(res.pstate.ErrOutput, ct.buffHas) {
		t.Errorf("Unexpected buffer return: %s", res.pstate.ErrOutput)
	}
	if ct.logHas != "" && !strings.Contains(lt.String(), ct.logHas) {
		t.Errorf("Unexpected log return: %s", lt.String())
	}
}

var longLine = func() string {
	// 6K is longer than the default buffer size, but still well under the
	// powershell command line length limit of 8191
	var runes [6 * 1024]rune
	for i := range runes {
		runes[i] = rune('0' + i%10)
	}
	return string(runes[:])
}()

var shellTests = []cmdTest{
	{
		name:   "echosuccess",
		cmd:    "echo moddtest; true",
		logHas: "moddtest",
	},
	{
		name:    "echofail",
		cmd:     "echo moddtest; false",
		logHas:  "moddtest",
		procerr: true,
	},
	{
		name:    "unknowncmd",
		cmd:     "definitelynosuchcommand",
		procerr: true,
	},
	{
		name:    "stderr-posix",
		cmd:     "echo moddstderr >&2",
		bufferr: true,
		buffHas: "moddstderr",
		shells:  []string{"modd", "sh", "bash"},
	},
	{
		name:    "stderr-powershell",
		cmd:     "Write-Error \"moddstderr\"",
		bufferr: true,
		procerr: true,
		buffHas: "moddstderr",
		shells:  []string{"powershell"},
	},
	{
		name:    "kill",
		cmd:     "echo moddtest; echo; sleep 999999",
		logHas:  "moddtest",
		kill:    true,
		procerr: true,
	},
	{
		name:   "longline",
		cmd:    "echo " + longLine + "; true",
		logHas: longLine,
	},
}

func TestShells(t *testing.T) {
	shellTesting = true

	var shells []string
	if runtime.GOOS == "windows" {
		shells = []string{
			"modd",
			"powershell",
		}
	} else {
		shells = []string{
			"sh",
			"bash",
			"modd",
			"powershell",
		}
	}
	for _, sh := range shells {
		for _, tc := range shellTests {
			t.Run(
				fmt.Sprintf("%s/%s", sh, tc.name),
				func(t *testing.T) {
					if _, err := CheckShell(sh); err != nil {
						t.Skipf("skipping - %s", err)
						return
					}
					testCmd(t, sh, tc)
				},
			)
		}
	}
}

func TestCaseInsensitivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping - only windows has case insensitive PATH")
	}

	oldpath := os.Getenv("PATH")
	fixpath := func() {
		os.Unsetenv("Path")
		os.Setenv("PATH", oldpath)
	}
	defer fixpath()
	os.Unsetenv("PATH")
	os.Setenv("Path", fmt.Sprintf("%s%ctrigger-text", oldpath, os.PathListSeparator))

	shellTesting = true

	pathTest := cmdTest{
		name:   "path-test",
		cmd:    "echo $PATH",
		logHas: "trigger-text",
	}
	sh := "modd"
	t.Run(
		"modd/path-capitalization",
		func(t *testing.T) {
			if _, err := CheckShell(sh); err != nil {
				t.Skipf("skipping - %s", err)
			}
			testCmd(t, sh, pathTest)
		},
	)
}
