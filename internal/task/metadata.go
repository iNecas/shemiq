package task

import (
	"bytes"
	"regexp"
	"strings"
)

var metadataKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// Issue is a validation finding. Fixed is true only when --fix inserted a UUID.
type Issue struct {
	Path    string
	Line    int
	Message string
	Fixed   bool
}

type metadataField struct {
	value string
	line  int
}

type directive struct {
	line       int
	insertAt   int
	lineEnding string
	parseable  bool
	fields     map[string]metadataField
}

type document struct {
	path       string
	data       []byte
	directives []directive
	issues     []Issue
}

// parseDocument retains byte offsets so repairs never reformat surrounding text.
// Only column-zero openers are significant; indented Markdown examples are text.
func parseDocument(path string, data []byte) document {
	doc := document{path: path, data: data}
	var current *directive
	for start, line := 0, 1; start < len(data); line++ {
		end := len(data)
		if i := bytes.IndexByte(data[start:], '\n'); i >= 0 {
			end = start + i + 1
		}
		text := string(bytes.TrimSuffix(data[start:end], []byte("\n")))
		text = strings.TrimSuffix(text, "\r")
		switch {
		case text == ":::shemiq":
			if current != nil {
				current.parseable = false
				doc.issues = append(doc.issues, Issue{path, line, "nested shemiq directive", false})
			} else {
				current = &directive{line: line, fields: make(map[string]metadataField), parseable: true}
			}
		case strings.HasPrefix(text, ":::shemiq"):
			doc.issues = append(doc.issues, Issue{path, line, "malformed shemiq directive opener", false})
			if current != nil {
				current.parseable = false
			}
		case text == ":::" && current != nil:
			current.insertAt = start
			current.lineEnding = lineEnding(data, start)
			doc.directives = append(doc.directives, *current)
			current = nil
		case current != nil:
			key, value, ok := strings.Cut(text, ":")
			if !ok || !metadataKeyPattern.MatchString(key) {
				current.parseable = false
				doc.issues = append(doc.issues, Issue{path, line, "malformed metadata field", false})
				break
			}
			if _, exists := current.fields[key]; exists {
				doc.issues = append(doc.issues, Issue{path, line, "repeated metadata field: " + key, false})
			} else {
				current.fields[key] = metadataField{strings.TrimSpace(value), line}
			}
		}
		start = end
	}
	if current != nil {
		current.parseable = false
		doc.issues = append(doc.issues, Issue{path, current.line, "unterminated shemiq directive", false})
		doc.directives = append(doc.directives, *current)
	}
	return doc
}

func lineEnding(data []byte, before int) string {
	// Prefer the style immediately preceding the closing line, even for mixed files.
	if before > 0 && data[before-1] == '\n' {
		if before > 1 && data[before-2] == '\r' {
			return "\r\n"
		}
		return "\n"
	}
	if i := bytes.IndexByte(data, '\n'); i >= 1 && data[i-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}
