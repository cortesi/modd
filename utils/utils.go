package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WithTempDir creates a temp directory, changes the current working directory
// to it, and returns a function that can be called to clean up. Use it like
// this:
//
//	defer WithTempDir(t)()
func WithTempDir(t *testing.T) func() {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	tmpdir, err := os.MkdirTemp("", "")
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

// GetRealWd returns the current working directory with the path
// segments changed to the actual case on disk. Sticks to the
// logic of os.Getwd, without resolving symlinks.
func GetRealWd() (string, error) {
	indir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Initial stat to check if the os considers the file to exist.
	// INFO: If we pass this point, the os considers this a path to an existing
	// file. If we now find case discrepancies, we can assume that the
	// filesystem is case-insensitive.
	_, err = os.Stat(indir)
	if err != nil {
		return "", err
	}

	sepString := string(filepath.Separator)
	prefix := filepath.VolumeName(indir) // empty on unix
	relativePath := indir[len(prefix)+len(sepString):]

	// Split the relative path into segments
	segments := strings.Split(relativePath, sepString)
	realPath := prefix + sepString

	// Validate each segment
Seg:
	for _, segment := range segments {
		currentPath := filepath.Join(realPath, segment)
		parentDir := filepath.Dir(currentPath)
		if parentDir == "." {
			parentDir = ""
		}

		entries, err := os.ReadDir(parentDir)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), segment) {
				realPath = filepath.Join(realPath, entry.Name())
				continue Seg
			}
		}

		return "", os.ErrNotExist
	}

	return realPath, nil
}
