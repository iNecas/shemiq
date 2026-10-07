package task

import "regexp"

var uuidV4 = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

var knownMetadataFields = map[string]struct{}{
	"type": {}, "parent": {}, "source": {}, "status": {}, "uuid": {},
}

var validTypes = map[string]struct{}{
	"top-level": {}, "task": {},
}

var validStatuses = map[string]struct{}{
	"new": {}, "refined": {}, "done": {},
}

type TaskType string

const (
	TypeTopLevel TaskType = "top-level"
	TypeTask     TaskType = "task"
)

// Issue is a located metadata finding. Fixed is true after a validation repair.
type Issue struct {
	Path    string
	Line    int
	Message string
	Fixed   bool
}

// Task represents one defining section. A standalone task's Path is its file;
// a parent-list entry's Path is its source target (which need not exist).
// Its private document reference always identifies the defining file, even for
// an entry whose Path points elsewhere. Referenced metadata is never substituted.
type Task struct {
	Type        TaskType
	Title       string
	Status      string
	Path        string
	ProjectRoot string
	Subtasks    []Task
	Issues      []Issue

	document  *parsedDocument
	section   *parsedSection
	directive *parsedDirective // heading-adjacent directive, nil if absent
	uuid      string           // interpreted UUID, empty if unusable
	sourceRef string           // resolved source path, empty if unusable
	parentRef string           // resolved parent path, empty if unusable
}

func sectionTask(
	doc *parsedDocument,
	section *parsedSection,
	parentList bool,
) Task {
	result := Task{
		Title:       section.title,
		ProjectRoot: documentProjectRoot(doc.path),
		document:    doc,
		section:     section,
	}
	if section.title == "" {
		result.Issues = append(result.Issues, Issue{
			Path:    doc.path,
			Line:    section.heading.line,
			Message: "empty task title",
		})
	}
	directive := adjacentDirective(doc, section)
	if directive == nil {
		// Directive-less subtask: valid with defaults.
		// Primary headings without directives keep an issue.
		if parentList {
			result.Type = TypeTask
			result.Status = "new"
		} else {
			result.Issues = append(result.Issues, Issue{
				Path: doc.path,
				Line: section.heading.line,
				Message: "missing heading-adjacent " +
					"task directive",
			})
		}
		if !parentList {
			result.Path = doc.path
		}
		return result
	}
	result.directive = directive
	fields, issues := directiveFields(doc.path, directive)
	result.Type = TaskType(fields["type"].value)
	result.Status = fields["status"].value
	if !hasParsedField(directive, "status") {
		result.Status = "new"
	}
	result.uuid = fields["uuid"].value
	if source, ok := fields["source"]; ok {
		result.sourceRef = resolvedReferencePath(
			doc.path, source.value)
	}
	if parent, ok := fields["parent"]; ok {
		result.parentRef = resolvedReferencePath(
			doc.path, parent.value)
	}
	result.Issues = append(result.Issues, issues...)
	// Type required for task query, not metadata-only.
	if !hasParsedField(directive, "type") {
		result.Issues = append(result.Issues, Issue{
			Path:    doc.path,
			Line:    directive.opening.line,
			Message: "missing task type",
		})
	}
	if parentList {
		result.Path = result.sourceRef
	} else {
		result.Path = doc.path
	}
	return result
}

// directiveFields returns unique, usable fields and local findings on demand.
// Validation can reuse this for any directive, including metadata-only ones.
func directiveFields(path string, parsed *parsedDirective) (map[string]parsedField, []Issue) {
	fields := make(map[string]parsedField)
	var issues []Issue
	counts := make(map[string]int)
	for _, field := range parsed.fields {
		counts[field.key]++
	}
	seen := make(map[string]bool)
	for _, field := range parsed.fields {
		if seen[field.key] {
			issues = append(issues, Issue{
				Path: path, Line: field.span.line,
				Message: "repeated metadata field: " + field.key,
			})
		}
		seen[field.key] = true
		if message := localFieldMessage(field.key, field.value); message != "" {
			issues = append(issues, Issue{
				Path: path, Line: field.span.line, Message: message,
			})
		} else if counts[field.key] == 1 {
			fields[field.key] = field
		}
	}
	if counts["source"] > 0 && counts["uuid"] > 0 {
		for _, field := range parsed.fields {
			if field.key == "uuid" {
				issues = append(issues, Issue{
					Path: path, Line: field.span.line,
					Message: "source and uuid are mutually exclusive",
				})
				break
			}
		}
		delete(fields, "source")
		delete(fields, "uuid")
	} else if counts["source"] == 0 && counts["uuid"] == 0 {
		issues = append(issues, Issue{
			Path: path, Line: parsed.opening.line, Message: "missing uuid",
		})
	}
	return fields, issues
}

// Shared by task conversion and legacy validation.
// Local conversion never checks reference existence or cross-document semantics.
func localFieldMessage(key, value string) string {
	if _, known := knownMetadataFields[key]; !known {
		return "unknown metadata field: " + key
	}
	if value == "" {
		return "empty metadata field: " + key
	}
	switch key {
	case "type":
		if _, valid := validTypes[value]; !valid {
			return "invalid type: " + value
		}
	case "status":
		if _, valid := validStatuses[value]; !valid {
			return "invalid status: " + value
		}
	case "uuid":
		if !uuidV4.MatchString(value) {
			return "invalid UUIDv4: " + value
		}
	}
	return ""
}
