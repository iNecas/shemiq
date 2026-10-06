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

var knownMetadataFields = map[string]struct{}{
	"type":   {},
	"parent": {},
	"source": {},
	"status": {},
	"uuid":   {},
}

var validTypes = map[string]struct{}{
	"top-level": {},
	"task":      {},
}

var validStatuses = map[string]struct{}{
	"new":     {},
	"refined": {},
	"done":    {},
}

// Validate scans selected Markdown files and their existing local references.
// An empty path selects the nearest existing .shemiq directory; it never
// creates a project. Operational failures are separate from metadata issues.
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
		fixed, err := repairStatuses(docs)
		if err != nil {
			return nil, err
		}
		repaired = append(repaired, fixed...)
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

// readDocuments follows local references in both directions. A missing source is
// expected while a task is awaiting refinement; a missing parent is diagnosed
// by validateDirective. Each file is parsed only once, even for cyclic links.
func readDocuments(paths []string) ([]document, error) {
	seen := make(map[string]document)
	queue := append([]string(nil), paths...)
	for len(queue) > 0 {
		path := filepath.Clean(queue[0])
		queue = queue[1:]
		if _, loaded := seen[path]; loaded {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		doc := parseDocument(path, data)
		seen[path] = doc
		for _, directive := range doc.directives {
			for _, key := range []string{"source", "parent"} {
				field, ok := directive.fields[key]
				if !ok || field.value == "" {
					continue
				}
				target := referencePath(path, field.value)
				info, err := os.Stat(target)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("inspect %s %s: %w", key, target, err)
				}
				if info.Mode().IsRegular() {
					queue = append(queue, target)
				}
			}
		}
	}
	paths = make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	docs := make([]document, 0, len(paths))
	for _, path := range paths {
		docs = append(docs, seen[path])
	}
	return docs, nil
}

func referencePath(path, value string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(path), value))
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

// repairStatuses only promotes valid reciprocal task pairs. Apply all edits to
// each document together so several subtasks can be repaired in one pass.
func repairStatuses(docs []document) ([]Issue, error) {
	pairs, _ := linkedTaskPairs(docs)
	byPath := make(map[string][]textEdit)
	var fixed []Issue
	for _, pair := range pairs {
		parent, child := pair[0], pair[1]
		parentStatus, parentOK := statusRank(parent.directive)
		childStatus, childOK := statusRank(child.directive)
		if !parentOK || !childOK || parentStatus == childStatus {
			continue
		}
		lower, target := parent, childStatus
		if childStatus < parentStatus {
			lower, target = child, parentStatus
		}
		field, present := lower.directive.fields["status"]
		line := lower.directive.line
		edit := textEdit{start: lower.directive.insertAt, end: lower.directive.insertAt,
			text: "status: " + statusNames[target] + lower.directive.lineEnding}
		if present {
			line = field.line
			edit = textEdit{start: field.valueStart, end: field.valueEnd, text: statusNames[target]}
		}
		byPath[lower.path] = append(byPath[lower.path], edit)
		fixed = append(fixed, Issue{lower.path, line, "status promoted to " + statusNames[target], true})
	}
	for _, doc := range docs {
		if edits := byPath[doc.path]; len(edits) > 0 {
			if err := applyEdits(doc, edits); err != nil {
				return nil, err
			}
		}
	}
	return fixed, nil
}

var statusNames = []string{"new", "refined", "done"}

func statusRank(d directive) (int, bool) {
	status, err := directiveStatus(d)
	if err != nil {
		return 0, false
	}
	for rank, name := range statusNames {
		if status == name {
			return rank, true
		}
	}
	return 0, false
}

// Routing uses the same status rules as validation without running Validate:
// unrelated links or unfinished source files must not block a choice.
func requireDirectiveType(d directive, want string) error {
	if !d.parseable || d.repeated["type"] {
		return fmt.Errorf("malformed %s directive", want)
	}
	if d.fields["type"].value != want {
		return fmt.Errorf("expected type: %s", want)
	}
	return nil
}

func directiveStatus(d directive) (string, error) {
	if d.repeated["status"] {
		return "", fmt.Errorf("repeated status")
	}
	field, present := d.fields["status"]
	if !present {
		return "new", nil
	}
	if _, ok := validStatuses[field.value]; !ok {
		return "", fmt.Errorf("invalid status: %q", field.value)
	}
	return field.value, nil
}

type textEdit struct {
	start, end int
	text       string
}

func applyEdits(doc document, edits []textEdit) error {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var output bytes.Buffer
	start := 0
	for _, edit := range edits {
		output.Write(doc.data[start:edit.start])
		output.WriteString(edit.text)
		start = edit.end
	}
	output.Write(doc.data[start:])
	if err := os.WriteFile(doc.path, output.Bytes(), 0644); err != nil {
		return fmt.Errorf("repair %s: %w", doc.path, err)
	}
	return nil
}

// linkedTaskPairs reports broken reciprocal links and returns pairs eligible
// for status comparison. Parent list directives are the source of each pair.
func linkedTaskPairs(docs []document) ([][2]taskDirective, []Issue) {
	byPath := make(map[string]document, len(docs))
	for _, doc := range docs {
		byPath[doc.path] = doc
	}
	var pairs [][2]taskDirective
	var issues []Issue
	for _, doc := range docs {
		for _, d := range doc.directives {
			if !d.parseable || d.fields["type"].value != "task" {
				continue
			}
			// This
			if source, ok := d.fields["source"]; ok && source.value != "" {
				target := referencePath(doc.path, source.value)
				child, exists := byPath[target]
				if !exists {
					continue // A subtask file need not exist yet.
				}
				matches := matchingTaskDirectives(child, "parent", doc.path)
				if len(matches) != 1 || len(matchingTaskDirectives(doc, "source", child.path)) != 1 {
					issues = append(issues, Issue{doc.path, source.line,
						fmt.Sprintf("source %s does not have exactly one reciprocal task parent", source.value), false})
					continue
				}
				pairs = append(pairs, [2]taskDirective{{doc.path, d}, {child.path, matches[0]}})
			}
			if parent, ok := d.fields["parent"]; ok && parent.value != "" {
				target := referencePath(doc.path, parent.value)
				owner, exists := byPath[target]
				if !exists {
					continue // Missing parent is reported by validateDirective.
				}
				if len(matchingTaskDirectives(owner, "source", doc.path)) != 1 {
					issues = append(issues, Issue{doc.path, parent.line,
						fmt.Sprintf("parent %s does not have exactly one reciprocal task source", parent.value), false})
				}
			}
		}
	}
	return pairs, issues
}

type taskDirective struct {
	path      string
	directive directive
}

func matchingTaskDirectives(doc document, key, target string) []directive {
	var matches []directive
	for _, d := range doc.directives {
		field, ok := d.fields[key]
		if d.parseable && d.fields["type"].value == "task" && ok && field.value != "" && referencePath(doc.path, field.value) == target {
			matches = append(matches, d)
		}
	}
	return matches
}

func validateDocuments(docs []document) ([]Issue, error) {
	fields, err := validateFields(docs)
	if err != nil {
		return nil, err
	}
	return append(fields, validateLinks(docs)...), nil
}

// validateFields checks directive metadata and UUID uniqueness across loaded files.
func validateFields(docs []document) ([]Issue, error) {
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

// validateLinks checks reciprocal task references and linked status agreement.
func validateLinks(docs []document) []Issue {
	pairs, issues := linkedTaskPairs(docs)
	for _, pair := range pairs {
		parent, child := pair[0], pair[1]
		parentRank, parentOK := statusRank(parent.directive)
		childRank, childOK := statusRank(child.directive)
		if parentOK && childOK && parentRank != childRank {
			field, present := parent.directive.fields["status"]
			line := parent.directive.line
			if present {
				line = field.line
			}
			issues = append(issues, Issue{parent.path, line,
				fmt.Sprintf("status mismatch with %s: %s vs %s", child.path, statusNames[parentRank], statusNames[childRank]), false})
		}
	}
	return issues
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
		if _, known := knownMetadataFields[key]; !known {
			message = "unknown metadata field: " + key
		} else if field.value == "" {
			message = "empty metadata field: " + key
		} else {
			switch key {
			case "type":
				if _, valid := validTypes[field.value]; !valid {
					message = "invalid type: " + field.value
				}
			case "status":
				if _, valid := validStatuses[field.value]; !valid {
					message = "invalid status: " + field.value
				}
			case "uuid":
				if !uuidV4.MatchString(field.value) {
					message = "invalid UUIDv4: " + field.value
				}
			case "parent", "source":
				target := referencePath(path, field.value)
				info, err := os.Stat(target)
				if os.IsNotExist(err) {
					if key == "parent" {
						message = "parent is not an existing file: " + field.value
					}
				} else if err != nil {
					return nil, fmt.Errorf("inspect %s %s: %w", key, target, err)
				} else if !info.Mode().IsRegular() {
					message = key + " is not an existing file: " + field.value
				}
			}
		}
		if message != "" {
			issues = append(issues, Issue{path, field.line, message, false})
		}
	}
	_, hasSource := fields["source"]
	uuid, hasUUID := fields["uuid"]
	if hasSource && hasUUID {
		issues = append(issues, Issue{path, uuid.line, "source and uuid are mutually exclusive", false})
	} else if !hasSource && !hasUUID {
		issues = append(issues, Issue{path, directive.line, "missing uuid", false})
	}
	return issues, nil
}
