package modd

import (
	"bufio"
	"io"
	"strings"
	"sync"

	"github.com/cortesi/termlog"
)

// shortCommand shortens a command to a name we can use in a notification
// header.
func shortCommand(command string) string {
	ret := command
	parts := strings.Split(command, "\n")
	for _, i := range parts {
		i = strings.TrimLeft(i, " \t#")
		i = strings.TrimRight(i, " \t\\")
		if i != "" {
			ret = i
			break
		}
	}
	return ret
}

// niceHeader tries to produce a nicer process name. We condense whitespace to
// make commands split over multiple lines with indentation more legible, and
// limit the line length to 80 characters.
func niceHeader(preamble string, command string) string {
	pre := termlog.DefaultPalette.Timestamp.SprintFunc()(preamble)
	command = termlog.DefaultPalette.Header.SprintFunc()(shortCommand(command))
	return pre + command
}

func logOutput(wg *sync.WaitGroup, fp io.ReadCloser, out func(string, ...interface{})) {
	defer wg.Done()
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out("%s", string(line))
	}
}
