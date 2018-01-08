package modd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/utils"
	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
)

const timeout = 2 * time.Second

func touch(p string) {
	p = filepath.FromSlash(p)
	d := filepath.Dir(p)
	err := os.MkdirAll(d, 0777)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write([]byte("teststring")); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	ioutil.ReadFile(p)
}

func events(p string) []string {
	parts := []string{}
	for _, p := range strings.Split(p, "\n") {
		if strings.HasPrefix(p, ":") {
			p = strings.TrimSpace(p)
			if !strings.HasSuffix(p, ":") {
				parts = append(parts, strings.TrimSpace(p))
			}
		}
	}
	return parts
}

func _testWatch(t *testing.T, modfunc func(), trigger string, expected []string) {
	defer utils.WithTempDir(t)()

	err := os.MkdirAll("a/inner", 0777)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll("b", 0777)
	if err != nil {
		t.Fatal(err)
	}

	touch("a/initial")
	// There's some race condition in rjeczalik/notify. If we don't wait a bit
	// here, we sometimes receive notifications for the change above even
	// though we haven't started the watcher.
	time.Sleep(200 * time.Millisecond)

	confTxt := `
        ** {
            prep +onchange: echo ":skipit:" @mods
            prep: echo ":all:" @mods
        }
        a/* {
            prep: echo ":a:" @mods
        }
        b/* {
            prep: echo ":b:" @mods
        }
        a/**/*.xxx {
            prep: echo ":c:" @mods
        }
        a/direct {
            prep: echo ":d:" @mods
        }
        direct {
            prep: echo ":e:" @mods
        }
    `

	if runtime.GOOS == "windows" {
		// Welcome to the wonderful world of Windows where
		//     echo "foo" "bar"
		// returns
		//     foo
		//     bar
		// instead of the following:
		//     foo bar
		//
		// To make our tests work we add <mods></mods> marker here and
		// strip them later when replacing newlines in between with spaces.
		confTxt = strings.Replace(confTxt, `" @mods`, `<mods>" @mods "</mods>"`, -1)
	}

	cnf, err := conf.Parse("test", confTxt)
	if err != nil {
		t.Fatal(err)
	}

	lt := termlog.NewLogTest()

	modchan := make(chan *moddwatch.Mod, 1024)
	cback := func() {
		start := time.Now()
		modfunc()
		for {
			if strings.Contains(lt.String(), trigger) {
				break
			}
			if time.Now().Sub(start) > timeout {
				fmt.Println("Timeout!")
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		modchan <- nil
	}

	mr := ModRunner{
		Log:    lt.Log,
		Config: cnf,
	}

	err = mr.runOnChan(modchan, cback)
	if err != nil {
		t.Fatalf("runOnChan: %s", err)
	}

	retStr := lt.String()

	if runtime.GOOS == "windows" {
		// see above for why we do this
		strip_nl := func(s string) string {
			s = s[6 : len(s)-7]
			return strings.Replace(s, "\n", " ", -1)
		}
		retStr = regexp.MustCompile(`<mods>([\s\S]*?)</mods>`).ReplaceAllStringFunc(retStr, strip_nl)
	}

	ret := events(retStr)

	if !reflect.DeepEqual(ret, expected) {
		t.Errorf("Expected\n%#v\nGot\n%#v", expected, ret)
	}
}

func TestWatch(t *testing.T) {
	_testWatch(
		t,
		func() { touch("a/touched") },
		"touched",
		[]string{
			":all: ./a/initial",
			":a: ./a/initial",
			":skipit: ./a/touched",
			":all: ./a/touched",
			":a: ./a/touched",
		},
	)
	_testWatch(
		t,
		func() {
			touch("a/touched")
			touch("b/touched")
		},
		"touched",
		[]string{
			":all: ./a/initial",
			":a: ./a/initial",
			":skipit: ./a/touched ./b/touched",
			":all: ./a/touched ./b/touched",
			":a: ./a/touched",
			":b: ./b/touched",
		},
	)
	_testWatch(
		t,
		func() {
			touch("a/inner/touched.xxx")
		},
		"touched",
		[]string{
			":all: ./a/initial",
			":a: ./a/initial",
			":skipit: ./a/inner/touched.xxx",
			":all: ./a/inner/touched.xxx",
			":c: ./a/inner/touched.xxx",
		},
	)
	_testWatch(
		t,
		func() {
			touch("a/direct")
		},
		"touched",
		[]string{
			":all: ./a/initial",
			":a: ./a/initial",
			":skipit: ./a/direct",
			":all: ./a/direct",
			":a: ./a/direct",
			":d: ./a/direct",
		},
	)
	_testWatch(
		t,
		func() {
			touch("direct")
		},
		"touched",
		[]string{
			":all: ./a/initial",
			":a: ./a/initial",
			":skipit: ./direct",
			":all: ./direct",
			":e: ./direct",
		},
	)
}
