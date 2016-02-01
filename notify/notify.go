package notify

import "os/exec"

const prog = "modd"

func hasExecutable(name string) bool {
	_, err := exec.LookPath("growlnotify")
	if err != nil {
		return false
	}
	return true
}

// A Notifier notifies
type Notifier interface {
	Push(title string, content string, icon string)
}

// GrowlNotifier is a notifier for Growl
type GrowlNotifier struct {
}

// Push implements Notifier
func (GrowlNotifier) Push(title string, text string, iconPath string) {
	cmd := exec.Command(
		"growlnotify", "-n", prog, "-d", prog, "-m", text, prog,
	)
	go cmd.Run()
}

// LibnotifyNotifier is a notifier for lib-notify
type LibnotifyNotifier struct {
}

// Push implements Notifier
func (LibnotifyNotifier) Push(title string, text string, iconPath string) {
	cmd := exec.Command(
		"notify-send", prog, text,
	)
	go cmd.Run()
}

// NewNotifier finds a notifier for this platform
func NewNotifier() Notifier {
	if hasExecutable("growlnotify") {
		return &GrowlNotifier{}
	} else if hasExecutable("notify-send") {
		return &LibnotifyNotifier{}
	}
	return nil
}
