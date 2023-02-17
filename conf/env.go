package conf

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var validEnv = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_0-9]*$`)

func processEnvFile(envFile string) ([]string, error) {
	_, err := os.Stat(envFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(" not found")
		}
		return nil, err
	}
	f, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var envOut []string
	lineNo := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		p := strings.SplitN(line, `=`, 2)
		if !validEnv.MatchString(p[0]) {
			return nil, fmt.Errorf("%d: invalid environment variable %q", lineNo, p[0])
		}
		if len(p) == 2 && p[1] == "" {
			line = p[0]
		}
		envOut = append(envOut, line)
	}
	if s.Err() != nil {
		return nil, fmt.Errorf("scan error: %v", err)
	}
	return envOut, nil
}
