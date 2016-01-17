package modd

import (
	"fmt"

	"github.com/bmatcuk/doublestar"
)

// Determine if a file should be included, based on the given exclude paths.
func shouldInclude(file string, includePatterns []string, excludePatterns []string) (bool, error) {
	include := false
	for _, pattern := range includePatterns {
		match, err := doublestar.Match(pattern, file)
		if err != nil {
			return false, fmt.Errorf("Error matching pattern '%s': %s", pattern, err)
		} else if match {
			include = true
			break
		}
	}
	if !include {
		return false, nil
	}
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

// Filter out the files that match the given patterns. At least ONE include
// pattern must match, and NONE of the exclude patterns must match.
func filterFiles(files, includePatterns []string, excludePatterns []string) ([]string, error) {
	ret := []string{}
	for _, file := range files {
		ok, err := shouldInclude(file, includePatterns, excludePatterns)
		if err != nil {
			return files, err
		}
		if ok {
			ret = append(ret, file)
		}
	}
	return ret, nil
}
