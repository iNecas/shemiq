package utils

import "strings"

// Dedent removes the indentation of the first content line from each line.
// A closing backtick on its own line leaves a trailing newline in the result.
func Dedent(content string) string {
	lines := strings.Split(strings.TrimPrefix(content, "\n"), "\n")
	indent := lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " \t"))]
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, indent)
	}
	return strings.Join(lines, "\n")
}
