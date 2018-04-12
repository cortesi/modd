package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cortesi/modd"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/termlog"
	"gopkg.in/alecthomas/kingpin.v2"
	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
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

var exec = kingpin.Flag("exec", "Execute a command in the built-in shell").
	String()

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(modd.Version)
	kingpin.Parse()

	if *exec != "" {
		parser := syntax.NewParser()
		runner := interp.Runner{
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		prog, err := parser.Parse(strings.NewReader(*exec), "")
		if err != nil {
			os.Exit(1)
		}
		runner.Reset()
		err = runner.Run(prog)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

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

	mr, err := modd.NewModRunner(*file, log, notifiers, !(*noconf))
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
