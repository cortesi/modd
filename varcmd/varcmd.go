package varcmd

import (
	"fmt"
	"regexp"
)

var name = regexp.MustCompile(`@\w+`)

// Vars returns a list of all variables in the string
func Vars(cmd string) []string {
	return name.FindAllString(cmd, -1)
}

// HasVar checks if the command has a given variable
func HasVar(cmd string, name string) bool {
	for _, v := range Vars(cmd) {
		if v == name {
			return true
		}
	}
	return false
}

// Render renders the command with a map of variables
func Render(cmd string, vars map[string]string) (string, error) {
	var err error
	cmd = string(
		name.ReplaceAllFunc(
			[]byte(cmd),
			func(key []byte) []byte {
				ks := string(key)
				val, ok := vars[ks]
				if !ok {
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
