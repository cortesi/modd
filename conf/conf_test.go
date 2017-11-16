package conf

import (
	"reflect"
	"testing"
)

func TestWatchPaths(t *testing.T) {
	c := Config{
		Blocks: []Block{
			{Include: []string{"a/foo", "a/bar"}},
			{Include: []string{"a/bar", "b/foo"}},
		},
	}
	expected := []string{"a/bar", "a/foo", "b/foo"}
	got := c.IncludePatterns()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %#v, got %#v", expected, got)
	}
}
