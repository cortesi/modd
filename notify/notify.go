package notify

import (
	"os/exec"
	"runtime"
)

const prog = "modd"

// A Notifier notifies
type Notifier interface {
	Push(title string, content string, icon string)
}

// GrowlNotifier is a notifier for Growl
type GrowlNotifier struct {
}

func hasGrowlNotify() bool {
	_, err := exec.LookPath("growlnotify")
	if err != nil {
		return false
	}
	return true
}

// Push implements Notifier
func (GrowlNotifier) Push(title string, text string, iconPath string) {
	cmd := exec.Command(
		"growlnotify", "-n", prog, "-d", prog, "-m", text, prog,
	)
	go cmd.Run()
}

// NewNotifier finds a notifier for this platform
func NewNotifier() Notifier {
	if runtime.GOOS == "darwin" && hasGrowlNotify() {
		return &GrowlNotifier{}
	}
	return nil
}
