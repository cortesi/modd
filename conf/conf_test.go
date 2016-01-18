package conf

import (
	"reflect"
	"testing"
)

var basePathTests = []struct {
	pattern  string
	expected string
}{
	{"foo", ""},
	{"test/foo", "test/"},
	{"test/foo*", "test/"},
	{"test/*.**", "test/"},
	{"**/*", ""},
	{"foo*/bar", ""},
	{"foo/**/bar", "foo/"},
}

func TestBasePath(t *testing.T) {
	for i, tt := range basePathTests {
		ret := basePath(tt.pattern)
		if ret != tt.expected {
			t.Errorf("%d: %q - Expected %q, got %q", i, tt.pattern, tt.expected, ret)
		}
	}
}

func TestWatchPaths(t *testing.T) {
	c := Config{
		[]Block{
			{Include: []string{"a/foo", "a/bar"}},
			{Include: []string{"a/bar", "a/oink", "foo", "b"}},
		},
	}
	if !reflect.DeepEqual(c.WatchPaths(), []string{"a/", "b", "."}) {
		t.Fail()
	}

}
