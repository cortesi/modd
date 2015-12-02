package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/cortesi/modd"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
)

const lullTime = time.Millisecond * 300

func main() {
	paths := kingpin.Arg(
		"path",
		"Paths to monitor for changes.",
	).Required().Strings()

	cmdstats := kingpin.Flag("cmdstats", "Show stats on command execution").
		Short('s').
		Default("false").
		Bool()

	daemons := kingpin.Flag("daemon", "Daemon to keep running").
		PlaceHolder("CMD").
		Short('d').
		Strings()

	excludes := kingpin.Flag("exclude", "Glob pattern for files to exclude from monitoring").
		PlaceHolder("PATTERN").
		Short('x').
		Strings()

	prep := kingpin.Flag("prep", "Prep command to run before daemons are restarted").
		PlaceHolder("CMD").
		Short('p').
		Strings()

	debug := kingpin.Flag("debug", "Debugging for devd development").
		Default("false").
		Bool()

	kingpin.Version(modd.Version)
	kingpin.Parse()
	log := termlog.NewLog()
	log.Notice("modd v%s", modd.Version)

	if *debug {
		log.Enable("debug")
		modd.Logger = log
	}
	if *cmdstats {
		log.Enable("cmdstats")
	}

	modchan := make(chan modd.Mod)
	err := modd.Watch(*paths, *excludes, lullTime, modchan)
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}
	err = modd.RunProcs(*prep, log)
	if err != nil {
		log.Shout("%s", err)
	}
	d := modd.DaemonPen{}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		d.Shutdown(<-c)
		os.Exit(0)
	}()
	d.Start(*daemons, log)
	for mod := range modchan {
		log.SayAs("debug", "Delta: \n%s", mod.String())
		err := modd.RunProcs(*prep, log)
		if err != nil {
			log.Shout("%s", err)
		}
		d.Restart()
	}
}
