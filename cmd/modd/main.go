package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
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

var cmdstats = kingpin.Flag("cmdstats", "Show stats on command execution").
	Short('s').
	Default("false").
	Bool()

var debug = kingpin.Flag("debug", "Debugging for modd development").
	Default("false").
	Bool()

func prepsAndNotify(b conf.Block, lmod *modd.Mod, log termlog.TermLog) error {
	err := modd.RunPreps(b, lmod, log)
	if err != nil {
		if *beep {
			fmt.Print("\a")
		}
	}
	if pe, ok := err.(modd.ProcError); ok && *doNotify {
		n := notify.NewNotifier()
		if n != nil {
			n.Push("modd error", pe.Output, "")
		}
	}
	return err
}

func run(log termlog.TermLog, cnf *conf.Config, watchconf string) *conf.Config {
	modchan := make(chan *modd.Mod, 1024)
	if *ignores {
		for _, patt := range modd.CommonExcludes {
			fmt.Println(patt)
		}
		os.Exit(0)
	}

	daemonPens := make([]*modd.DaemonPen, len(cnf.Blocks))
	for i, b := range cnf.Blocks {
		if !b.NoCommonFilter {
			b.Exclude = append(b.Exclude, modd.CommonExcludes...)
		}
		cnf.Blocks[i] = b

		prepsAndNotify(b, nil, log)

		d := modd.DaemonPen{}
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		go func() {
			d.Shutdown(<-c)
			os.Exit(0)
		}()
		if !*prep {
			d.Start(b.Daemons, log)
		}
		daemonPens[i] = &d
	}
	if *prep {
		os.Exit(0)
	}

	watchpaths := cnf.WatchPaths()
	if watchconf != "" {
		watchpaths = append(watchpaths, watchconf)
	}

	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	watcher, err := modd.Watch(watchpaths, lullTime, modchan)
	defer watcher.Stop()
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}

	for mod := range modchan {
		if watchconf != "" && mod.Has(watchconf) {
			ret, err := ioutil.ReadFile(watchconf)
			if err != nil {
				log.Warn("Reloading config - error reading %s: %s", watchconf, err)
				continue
			}
			cnf, err := conf.Parse(*file, string(ret))
			if err != nil {
				log.Warn("Reloading config - error reading %s: %s", watchconf, err)
				continue
			}
			log.Notice("Reloading config %s", watchconf)
			return cnf
		}
		if mod == nil {
			break
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

			err = prepsAndNotify(b, lmod, log)
			if err != nil {
				continue
			}
			daemonPens[i].Restart()
		}
	}
	return nil
}

func main() {
	kingpin.Version(modd.Version)
	kingpin.Parse()

	log := termlog.NewLog()
	if *debug {
		log.Enable("debug")
		modd.Logger = log
	}
	if *cmdstats {
		log.Enable("cmdstats")
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
		cnf = run(log, cnf, watchfile)
		if cnf == nil {
			break
		}
	}
}
