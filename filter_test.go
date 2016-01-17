package modd

import (
	"reflect"
	"testing"
)

var filterFilesTests = []struct {
	includes []string
	excludes []string
	files    []string
	expected []string
	err      bool
}{
	{
		nil,
		[]string{"*"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{},
		false,
	},
	{
		nil,
		[]string{"*.go"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.cpp", "main.h", "bar.py"},
		false,
	},
	// Invalid patterns won't match anything. This would trigger a warning at
	// runtime.
	{
		nil,
		[]string{"[["},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		true,
	},

	{
		[]string{"main.*"},
		[]string{"*.cpp"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.go", "main.h"},
		false,
	},
	{
		nil, nil,
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		[]string{"main.cpp", "main.go", "main.h", "foo.go", "bar.py"},
		false,
	},
}

func TestFilterFiles(t *testing.T) {
	for i, tt := range filterFilesTests {
		result, err := filterFiles(tt.files, tt.includes, tt.excludes)
		if !tt.err && err != nil {
			t.Errorf("Test %d: error %s", i, err)
		}
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf(
				"Test %d (inc: %v, ex: %v), expected \"%v\" got \"%v\"",
				i, tt.includes, tt.excludes, tt.expected, result,
			)
		}
	}
}
