package conf

import (
	"reflect"
	"testing"
)

func TestWatchPaths(t *testing.T) {
	c := Config{
		Blocks: []Block{
			{Include: []string{"a/foo", "a/bar"}},
			{Include: []string{"a/bar", "a/oink", "foo", "b/foo"}},
		},
	}
	expected := []string{"./..."}
	got := c.WatchPatterns()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %#v, got %#v", expected, got)
	}
}
