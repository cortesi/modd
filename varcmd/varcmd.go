package varcmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/moddwatch"
	"github.com/cortesi/moddwatch/filter"
)

var name = regexp.MustCompile(`@\w+`)

func getDirs(paths []string) []string {
	m := map[string]bool{}
	for _, p := range paths {
		p := filepath.Dir(p)
		m[p] = true
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// quotePath quotes a path for use on the command-line. The path must be in
// slash-delimited format, and the quoted path will use the native OS separator.
func quotePath(path string) string {
	path = strings.Replace(path, "\"", "\\\"", -1)
	return "\"" + path + "\""
}

// mkArgs prepares a list of paths for the command line
func mkArgs(paths []string) string {
	escaped := make([]string, len(paths))
	for i, s := range paths {
		// FIXME: We'll need to find a more portable way for Windows
		escaped[i] = quotePath(
			"./" + s,
		)
	}
	return strings.Join(escaped, " ")
}

// VarCmd represents a set of variables for a specific block and mod set. It
// should be re-created anew each time the block is executed.
type VarCmd struct {
	Block *conf.Block
	Mod   *moddwatch.Mod
	Vars  map[string]string
}

// Get a variable by name
func (v *VarCmd) get(name string) (string, error) {
	if val, ok := v.Vars[name]; ok {
		return val, nil
	}
	if (name == "@mods" || name == "@dirmods") && v.Block != nil {
		var modified []string
		if v.Mod == nil {
			var err error
			// FIXME: this is a bug - it doesn't cope with absolute root paths
			modified, err = filter.Find(".", v.Block.Include, v.Block.Exclude)
			if err != nil {
				return "", err
			}
		} else {
			modified = v.Mod.All()
		}
		v.Vars["@mods"] = mkArgs(modified)
		v.Vars["@dirmods"] = mkArgs(getDirs(modified))
		return v.Vars[name], nil
	}
	return "", fmt.Errorf("No such variable: %s", name)
}

// Render renders the command with a map of variables
func (v *VarCmd) Render(cmd string) (string, error) {
	var err error
	cmd = string(
		name.ReplaceAllFunc(
			[]byte(cmd),
			func(key []byte) []byte {
				ks := string(key)
				val, errv := v.get(ks)
				if errv != nil {
					err = fmt.Errorf("No such variable: %s", ks)
					return nil
				}
				return []byte(val)
			},
		),
	)
	if err != nil {
		return "", err
	}
	return cmd, nil
}
