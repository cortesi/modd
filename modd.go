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
	"github.com/cortesi/modd/watch"
	"github.com/cortesi/termlog"
)

// Version is the modd release version
const Version = "0.2"

const lullTime = time.Millisecond * 100

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

	// Python
	"**.py[cod]",

	// Node
	"**/node_modules/**",
}

// PrepOnly runs all prep functions and exits
func PrepOnly(log termlog.TermLog, cnf *conf.Config, notifiers []notify.Notifier) error {
	for _, b := range cnf.Blocks {
		err := RunPreps(b, cnf.GetVariables(), nil, log, notifiers)
		if err != nil {
			return err
		}
	}
	return nil
}

// Gives control of chan to caller
func runOnChan(modchan chan *watch.Mod, readyCallback func(), log termlog.TermLog, cnf *conf.Config, watchconf string, notifiers []notify.Notifier) (*conf.Config, error) {
	err := PrepOnly(log, cnf, notifiers)
	if err != nil {
		return nil, err
	}

	dworld, err := NewDaemonWorld(cnf, log)
	if err != nil {
		return nil, err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	defer signal.Reset(os.Interrupt, os.Kill)
	defer dworld.Shutdown(os.Kill)
	go func() {
		dworld.Shutdown(<-c)
		os.Exit(0)
	}()

	dworld.Start()
	watchpaths := cnf.WatchPatterns()
	if watchconf != "" {
		watchpaths = append(watchpaths, filepath.Dir(watchconf))
	}

	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	watcher, err := watch.Watch(watchpaths, lullTime, modchan)
	if err != nil {
		return nil, fmt.Errorf("Error watching: %s", err)
	}
	defer watcher.Stop()
	go readyCallback()

	for mod := range modchan {
		if mod == nil {
			break
		}
		if watchconf != "" && mod.Has(watchconf) {
			ret, err := ioutil.ReadFile(watchconf)
			if err != nil {
				log.Warn("Reloading config - error reading %s: %s", watchconf, err)
				continue
			}
			newcnf, err := conf.Parse(watchconf, string(ret))
			if err != nil {
				log.Warn("Reloading config - error reading %s: %s", watchconf, err)
				continue
			}
			log.Notice("Reloading config %s", watchconf)
			return newcnf, nil
		}
		log.SayAs("debug", "Delta: \n%s", mod.String())
		for i, b := range cnf.Blocks {
			lmod, err := mod.Filter(b.Include, b.Exclude)
			if err != nil {
				log.Shout("Error filtering events: %s", err)
				continue
			}
			if lmod.Empty() {
				continue
			}
			err = RunPreps(b, cnf.GetVariables(), lmod, log, notifiers)
			if err != nil {
				if _, ok := err.(ProcError); ok {
					continue
				} else {
					return nil, err
				}
			}
			dworld.DaemonPens[i].Restart()
		}
	}
	return nil, nil
}

// Run is the top-level runner for modd
func Run(log termlog.TermLog, cnf *conf.Config, watchconf string, notifiers []notify.Notifier) (*conf.Config, error) {
	modchan := make(chan *watch.Mod, 1024)
	return runOnChan(modchan, func() {}, log, cnf, watchconf, notifiers)
}
