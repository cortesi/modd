package modd

import (
	"fmt"

	"github.com/bmatcuk/doublestar"
)

// Determine if a file should be included, based on the given exclude paths.
func shouldInclude(file string, excludePatterns []string) (bool, error) {
	for _, pattern := range excludePatterns {
		match, err := doublestar.Match(pattern, file)
		if err != nil {
			return false, fmt.Errorf("Error matching pattern '%s': %s", pattern, err)
		} else if match {
			return false, nil
		}
	}
	return true, nil
}

// Filter out the files that match the given exclude patterns.
func filterFiles(files, excludePatterns []string) ([]string, error) {
	ret := []string{}
	for _, file := range files {
		ok, err := shouldInclude(file, excludePatterns)
		if err != nil {
			return files, err
		}
		if ok {
			ret = append(ret, file)
		}
	}
	return ret, nil
}
