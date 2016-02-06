package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/watch"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
)

const modfile = "./modd.conf"
const lullTime = time.Millisecond * 100

var file = kingpin.Flag(
	"file",
	fmt.Sprintf("Path to modfile (%s)", modfile),
).
	Default(modfile).
	PlaceHolder("PATH").
	Short('f').
	String()

var noconf = kingpin.Flag("noconf", "Don't watch our own config file").
	Short('c').
	Bool()

var beep = kingpin.Flag("bell", "Ring terminal bell if any command returns an error").
	Short('b').
	Bool()

var ignores = kingpin.Flag("ignores", "List default ignore patterns and exit").
	Short('i').
	Bool()

var doNotify = kingpin.Flag("notify", "Send stderr to system notification if commands error").
	Short('n').
	Bool()

var prep = kingpin.Flag("prep", "Run prep commands and exit").
	Short('p').
	Bool()

var debug = kingpin.Flag("debug", "Debugging for modd development").
	Default("false").
	Bool()

// Returns a (continue, error) tuple. If continue is true, execution of the
// remainder of the block should proceed. If error is not nil, modd should
// exit.
func prepsAndNotify(b conf.Block, vars map[string]string, lmod *watch.Mod, log termlog.TermLog) (bool, error) {
	err := modd.RunPreps(b, vars, lmod, log)
	if pe, ok := err.(modd.ProcError); ok {
		if *beep {
			fmt.Print("\a")
		}
		if *doNotify {
			n := notify.NewNotifier()
			if n == nil {
				log.Shout("Could not find a desktop notifier")
			} else {
				n.Push("modd error", pe.Output, "")
			}
		}
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func run(log termlog.TermLog, cnf *conf.Config, watchconf string) *conf.Config {
	for _, b := range cnf.Blocks {
		_, err := prepsAndNotify(b, cnf.GetVariables(), nil, log)
		if err != nil {
			log.Shout("%s", err)
			return nil
		}
	}
	dworld, err := modd.NewDaemonWorld(cnf, log)
	if err != nil {
		log.Shout("%s", err)
		return nil
	}
	if *prep {
		return nil
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
			newcnf, err := conf.Parse(*file, string(ret))
			if err != nil {
				log.Warn("Reloading config - error reading %s: %s", watchconf, err)
				continue
			}
			log.Notice("Reloading config %s", watchconf)
			return newcnf
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

			proceed, err := prepsAndNotify(b, cnf.GetVariables(), lmod, log)
			if err != nil {
				log.Shout("%s", err)
				return nil
			}
			if !proceed {
				continue
			}
			dworld.DaemonPens[i].Restart()
		}
	}
	return nil
}

func main() {
	kingpin.Version(watch.Version)
	kingpin.Parse()

	if *ignores {
		for _, patt := range watch.CommonExcludes {
			fmt.Println(patt)
		}
		os.Exit(0)
	}

	log := termlog.NewLog()
	if *debug {
		log.Enable("debug")
		watch.Logger = log
	}

	ret, err := ioutil.ReadFile(*file)
	if err != nil {
		kingpin.Fatalf("%s", err)
	}
	cnf, err := conf.Parse(*file, string(ret))
	if err != nil {
		kingpin.Fatalf("%s", err)
	}
	watchfile := *file
	if *noconf {
		watchfile = ""
	}

	for {
		cnf.CommonExcludes(watch.CommonExcludes)
		cnf = run(log, cnf, watchfile)
		if cnf == nil {
			break
		}
	}
}
