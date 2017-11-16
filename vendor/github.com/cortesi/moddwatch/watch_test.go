package moddwatch

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
)

// WithTempDir creates a temp directory, changes the current working directory
// to it, and returns a function that can be called to clean up. Use it like
// this:
//      defer WithTempDir(t)()
func WithTempDir(t *testing.T) func() {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	tmpdir, err := ioutil.TempDir("", "")
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

type TEventInfo struct {
	event notify.Event
	path  string
}

func (te TEventInfo) Path() string {
	return te.path
}

func (te TEventInfo) Event() notify.Event {
	return te.event
}

func (te TEventInfo) Sys() interface{} {
	return nil
}

type testExistenceChecker struct {
	paths map[string]bool
}

func (e *testExistenceChecker) Check(p string) bool {
	_, ok := e.paths[p]
	return ok
}

func exists(paths ...string) *testExistenceChecker {
	et := testExistenceChecker{make(map[string]bool)}
	for _, p := range paths {
		et.paths[p] = true
	}
	return &et
}

var batchTests = []struct {
	events   []TEventInfo
	exists   *testExistenceChecker
	expected Mod
}{
	{
		[]TEventInfo{
			TEventInfo{notify.Create, "foo"},
			TEventInfo{notify.Create, "bar"},
		},
		exists("bar", "foo"),
		Mod{Added: []string{"bar", "foo"}},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Rename, "foo"},
			TEventInfo{notify.Rename, "bar"},
		},
		exists("foo"),
		Mod{Added: []string{"foo"}, Deleted: []string{"bar"}},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Write, "foo"},
		},
		exists("foo"),
		Mod{Changed: []string{"foo"}},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Write, "foo"},
			TEventInfo{notify.Remove, "foo"},
		},
		exists(),
		Mod{Deleted: []string{"foo"}},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Remove, "foo"},
		},
		exists("foo"),
		Mod{},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Create, "foo"},
			TEventInfo{notify.Create, "bar"},
			TEventInfo{notify.Remove, "bar"},
		},
		exists("bar", "foo"),
		Mod{Added: []string{"bar", "foo"}},
	},
	{
		[]TEventInfo{
			TEventInfo{notify.Create, "foo"},
		},
		exists(),
		Mod{},
	},
}

func TestBatch(t *testing.T) {
	for i, tst := range batchTests {
		input := make(chan notify.EventInfo, len(tst.events))
		for _, e := range tst.events {
			input <- e
		}
		ret := batch(time.Millisecond*10, MaxLullWait, tst.exists, input)
		if !reflect.DeepEqual(ret, tst.expected) {
			t.Errorf("Test %d: expected\n%#v\ngot\n%#v", i, tst.expected, ret)
		}
	}
}

func abs(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		panic("Could not get current working directory")
	}
	return filepath.ToSlash(filepath.Join(wd, path))
}

var isUnderTests = []struct {
	parent   string
	child    string
	expected bool
}{
	{"/foo", "/foo/bar", true},
	{"/foo", "/foo", true},
	{"/foo", "/foobar/bar", false},
}

func TestIsUnder(t *testing.T) {
	for i, tst := range isUnderTests {
		ret := isUnder(tst.parent, tst.child)
		if ret != tst.expected {
			t.Errorf("Test %d: expected %#v, got %#v", i, tst.expected, ret)
		}
	}
}

func TestMod(t *testing.T) {
	if !(Mod{}.Empty()) {
		t.Error("Expected mod to be empty.")
	}
	m := Mod{
		Added:   []string{"add"},
		Deleted: []string{"rm"},
		Changed: []string{"change"},
	}
	if m.Empty() {
		t.Error("Expected mod not to be empty")
	}
	if !reflect.DeepEqual(m.All(), []string{"add", "change"}) {
		t.Error("Unexpeced return from Mod.All")
	}

	m = Mod{
		Added:   []string{abs("add")},
		Deleted: []string{abs("rm")},
		Changed: []string{abs("change")},
	}
	if _, err := m.normPaths("."); err != nil {
		t.Error(err)
	}
}

func testListBasic(t *testing.T) {
	var findTests = []struct {
		include  []string
		exclude  []string
		expected []string
	}{
		{
			[]string{"**"},
			[]string{},
			[]string{"a/a.test1", "a/b.test2", "b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**/*.test1"},
			[]string{},
			[]string{"a/a.test1", "b/a.test1", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"*.test1"},
			[]string{"a/a.test1", "a/b.test2", "b/a.test1", "b/b.test2", "x"},
		},
		{
			[]string{"**"},
			[]string{"a/**"},
			[]string{"b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"a/*"},
			[]string{"b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"**/*.test1", "**/*.test2"},
			[]string{"x"},
		},
	}

	defer WithTempDir(t)()
	paths := []string{
		"a/a.test1",
		"a/b.test2",
		"b/a.test1",
		"b/b.test2",
		"x",
		"x.test1",
	}
	for _, p := range paths {
		dst := filepath.Join(".", p)
		err := os.MkdirAll(filepath.Dir(dst), 0777)
		if err != nil {
			t.Fatalf("Error creating test dir: %v", err)
		}
		err = ioutil.WriteFile(dst, []byte("test"), 0777)
		if err != nil {
			t.Fatalf("Error writing test file: %v", err)
		}
	}

	for i, tt := range findTests {
		ret, err := List(".", tt.include, tt.exclude)
		if err != nil {
			t.Fatal(err)
		}
		expected := tt.expected
		for i := range ret {
			ret[i] = filepath.ToSlash(ret[i])
		}
		if !reflect.DeepEqual(ret, expected) {
			t.Errorf(
				"%d: %#v, %#v - Expected\n%#v\ngot:\n%#v",
				i, tt.include, tt.exclude, expected, ret,
			)
		}
	}
}

func testList(t *testing.T) {
	var findTests = []struct {
		include  []string
		exclude  []string
		expected []string
	}{
		{
			[]string{"**"},
			[]string{},
			[]string{"a/a.test1", "a/b.test2", "a/sub/c.test2", "b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**/*.test1"},
			[]string{},
			[]string{"a/a.test1", "b/a.test1", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"*.test1"},
			[]string{"a/a.test1", "a/b.test2", "a/sub/c.test2", "b/a.test1", "b/b.test2", "x"},
		},
		{
			[]string{"**"},
			[]string{"a/**"},
			[]string{"b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"a/**"},
			[]string{"b/a.test1", "b/b.test2", "x", "x.test1"},
		},
		{
			[]string{"**"},
			[]string{"**/*.test1", "**/*.test2"},
			[]string{"x"},
		},
		{
			[]string{"a/relsymlink"},
			[]string{},
			[]string{},
		},
		{
			[]string{"a/relfilesymlink"},
			[]string{},
			[]string{"x"},
		},
		{
			[]string{"a/relsymlink/**"},
			[]string{},
			[]string{"b/a.test1", "b/b.test2"},
		},
		{
			[]string{"a/**", "a/relsymlink/**"},
			[]string{},
			[]string{"a/a.test1", "a/b.test2", "a/sub/c.test2", "b/a.test1", "b/b.test2"},
		},
		{
			[]string{"a/abssymlink/**"},
			[]string{},
			[]string{"b/a.test1", "b/b.test2"},
		},
		{
			[]string{"a/**", "a/abssymlink/**"},
			[]string{},
			[]string{"a/a.test1", "a/b.test2", "a/sub/c.test2", "b/a.test1", "b/b.test2"},
		},
	}

	defer WithTempDir(t)()
	paths := []string{
		"a/a.test1",
		"a/b.test2",
		"a/sub/c.test2",
		"b/a.test1",
		"b/b.test2",
		"x",
		"x.test1",
	}
	for _, p := range paths {
		dst := filepath.Join(".", p)
		err := os.MkdirAll(filepath.Dir(dst), 0777)
		if err != nil {
			t.Fatalf("Error creating test dir: %v", err)
		}
		err = ioutil.WriteFile(dst, []byte("test"), 0777)
		if err != nil {
			t.Fatalf("Error writing test file: %v", err)
		}
	}
	if err := os.Symlink("../../b", "./a/relsymlink"); err != nil {
		t.Fatal(err)
		return
	}
	if err := os.Symlink("../../x", "./a/relfilesymlink"); err != nil {
		t.Fatal(err)
		return
	}

	sabs, err := filepath.Abs("./b")
	if err != nil {
		t.Fatal(err)
		return
	}
	if err = os.Symlink(sabs, "./a/abssymlink"); err != nil {
		t.Fatal(err)
		return
	}

	for i, tt := range findTests {
		t.Run(
			fmt.Sprintf("%.3d", i),
			func(t *testing.T) {
				ret, err := List(".", tt.include, tt.exclude)
				if err != nil {
					t.Fatal(err)
				}
				expected := tt.expected
				for i := range ret {
					if filepath.IsAbs(ret[i]) {
						wd, err := os.Getwd()
						rel, err := filepath.Rel(wd, filepath.ToSlash(ret[i]))
						if err != nil {
							t.Fatal(err)
							return
						}
						ret[i] = rel
					} else {
						ret[i] = filepath.ToSlash(ret[i])
					}
				}
				if !reflect.DeepEqual(ret, expected) {
					t.Errorf(
						"%d: %#v, %#v - Expected\n%#v\ngot:\n%#v",
						i, tt.include, tt.exclude, expected, ret,
					)
				}
			},
		)
	}
}

func TestList(t *testing.T) {
	testListBasic(t)
	if runtime.GOOS != "windows" {
		testList(t)
	}
}
