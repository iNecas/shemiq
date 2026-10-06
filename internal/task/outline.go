package task

import (
	"fmt"
	"strings"
)

type namedDirective struct {
	title     string
	directive directive
}

// parseTopLevelOutline pairs headings with adjacent directives (allowing blank
// lines). It does not follow links or validate unrelated document metadata.
func parseTopLevelOutline(path string, data []byte) (string, directive, []namedDirective, error) {
	doc := parseDocument(path, data)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	byLine := make(map[int]directive)
	for _, d := range doc.directives {
		byLine[d.line] = d
	}
	paired := func(line int) (directive, bool) {
		for line < len(lines) && strings.TrimSpace(lines[line]) == "" {
			line++
		}
		d, ok := byLine[line+1]
		return d, ok
	}
	var title string
	var top directive
	found := false
	inTasks := false
	var subs []namedDirective
	inDirective := false
	for i, line := range lines {
		if line == ":::shemiq" {
			inDirective = true
			continue
		}
		if inDirective {
			if line == ":::" {
				inDirective = false
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "# ") && !found:
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			top, found = paired(i + 1)
			if title == "" || !found {
				return "", directive{}, nil, fmt.Errorf("%s: missing top-level heading/directive pair", path)
			}
		case strings.HasPrefix(line, "# "):
			inTasks = false
		case strings.HasPrefix(line, "## "):
			inTasks = line == "## Tasks"
		case strings.HasPrefix(line, "### ") && inTasks:
			subtitle := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			if d, ok := paired(i + 1); ok {
				subs = append(subs, namedDirective{subtitle, d})
			}
		}
	}
	if !found {
		return "", directive{}, nil, fmt.Errorf("%s: missing top-level heading/directive pair", path)
	}
	return title, top, subs, nil
}
