package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/watch"
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

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(modd.Version)
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

	if *prep {
		err := modd.PrepOnly(log, cnf, notifiers)
		if err != nil {
			log.Shout("%s", err)
		}
	} else {
		for {
			cnf.CommonExcludes(modd.CommonExcludes)
			cnf, err = modd.Run(log, cnf, watchfile, notifiers)
			if err != nil {
				log.Shout("%s", err)
				break
			}
			if cnf == nil {
				break
			}
		}
	}
}
