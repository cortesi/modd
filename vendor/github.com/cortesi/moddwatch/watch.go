package moddwatch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cortesi/moddwatch/filter"
	"github.com/rjeczalik/notify"
)

// MaxLullWait is the maximum time to wait for a lull. This only kicks in if
// we've had a constant stream of modifications blocking us.
const MaxLullWait = time.Second * 8

// isUnder takes two absolute paths, and returns true if child is under parent.
func isUnder(parent string, child string) bool {
	parent = filepath.ToSlash(parent)
	child = filepath.ToSlash(child)
	off := strings.Index(child, parent)
	if off == 0 && (len(child) == len(parent) || child[len(parent)] == '/') {
		return true
	}
	return false
}

func normPaths(root string, abspaths []string) ([]string, error) {
	aroot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	ret := make([]string, len(abspaths))
	for i, p := range abspaths {
		norm, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		if isUnder(aroot, norm) {
			norm, err = filepath.Rel(aroot, norm)
			if err != nil {
				return nil, err
			}
		}
		ret[i] = norm
	}
	return ret, nil
}

// An existenceChecker checks the existence of a file
type existenceChecker interface {
	Check(p string) bool
}

type statExistenceChecker struct{}

func (sc statExistenceChecker) Check(p string) bool {
	fi, err := os.Stat(p)
	if err == nil && !fi.IsDir() {
		return true
	}
	return false
}

// Mod encapsulates a set of changes
type Mod struct {
	Changed []string
	Deleted []string
	Added   []string
}

func (mod Mod) String() string {
	return fmt.Sprintf(
		"Added: %v\nDeleted: %v\nChanged: %v",
		mod.Added, mod.Deleted, mod.Changed,
	)
}

// All returns a single list of all files changed or added - deleted files are
// not included.
func (mod Mod) All() []string {
	all := make(map[string]bool)
	for _, p := range mod.Changed {
		all[p] = true
	}
	for _, p := range mod.Added {
		all[p] = true
	}
	return _keys(all)
}

// Has checks if a given Mod includes a specified file
func (mod Mod) Has(p string) bool {
	for _, v := range mod.All() {
		if filepath.Clean(p) == filepath.Clean(v) {
			return true
		}
	}
	return false
}

// Empty checks if this mod set is empty
func (mod Mod) Empty() bool {
	if (len(mod.Changed) + len(mod.Deleted) + len(mod.Added)) > 0 {
		return false
	}
	return true
}

// Filter applies a filter, returning a new Mod structure
func (mod Mod) Filter(root string, includes []string, excludes []string) (*Mod, error) {
	changed, err := filter.Files(mod.Changed, includes, excludes)
	if err != nil {
		return nil, err
	}
	deleted, err := filter.Files(mod.Deleted, includes, excludes)
	if err != nil {
		return nil, err
	}
	added, err := filter.Files(mod.Added, includes, excludes)
	if err != nil {
		return nil, err
	}
	return &Mod{Changed: changed, Deleted: deleted, Added: added}, nil
}

func (mod *Mod) normPaths(root string) (*Mod, error) {
	changed, err := normPaths(root, mod.Changed)
	if err != nil {
		return nil, err
	}
	deleted, err := normPaths(root, mod.Deleted)
	if err != nil {
		return nil, err
	}
	added, err := normPaths(root, mod.Added)
	if err != nil {
		return nil, err
	}
	return &Mod{Changed: changed, Deleted: deleted, Added: added}, nil
}

func _keys(m map[string]bool) []string {
	if len(m) > 0 {
		keys := make([]string, len(m))
		i := 0
		for k := range m {
			keys[i] = k
			i++
		}
		sort.Strings(keys)
		return keys
	}
	return nil
}

type fset map[string]bool

func mkmod(exists existenceChecker, added fset, removed fset, changed fset, renamed fset) Mod {
	ret := Mod{}
	for k := range renamed {
		// If a file is moved from A to B, we'll get separate rename
		// events for both A and B. The only way to know if it was the
		// source or destination is to check if the file exists.
		if exists.Check(k) {
			added[k] = true
		} else {
			removed[k] = true
		}
	}
	for k := range added {
		if exists.Check(k) {
			// If a file exists, and has been both added and
			// changed, we just mark it as added
			delete(changed, k)
			delete(removed, k)
		} else {
			// If a file has been added, and now does not exist, we
			// strike it everywhere. This probably means the file is
			// transient - i.e. has been quickly added and removed, or
			// we've just not recieved a removal notification.
			delete(added, k)
			delete(removed, k)
			delete(changed, k)
		}
	}
	for k := range removed {
		if exists.Check(k) {
			delete(removed, k)
		} else {
			delete(added, k)
			delete(changed, k)
		}
	}
	ret.Added = _keys(added)
	ret.Changed = _keys(changed)
	ret.Deleted = _keys(removed)
	return ret

}

// This function batches events up, and emits just a list of paths for files
// considered changed. It applies some heuristics to deal with short-lived
// temporary files and unreliable filesystem events. There are all sorts of
// challenges here, that mean we can only do a mediocre job as it stands.
//
// - There's no cross-platform way to get the source as well as the destination
// for Rename events.
// - Events can arrive out of order - i.e. we can get a removal event first
// then a creation event for a transient file.
// - Events seem to be unreliable on some platforms - i.e. we might get a
// removal event but never see a creation event.
// - Events appear nonsensical on some platforms - i.e. we sometimes get a
// Create event as well as a Remove event when a pre-existing file is removed.
//
// In the face of all this, all we can do is layer on a set of heuristics to
// try to get intuitive results.
func batch(lullTime time.Duration, maxTime time.Duration, exists existenceChecker, ch chan notify.EventInfo) Mod {
	added := make(map[string]bool)
	removed := make(map[string]bool)
	changed := make(map[string]bool)
	renamed := make(map[string]bool)
	// Have we had a modification in the last lull
	hadLullMod := false
	for {
		select {
		case evt := <-ch:
			hadLullMod = true
			switch evt.Event() {
			case notify.Create:
				added[evt.Path()] = true
			case notify.Remove:
				removed[evt.Path()] = true
			case notify.Write:
				changed[evt.Path()] = true
			case notify.Rename:
				renamed[evt.Path()] = true
			}
		case <-time.After(lullTime):
			// Have we had a lull?
			if hadLullMod == false {
				return mkmod(exists, added, removed, changed, renamed)
			}
			hadLullMod = false
		case <-time.After(maxTime):
			return mkmod(exists, added, removed, changed, renamed)
		}
	}
}

// Watcher is a handle that allows a Watch to be terminated
type Watcher struct {
	evtch chan notify.EventInfo
}

// Stop watching
func (w *Watcher) Stop() {
	notify.Stop(w.evtch)
}

// Given a set of include patterns relative to a root, which directories do we
// need to monitor for changes?
func baseDirs(root string, includePatterns []string) []string {
	root = filepath.FromSlash(root)
	bases := make([]string, len(includePatterns))
	for i, v := range includePatterns {
		bdir, trailer := filter.SplitPattern(v)
		if !filepath.IsAbs(bdir) {
			bdir = filepath.Join(root, filepath.FromSlash(bdir))
		}
		if stat, err := os.Lstat(bdir); err != nil {
			continue
		} else {
			// A symlink, so we rebase the include patterns and the base directory
			if stat.Mode()&os.ModeSymlink != 0 {
				lnk, err := os.Readlink(bdir)
				if err != nil {
					continue
				}
				if filepath.IsAbs(lnk) {
					bdir = lnk
				} else {
					bdir = filepath.Join(bdir, lnk)
				}
				includePatterns[i] = bdir + "/" + trailer
			}
		}
		bases[i] = bdir
	}
	return bases
}

// Watch watches a set of include and exclude patterns relative to a given root.
// Mod structs representing discrete changesets are sent on the channel ch.
//
// Watch applies heuristics to cope with transient files and unreliable event
// notifications. Modifications are batched up until there is a a lull in the
// stream of changes of duration lullTime. This lets us represent processes that
// progressively affect multiple files, like rendering, as a single changeset.
//
// All paths emitted are slash-delimited and normalised. If a path lies under
// the specified root, it is converted to a path relative to the root, otherwise
// the returned path is absolute.
//
// Pattern syntax is as follows:
//   *              any sequence of non-path-separators
//   **             any sequence of characters, including path separators
//   ?              any single non-path-separator character
//   [class]        any single non-path-separator character against a class
//                  of characters (see below)
//   {alt1,...}     a sequence of characters if one of the comma-separated
//                  alternatives matches
//
//  Any character with a special meaning can be escaped with a backslash (\).
//
// Character classes support the following:
// 		[abc]		any single character within the set
// 		[a-z]		any single character in the range
// 		[^class] 	any single character which does not match the class
func Watch(
	root string,
	includes []string,
	excludes []string,
	lullTime time.Duration,
	ch chan *Mod,
) (*Watcher, error) {
	evtch := make(chan notify.EventInfo, 4096)
	paths := baseDirs(root, includes)
	for _, p := range paths {
		err := notify.Watch(filepath.Join(p, "..."), evtch, notify.All)
		if err != nil {
			notify.Stop(evtch)
			return nil, err
		}
	}
	go func() {
		for {
			b := batch(lullTime, MaxLullWait, statExistenceChecker{}, evtch)
			if !b.Empty() {
				b, err := b.normPaths(root)
				if err != nil {
					// FIXME: Do something more decisive
					continue
				}
				b, err = b.Filter(root, includes, excludes)
				if err != nil {
					// FIXME: Do something more decisive
					continue
				}
				if !b.Empty() {
					ch <- b
				}
			}
		}
	}()
	return &Watcher{evtch}, nil
}

// List all files under the root that match the specified patterns. The file
// list returned is a catalogue of all files currently on disk that could occur
// in a Mod structure for a corresponding watch.
//
// All paths returned are slash-delimited and normalised. If a path lies under
// the specified root, it is converted to a path relative to the root, otherwise
// the returned path is absolute.
//
// The pattern syntax is the same as Watch.
func List(root string, includePatterns []string, excludePatterns []string) ([]string, error) {
	root = filepath.FromSlash(root)
	bases := baseDirs(root, includePatterns)
	ret := []string{}
	for _, b := range bases {
		err := filepath.Walk(
			b,
			func(p string, fi os.FileInfo, err error) error {
				if err != nil || fi.Mode()&os.ModeSymlink != 0 {
					return nil
				}
				cleanpath, err := filter.File(p, includePatterns, excludePatterns)
				if err != nil {
					return nil
				}
				if fi.IsDir() {
					m, err := filter.MatchAny(p, excludePatterns)
					// We skip the dir only if it's explicitly excluded
					if err != nil && !m {
						return filepath.SkipDir
					}
				} else if cleanpath != "" {
					ret = append(ret, cleanpath)
				}
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return normPaths(root, ret)
}
