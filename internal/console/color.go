package console

import (
	"os"

	"golang.org/x/term"
)

// ANSI escape sequences for status coloring.
const (
	ansiReset   = "\033[0m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiGreen   = "\033[32m"
)

// ColorEnabled reports whether ANSI color escapes should be used for the given
// output writer. It returns false for non-terminal writers, when NO_COLOR is
// set, or when TERM=dumb.
func ColorEnabled(output interface{}) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	type fder interface {
		Fd() uintptr
	}
	f, ok := output.(fder)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// ColorStatus wraps a status string in the appropriate ANSI color escape
// when color is enabled.
func ColorStatus(status string, color bool) string {
	if !color {
		return status
	}
	var code string
	switch status {
	case "new":
		code = ansiCyan
	case "refined":
		code = ansiYellow
	case "done":
		code = ansiGreen
	default:
		return status
	}
	return code + status + ansiReset
}
