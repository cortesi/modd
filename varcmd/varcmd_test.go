package varcmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/utils"
	"github.com/cortesi/modd/watch"
)

var quotePathTests = []struct {
	path     string
	expected string
}{
	{`one`, `"one"`},
	{` one`, `" one"`},
	{`one `, `"one "`},
}

func TestQuotePath(t *testing.T) {
	for i, tst := range quotePathTests {
		result := quotePath(tst.path)
		if result != tst.expected {
			t.Errorf("Test %d: expected\n%q\ngot\n%q", i, tst.expected, result)
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
		b := conf.Block{}
		mod := watch.Mod{}
		vc := VarCmd{&b, &mod, tt.vars}
		ret, err := vc.Render(tt.in)
		if err != nil {
			t.Error("Unexpected error")
		}
		if ret != tt.out {
			t.Errorf("expected %q, got %q", tt.out, ret)
		}
	}
}

func TestVarCmd(t *testing.T) {
	defer utils.WithTempDir(t)()

	dst := filepath.Join(".", "tdir")
	err := os.MkdirAll(dst, 0777)
	if err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(dst, "tfile"), []byte("test"), 0777)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	b := conf.Block{}
	b.Include = []string{"tdir/**"}
	vc := VarCmd{&b, nil, map[string]string{}}
	ret, err := vc.Render("@mods @dirmods")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expect := `"./tdir/tfile" "./tdir"`
	if runtime.GOOS == "windows" {
		expect = `".\tdir\tfile" ".\tdir"`
	}
	if ret != expect {
		t.Errorf("Unexpected return: %#v", ret)
	}

	vc = VarCmd{
		&b,
		&watch.Mod{Changed: []string{"foo"}},
		map[string]string{},
	}
	ret, err = vc.Render("@mods @dirmods")
	if err != nil {
		t.Fatal("unexpected error")
	}
	expected := `"./foo" "./."`
	if runtime.GOOS == "windows" {
		expected = `".\foo" ".\."`
	}
	if ret != expected {
		t.Errorf("Unexpected return: %s", ret)
	}
}

func TestRenderErrors(t *testing.T) {
	b := conf.Block{}
	mod := watch.Mod{}
	vc := VarCmd{&b, &mod, map[string]string{}}
	_, err := vc.Render("@nonexistent")
	if err == nil {
		t.Error("Expected error")
	}
}
