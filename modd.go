package modd

import (
	"io/ioutil"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/watch"
	"github.com/cortesi/termlog"
)

const lullTime = time.Millisecond * 100

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

// Run is the top-level runner for modd
func Run(log termlog.TermLog, cnf *conf.Config, watchconf string, notifiers []notify.Notifier) (*conf.Config, error) {
	err := PrepOnly(log, cnf, notifiers)
	if err != nil {
		return nil, err
	}

	dworld, err := NewDaemonWorld(cnf, log)
	if err != nil {
		return nil, err
	}

	dworld.Start()
	watchpaths := cnf.WatchPatterns()
	if watchconf != "" {
		watchpaths = append(watchpaths, watchconf)
	}

	modchan := make(chan *watch.Mod, 1024)
	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	watcher, err := watch.Watch(watchpaths, lullTime, modchan)
	defer watcher.Stop()
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}

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
				return nil, err
			}
			dworld.DaemonPens[i].Restart()
		}
	}
	return nil, nil
}
