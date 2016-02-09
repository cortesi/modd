package utils

import (
	"io/ioutil"
	"os"
	"testing"
)

// WithTempDir creates a temp directory, changes the current working directory
// to it, and returns a function that can be called to clean up. Use it like
// this:
//      defer WithTempDir(t)()
func WithTempDir(t *testing.T) func() {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	err = os.Chdir(tmpdir)
	if err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	return func() {
		err := os.Chdir(cwd)
		if err != nil {
			t.Fatalf("Chdir: %v", err)
		}
		err = os.RemoveAll(tmpdir)
		if err != nil {
			t.Fatalf("Removing tmpdir: %s", err)
		}
	}
}
