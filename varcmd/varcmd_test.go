package varcmd

import (
	"reflect"
	"testing"
)

var varsTests = []struct {
	in  string
	out []string
}{
	{"@foo", []string{"@foo"}},
	{"foo", nil},
	{"@foo bar @voing", []string{"@foo", "@voing"}},
}

func TestVars(t *testing.T) {
	for _, tt := range varsTests {
		ret := Vars(tt.in)
		if !reflect.DeepEqual(ret, tt.out) {
			t.Errorf("expected %#v, got %#v", tt.out, ret)
		}
		for _, v := range tt.out {
			if !HasVar(tt.in, v) {
				t.Errorf("Expected to have %q", v)
			}
		}
		if HasVar(tt.in, "nonexistent") {
			t.Errorf("Expected not to have nonexistent")
		}
	}
}

var renderTests = []struct {
	in   string
	out  string
	vars map[string]string
}{
	{"@foo", "bar", map[string]string{"@foo": "bar"}},
	{"@foo@foo", "barbar", map[string]string{"@foo": "bar"}},
	{"@foo@bar", "barvoing", map[string]string{"@foo": "bar", "@bar": "voing"}},
}

func TestRender(t *testing.T) {
	for _, tt := range renderTests {
		ret, err := Render(tt.in, tt.vars)
		if err != nil {
			t.Error("Unexpected error")
		}
		if ret != tt.out {
			t.Errorf("expected %q, got %q", tt.out, ret)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	_, err := Render("@nonexistent", map[string]string{})
	if err == nil {
		t.Error("Expected error")
	}
}
