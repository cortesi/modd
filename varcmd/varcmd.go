package varcmd

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/moddwatch"
)

var name = regexp.MustCompile(`(\\*)@\w+`)

func getDirs(paths []string) []string {
	m := map[string]bool{}
	for _, p := range paths {
		p := path.Dir(p)
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
// FIXME: This is actually dependent on the shell used.
func quotePath(path string) string {
	path = strings.Replace(path, "\"", "\\\"", -1)
	return "\"" + path + "\""
}

// The paths we receive from Go's path manipulation functions are "cleaned",
// which removes redundancy, but also removes the leading "./" needed by many
// command-line tools. This function turns cleaned paths into "really relative"
// paths.
func realRel(p string) string {
	// They should already be clean, but let's make sure.
	p = path.Clean(p)
	if path.IsAbs(p) {
		return p
	} else if p == "." {
		return "./"
	}
	return "./" + p
}

// mkArgs prepares a list of paths for the command line
func mkArgs(paths []string) string {
	escaped := make([]string, len(paths))
	for i, s := range paths {
		escaped[i] = quotePath(realRel(s))
	}
	return strings.Join(escaped, " ")
}

// VarCmd represents a set of variables for a specific block and mod set. It
// should be re-created anew each time the block is executed.
type VarCmd struct {
	Block    *conf.Block
	Modified []string
	Vars     map[string]string
}

// Get a variable by name
func (v *VarCmd) get(name string) (string, error) {
	if val, ok := v.Vars[name]; ok {
		return val, nil
	}
	if (name == "@mods" || name == "@dirmods") && v.Block != nil {
		var modified []string
		if v.Modified == nil {
			var err error
			modified, err = moddwatch.List(".", v.Block.Include, v.Block.Exclude)
			if err != nil {
				return "", err
			}
		} else {
			modified = v.Modified
		}
		v.Vars["@mods"] = mkArgs(modified)
		v.Vars["@dirmods"] = mkArgs(getDirs(modified))
		return v.Vars[name], nil
	}
	return "", fmt.Errorf("No such variable: %s", name)
}

const esc = '\\'

// Render renders the command with a map of variables
func (v *VarCmd) Render(cmd string) (string, error) {
	var err error
	cmd = string(
		name.ReplaceAllFunc(
			[]byte(cmd),
			func(key []byte) []byte {
				cnt := 0
				for _, c := range key {
					if c != esc {
						break
					}
					cnt++
				}
				ks := strings.TrimLeft(string(key), string(esc))
				if cnt%2 != 0 {
					return []byte(strings.Repeat(string(esc), (cnt-1)/2) + ks)
				}
				val, errv := v.get(ks)
				if errv != nil {
					err = fmt.Errorf("No such variable: %s", ks)
					return nil
				}
				val = strings.Repeat(string(esc), cnt/2) + val
				return []byte(val)
			},
		),
	)
	if err != nil {
		return "", err
	}
	return cmd, nil
}
