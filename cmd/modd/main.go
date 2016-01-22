package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/conf"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
)

const modfile = "./modd.conf"
const lullTime = time.Millisecond * 100

func main() {
	file := kingpin.Flag(
		"file",
		fmt.Sprintf("Path to modfile (%s)", modfile),
	).
		Default(modfile).
		PlaceHolder("PATH").
		Short('f').
		String()

	beep := kingpin.Flag("beep", "Beep if any command returned an error").
		Short('b').
		Bool()

	prep := kingpin.Flag("prep", "Run prep commands and exit").
		Short('p').
		Bool()

	cmdstats := kingpin.Flag("cmdstats", "Show stats on command execution").
		Short('s').
		Default("false").
		Bool()

	debug := kingpin.Flag("debug", "Debugging for modd development").
		Default("false").
		Bool()

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

	daemonPens := make([]*modd.DaemonPen, len(cnf.Blocks))
	for i, b := range cnf.Blocks {
		if !b.NoCommonFilter {
			b.Exclude = append(b.Exclude, modd.CommonExcludes...)
		}
		cnf.Blocks[i] = b

		err = modd.RunPreps(b, nil, log)
		if err != nil {
			if *beep {
				fmt.Print("\a")
			}
		}
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

	// FIXME: This takes a long time. We could start it in parallel with the
	// first process run in a goroutine
	modchan := make(chan modd.Mod)
	err = modd.Watch(cnf.WatchPaths(), lullTime, modchan)
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}

	for mod := range modchan {
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

			err = modd.RunPreps(b, lmod, log)
			if err != nil {
				if *beep {
					fmt.Print("\a")
				}
				continue
			}
			daemonPens[i].Restart()
		}
	}
}
