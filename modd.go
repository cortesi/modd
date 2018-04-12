package modd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/shell"
	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
)

// Version is the modd release version
const Version = "0.6"

const lullTime = time.Millisecond * 100

const shellVarName = "@shell"

// CommonExcludes is a list of commonly excluded files suitable for passing in
// the excludes parameter to Watch - includes repo directories, temporary
// files, and so forth.
var CommonExcludes = []string{
	// VCS
	"**/.git/**",
	"**/.hg/**",
	"**/.svn/**",
	"**/.bzr/**",

	// OSX
	"**/.DS_Store/**",

	// Temporary files
	"**.tmp",
	"**~",
	"**#",
	"**.bak",
	"**.swp",
	"**.___jb_old___",
	"**.___jb_bak___",

	// Python
	"**.py[cod]",

	// Node
	"**/node_modules/**",
}

// ModRunner coordinates running the modd command
type ModRunner struct {
	Log        termlog.TermLog
	Config     *conf.Config
	ConfPath   string
	ConfReload bool
	Notifiers  []notify.Notifier
}

// NewModRunner constructs a new ModRunner
func NewModRunner(confPath string, log termlog.TermLog, notifiers []notify.Notifier, confreload bool) (*ModRunner, error) {
	mr := &ModRunner{
		Log:        log,
		ConfPath:   confPath,
		ConfReload: confreload,
		Notifiers:  notifiers,
	}
	err := mr.ReadConfig()
	if err != nil {
		return nil, err
	}
	return mr, nil
}

// ReadConfig parses the configuration file in ConfPath
func (mr *ModRunner) ReadConfig() error {
	ret, err := ioutil.ReadFile(mr.ConfPath)
	if err != nil {
		return fmt.Errorf("Error reading config file %s: %s", mr.ConfPath, err)
	}
	newcnf, err := conf.Parse(mr.ConfPath, string(ret))
	if err != nil {
		return fmt.Errorf("Error reading config file %s: %s", mr.ConfPath, err)
	}

	shellMethod := newcnf.GetVariables()[shellVarName]
	if shellMethod != "" && !shell.Has(shellMethod) {
		return fmt.Errorf("No such shell: %q", shellMethod)
	}

	newcnf.CommonExcludes(CommonExcludes)
	mr.Config = newcnf
	return nil
}

// PrepOnly runs all prep functions and exits
func (mr *ModRunner) PrepOnly(initial bool) error {
	for _, b := range mr.Config.Blocks {
		err := RunPreps(b, mr.Config.GetVariables(), nil, mr.Log, mr.Notifiers, initial)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mr *ModRunner) runBlock(b conf.Block, mod *moddwatch.Mod, dpen *DaemonPen) {
	if b.InDir != "" {
		currentDir, err := os.Getwd()
		if err != nil {
			mr.Log.Shout("Error getting current working directory: %s", err)
			return
		}
		err = os.Chdir(b.InDir)
		if err != nil {
			mr.Log.Shout(
				"Error changing to indir directory \"%s\": %s",
				b.InDir,
				err,
			)
			return
		}
		defer func() {
			err := os.Chdir(currentDir)
			if err != nil {
				mr.Log.Shout("Error returning to original directory: %s", err)
			}
		}()
	}
	err := RunPreps(
		b,
		mr.Config.GetVariables(),
		mod, mr.Log,
		mr.Notifiers,
		mod == nil,
	)
	if err != nil {
		if _, ok := err.(ProcError); !ok {
			mr.Log.Shout("Error running prep: %s", err)
		}
		return
	}
	dpen.Restart()
}

func (mr *ModRunner) trigger(root string, mod *moddwatch.Mod, dworld *DaemonWorld) {
	for i, b := range mr.Config.Blocks {
		lmod := mod
		if lmod != nil {
			var err error
			lmod, err = mod.Filter(root, b.Include, b.Exclude)
			if err != nil {
				mr.Log.Shout("Error filtering events: %s", err)
				continue
			}
			if lmod.Empty() {
				continue
			}
		}
		mr.runBlock(b, lmod, dworld.DaemonPens[i])
	}
}

// Gives control of chan to caller
func (mr *ModRunner) runOnChan(modchan chan *moddwatch.Mod, readyCallback func()) error {
	dworld, err := NewDaemonWorld(mr.Config, mr.Log)
	if err != nil {
		return err
	}
	defer dworld.Shutdown(os.Kill)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	defer signal.Reset(os.Interrupt, os.Kill)
	go func() {
		dworld.Shutdown(<-c)
		os.Exit(0)
	}()

	ipatts := mr.Config.IncludePatterns()
	if mr.ConfReload {
		ipatts = append(ipatts, filepath.Dir(mr.ConfPath))
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	watcher, err := moddwatch.Watch(currentDir, ipatts, []string{}, lullTime, modchan)

	if err != nil {
		return fmt.Errorf("Error watching: %s", err)
	}
	defer watcher.Stop()

	mr.trigger(currentDir, nil, dworld)
	go readyCallback()
	for mod := range modchan {
		if mod == nil {
			break
		}
		if mr.ConfReload && mod.Has(mr.ConfPath) {
			mr.Log.Notice("Reloading config %s", mr.ConfPath)
			err := mr.ReadConfig()
			if err != nil {
				mr.Log.Warn("%s", err)
				continue
			} else {
				return nil
			}
		}
		mr.Log.SayAs("debug", "Delta: \n%s", mod.String())
		mr.trigger(currentDir, mod, dworld)
	}
	return nil
}

// Run is the top-level runner for modd
func (mr *ModRunner) Run() error {
	for {
		modchan := make(chan *moddwatch.Mod, 1024)
		err := mr.runOnChan(modchan, func() {})
		if err != nil {
			return err
		}
	}
}
