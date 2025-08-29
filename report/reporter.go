package report

import (
	"fmt"
	"os"
	"time"
)

// Reporter publishes status updates.
type Reporter interface {
	Report(timestamp time.Time, status int)
}

// FileReporter writes status updates to a file.
type FileReporter struct {
	Filename string
}

// Report implements the Reporter interface.
func (fs FileReporter) Report(timestamp time.Time, status int) {
	ts := time.Now().UTC().Format(time.RFC3339)
	msg := fmt.Sprintf("%s,%d\n", ts, status)

	// Open for write, create if missing, truncate to zero length.
	f, err := os.OpenFile(fs.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.Write([]byte(msg)); err != nil {
		return
	}

	if err := f.Sync(); err != nil {
		return
	}
}
