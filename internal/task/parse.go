package task

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var metadataKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// sourceSpan uses zero-based, half-open byte offsets into the original data.
// line is the one-based diagnostic line at start.
type sourceSpan struct {
	start int
	end   int
	line  int
}

type parsedDocument struct {
	path string
	data []byte // Unchanged source; callers must not mutate it.
	root *parsedSection
}

type parsedSection struct {
	level      int // The synthetic root has level zero and no heading.
	title      string
	heading    sourceSpan // The entire heading line, including its line ending.
	span       sourceSpan // Includes descendants, up to the next peer/ancestor.
	directives []*parsedDirective
	children   []*parsedSection
}

type parsedDirective struct {
	span       sourceSpan
	opening    sourceSpan // Entire opening and closing lines, including endings.
	closing    sourceSpan
	insertAt   int // Immediately before the closing line.
	lineEnding string
	fields     []parsedField // Every occurrence, in source order.
}

type parsedField struct {
	key       string
	value     string
	span      sourceSpan // Entire field line, including its line ending.
	keySpan   sourceSpan
	valueSpan sourceSpan // Trimmed value; may be zero-length.
}

// parseMarkdownDocument extracts structure only. It never reads files or
// interprets metadata, and returns no document on the first syntax error.
// Directives are permitted only immediately after the document's primary
// heading or direct ### subtask headings under its ## Tasks section.
func parseMarkdownDocument(
	path string, data []byte,
) (*parsedDocument, error) {
	root := &parsedSection{span: sourceSpan{0, len(data), 1}}
	doc := &parsedDocument{path: path, data: data, root: root}
	sections := []*parsedSection{root}
	var current *parsedDirective
	var fence markdownFence

	for start, line := 0, 1; start < len(data); line++ {
		text, span := markdownLine(data, start, line)
		start = span.end
		if current != nil {
			switch {
			case text == ":::shemiq":
				return nil, fmt.Errorf("%s:%d: nested shemiq directive", path, line)
			case strings.HasPrefix(text, ":::shemiq"):
				return nil, fmt.Errorf("%s:%d: malformed shemiq directive opener", path, line)
			case text == ":::":
				current.closing = span
				current.span.end = span.end
				current.insertAt = span.start
				current.lineEnding = lineEnding(data, span.start)
				current = nil
			default:
				field, err := parseMarkdownField(
					path, text, span)
				if err != nil {
					return nil, err
				}
				current.fields = append(current.fields, field)
			}
			continue
		}

		if fence.length != 0 {
			if fence.closes(text) {
				fence = markdownFence{}
			}
			continue
		}
		if opening, ok := markdownFenceOpening(text); ok {
			fence = opening
			continue
		}

		if strings.HasPrefix(text, ":::shemiq") {
			if text != ":::shemiq" {
				return nil, fmt.Errorf("%s:%d: malformed shemiq directive opener", path, line)
			}
			current = &parsedDirective{
				span: span, opening: span,
			}
			section := sections[len(sections)-1]
			section.directives = append(
				section.directives, current)
			continue
		}
		if level, title, ok := markdownHeading(text); ok {
			for sections[len(sections)-1].level >= level {
				sections[len(sections)-1].span.end = span.start
				sections = sections[:len(sections)-1]
			}
			section := &parsedSection{
				level: level, title: title, heading: span,
				span: sourceSpan{span.start, len(data), line},
			}
			parent := sections[len(sections)-1]
			parent.children = append(
				parent.children, section)
			sections = append(sections, section)
		}
	}
	if current != nil {
		return nil, fmt.Errorf(
			"%s:%d: unterminated shemiq directive",
			path, current.opening.line)
	}
	if err := validateDirectivePlacement(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// validateDirectivePlacement enforces that every directive
// immediately follows (only blank lines between) either the
// document's primary # heading or a direct ### child of its
// ## Tasks section. At most one directive per heading.
func validateDirectivePlacement(
	doc *parsedDocument,
) error {
	allowed := make(map[*parsedDirective]bool)
	var primary *parsedSection
	for _, child := range doc.root.children {
		if child.level == 1 {
			primary = child
			break
		}
	}
	if primary != nil {
		if d := adjacentDirective(doc, primary); d != nil {
			allowed[d] = true
		}
		for _, child := range primary.children {
			if child.level == 2 && child.title == "Tasks" {
				for _, entry := range child.children {
					if entry.level != 3 {
						continue
					}
					if d := adjacentDirective(
						doc, entry,
					); d != nil {
						allowed[d] = true
					}
				}
				break
			}
		}
	}
	for _, d := range allDirectives(doc.root) {
		if !allowed[d] {
			return fmt.Errorf(
				"%s:%d: unsupported directive position",
				doc.path, d.opening.line)
		}
	}
	return nil
}

func adjacentDirective(doc *parsedDocument, section *parsedSection) *parsedDirective {
	if len(section.directives) == 0 {
		return nil
	}
	directive := section.directives[0]
	between := doc.data[section.heading.end:directive.opening.start]
	if len(bytes.TrimSpace(between)) != 0 {
		return nil
	}
	return directive
}

func hasParsedField(directive *parsedDirective, key string) bool {
	for _, field := range directive.fields {
		if field.key == key {
			return true
		}
	}
	return false
}

func markdownLine(data []byte, start, line int) (string, sourceSpan) {
	end := len(data)
	if i := bytes.IndexByte(data[start:], '\n'); i >= 0 {
		end = start + i + 1
	}
	text := strings.TrimSuffix(string(data[start:end]), "\n")
	text = strings.TrimSuffix(text, "\r")
	return text, sourceSpan{start, end, line}
}

func parseMarkdownField(path, text string, span sourceSpan) (parsedField, error) {
	key, value, ok := strings.Cut(text, ":")
	if !ok || !metadataKeyPattern.MatchString(key) {
		return parsedField{}, fmt.Errorf("%s:%d: malformed metadata field", path, span.line)
	}
	trimmed := strings.TrimSpace(value)
	valueStart := span.start + len(key) + 1
	if trimmed != "" {
		valueStart += strings.Index(value, trimmed)
	}
	return parsedField{
		key: key, value: trimmed, span: span,
		keySpan:   sourceSpan{span.start, span.start + len(key), span.line},
		valueSpan: sourceSpan{valueStart, valueStart + len(trimmed), span.line},
	}, nil
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

func markdownHeading(text string) (int, string, bool) {
	level := 0
	for level < len(text) && text[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || (level < len(text) && text[level] != ' ' && text[level] != '\t') {
		return 0, "", false
	}
	title := strings.TrimSpace(text[level:])
	end := len(title)
	for end > 0 && title[end-1] == '#' {
		end--
	}
	if end < len(title) && (end == 0 || title[end-1] == ' ' || title[end-1] == '\t') {
		title = strings.TrimSpace(title[:end])
	}
	return level, title, true
}

type markdownFence struct {
	marker byte
	length int
}

func markdownFenceOpening(text string) (markdownFence, bool) {
	fence, tail := markdownFencePrefix(text)
	if fence.length == 0 || (fence.marker == '`' && strings.ContainsRune(tail, '`')) {
		return markdownFence{}, false
	}
	return fence, true
}

func (f markdownFence) closes(text string) bool {
	closing, tail := markdownFencePrefix(text)
	return closing.marker == f.marker && closing.length >= f.length && strings.Trim(tail, " \t") == ""
}

// markdownFencePrefix recognizes only runs of at least three matching markers
// with up to three leading spaces. The remaining text is an info string or,
// for a closer, must consist entirely of spaces/tabs.
func markdownFencePrefix(text string) (markdownFence, string) {
	start := 0
	for start < len(text) && text[start] == ' ' {
		start++
	}
	if start > 3 || start == len(text) || (text[start] != '`' && text[start] != '~') {
		return markdownFence{}, ""
	}
	end := start
	for end < len(text) && text[end] == text[start] {
		end++
	}
	if end-start < 3 {
		return markdownFence{}, ""
	}
	return markdownFence{text[start], end - start}, text[end:]
}
