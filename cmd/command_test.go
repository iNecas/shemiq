package cmd

import (
	"bytes"
	"strings"
)

func runCommand(input string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := Execute(strings.NewReader(input), &stdout, &stderr, args)
	return stdout.String(), stderr.String(), err
}
