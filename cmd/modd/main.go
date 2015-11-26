package main

import (
	"time"

	"github.com/cortesi/modd"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
)

const batchTime = time.Millisecond * 200

func main() {
	paths := kingpin.Arg(
		"path",
		"Paths to monitor for changes.",
	).Required().Strings()

	cmdstats := kingpin.Flag("cmdstats", "Show stats on command execution").
		Short('s').
		Default("false").
		Bool()

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
	err := modd.Watch(*paths, *excludes, batchTime, modchan)
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}
	modd.RunProcs(*prep, log)
	for mod := range modchan {
		if len(mod.Added) > 0 {
			log.SayAs("debug", "Added: %v\n", mod.Added)
		}
		if len(mod.Changed) > 0 {
			log.SayAs("debug", "Changed: %v\n", mod.Changed)
		}
		if len(mod.Deleted) > 0 {
			log.SayAs("debug", "Deleted: %v\n", mod.Deleted)
		}
		modd.RunProcs(*prep, log)
	}
}
