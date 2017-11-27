package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
)

const modfile = "./modd.conf"

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

var pipesignals = new(bool)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(modd.Version)

	if runtime.GOOS == "windows" {
		pipesignals = kingpin.Flag(
			"pipesignals",
			"For signals that don't exist on Windows, write their name to the stdin of daemons instead").
			// TODO(DH): Should it be enabled by default?
			Default("false").
			Bool()
	}

	kingpin.Parse()

	if *ignores {
		for _, patt := range modd.CommonExcludes {
			fmt.Println(patt)
		}
		os.Exit(0)
	}

	log := termlog.NewLog()
	if *debug {
		log.Enable("debug")
	}

	notifiers := []notify.Notifier{}
	if *doNotify {
		n := notify.PlatformNotifier()
		if n == nil {
			log.Shout("Could not find a desktop notifier")
		} else {
			notifiers = append(notifiers, n)
		}
	}
	if *beep {
		notifiers = append(notifiers, &notify.BeepNotifier{})
	}

	mr, err := modd.NewModRunner(*file, log, notifiers, !(*noconf), *pipesignals)
	if err != nil {
		log.Shout("%s", err)
		return
	}

	if *prep {
		err := mr.PrepOnly(true)
		if err != nil {
			log.Shout("%s", err)
		}
	} else {
		err = mr.Run()
		if err != nil {
			log.Shout("%s", err)
		}
	}
}
