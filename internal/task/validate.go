package task

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Validate scans the selected Markdown files. An empty path selects the nearest
// existing .shemiq directory; it never creates a project. Operational failures
// are returned separately from metadata issues.
func Validate(cwd, path string, fix bool) ([]Issue, error) {
	paths, err := selectMarkdown(cwd, path)
	if err != nil {
		return nil, err
	}
	docs, err := readDocuments(paths)
	if err != nil {
		return nil, err
	}
	var repaired []Issue
	if fix {
		for _, doc := range docs {
			fixed, err := repairMissingUUIDs(doc)
			if err != nil {
				return nil, err
			}
			repaired = append(repaired, fixed...)
		}
		// Recheck from disk: all unresolved findings describe the final state.
		docs, err = readDocuments(paths)
		if err != nil {
			return nil, err
		}
	}
	unresolved, err := validateDocuments(docs)
	if err != nil {
		return nil, err
	}
	return append(repaired, unresolved...), nil
}

func selectMarkdown(cwd, path string) ([]string, error) {
	if path == "" {
		var err error
		path, err = FindProjectDirectory(cwd)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no .shemiq directory found above %s", cwd)
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", path)
		}
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() || !isMarkdown(path) {
			return nil, fmt.Errorf("not a Markdown file: %s", path)
		}
		return []string{path}, nil
	}
	var paths []string
	err = filepath.WalkDir(path, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && isMarkdown(file) {
			paths = append(paths, file)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return paths, nil
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func readDocuments(paths []string) ([]document, error) {
	docs := make([]document, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		docs = append(docs, parseDocument(path, data))
	}
	return docs, nil
}

func repairMissingUUIDs(doc document) ([]Issue, error) {
	var edits []struct {
		offset int
		text   string
	}
	var fixed []Issue
	for _, directive := range doc.directives {
		if !directive.parseable {
			continue
		}
		if _, hasUUID := directive.fields["uuid"]; hasUUID {
			continue
		}
		if _, hasSource := directive.fields["source"]; hasSource {
			continue
		}
		id, err := NewUUID()
		if err != nil {
			return nil, err
		}
		edits = append(edits, struct {
			offset int
			text   string
		}{directive.insertAt, "uuid: " + id + directive.lineEnding})
		fixed = append(fixed, Issue{doc.path, directive.line, "missing uuid", true})
	}
	if len(edits) == 0 {
		return nil, nil
	}
	var output bytes.Buffer
	start := 0
	for _, edit := range edits {
		output.Write(doc.data[start:edit.offset])
		output.WriteString(edit.text)
		start = edit.offset
	}
	output.Write(doc.data[start:])
	// WriteFile retains the permissions of an existing file.
	if err := os.WriteFile(doc.path, output.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("repair %s: %w", doc.path, err)
	}
	return fixed, nil
}

func validateDocuments(docs []document) ([]Issue, error) {
	var issues []Issue
	seen := make(map[string]Issue)
	for _, doc := range docs {
		issues = append(issues, doc.issues...)
		for _, directive := range doc.directives {
			findings, err := validateDirective(doc.path, directive)
			if err != nil {
				return nil, err
			}
			issues = append(issues, findings...)
			if field, ok := directive.fields["uuid"]; ok && field.value != "" {
				if first, exists := seen[field.value]; exists {
					issues = append(issues, Issue{doc.path, field.line,
						fmt.Sprintf("duplicate uuid %s (first at %s:%d)", field.value, first.Path, first.Line), false})
				} else {
					seen[field.value] = Issue{Path: doc.path, Line: field.line}
				}
			}
		}
	}
	return issues, nil
}

func validateDirective(path string, directive directive) ([]Issue, error) {
	var issues []Issue
	fields := directive.fields
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return fields[keys[i]].line < fields[keys[j]].line })
	for _, key := range keys {
		field := fields[key]
		message := ""
		switch {
		case key != "type" && key != "parent" && key != "source" && key != "status" && key != "uuid":
			message = "unknown metadata field: " + key
		case field.value == "":
			message = "empty metadata field: " + key
		case key == "type" && field.value != "top-level" && field.value != "task":
			message = "invalid type: " + field.value
		case key == "status" && field.value != "new" && field.value != "progress" && field.value != "done":
			message = "invalid status: " + field.value
		case key == "uuid" && !uuidV4.MatchString(field.value):
			message = "invalid UUIDv4: " + field.value
		case key == "parent":
			target := filepath.Join(filepath.Dir(path), field.value)
			info, err := os.Stat(target)
			if os.IsNotExist(err) || err == nil && !info.Mode().IsRegular() {
				message = "parent is not an existing file: " + field.value
			} else if err != nil {
				return nil, fmt.Errorf("inspect parent %s: %w", target, err)
			}
		}
		if message != "" {
			issues = append(issues, Issue{path, field.line, message, false})
		}
	}
	_, source := fields["source"]
	uuid, hasUUID := fields["uuid"]
	if source && hasUUID {
		issues = append(issues, Issue{path, uuid.line, "source and uuid are mutually exclusive", false})
	} else if !source && !hasUUID {
		issues = append(issues, Issue{path, directive.line, "missing uuid", false})
	}
	return issues, nil
}
