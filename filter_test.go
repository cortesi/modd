package modd

import (
	"reflect"
	"testing"
)

var filterFilesTests = []struct {
	pattern  string
	files    []string
	expected []string
	err      bool
}{
	{
		"*",
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{},
		false,
	},
	{
		"*.go",
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.cpp", "main.h", "bar.py"},
		false,
	},
	// Invalid patterns won't match anything. This would trigger a warning at
	// runtime.
	{
		"[[",
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		true,
	},
}

func TestFilterFiles(t *testing.T) {
	for i, tt := range filterFilesTests {
		result, err := filterFiles(tt.files, []string{tt.pattern})
		if !tt.err && err != nil {
			t.Errorf("Test %d: error %s", i, err)
		}
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf(
				"Test %d (pattern %s), expected \"%v\" got \"%v\"",
				i, tt.pattern, tt.expected, result,
			)
		}
	}
}
