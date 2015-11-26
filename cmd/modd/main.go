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

	excludes := kingpin.Flag("exclude", "Glob pattern for files to exclude from livereload").
		PlaceHolder("PATTERN").
		Short('x').
		Strings()

	debug := kingpin.Flag("debug", "Debugging for devd development").
		Default("false").
		Bool()

	kingpin.Version(modd.Version)
	kingpin.Parse()
	log := termlog.NewLog()

	if *debug {
		log.Enable("debug")
		modd.Logger = log
	}

	modchan := make(chan modd.Mod)
	err := modd.Watch(*paths, *excludes, batchTime, modchan)
	if err != nil {
		kingpin.Fatalf("Fatal error: %s", err)
	}
	for mod := range modchan {
		if len(mod.Added) > 0 {
			log.Say("Added: %v\n", mod.Added)
		}
		if len(mod.Changed) > 0 {
			log.Say("Changed: %v\n", mod.Changed)
		}
		if len(mod.Deleted) > 0 {
			log.Say("Deleted: %v\n", mod.Deleted)
		}
	}
}
