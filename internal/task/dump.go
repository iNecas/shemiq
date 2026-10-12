package task

import (
	"fmt"
	"sort"
	"strings"
)

// DumpTaskType is the exported task type for the JSON contract.
type DumpTaskType string

const (
	DumpTypeTopLevel DumpTaskType = "top-level"
	DumpTypeTask     DumpTaskType = "task"
)

// DumpStatus is the exported task status for the JSON contract.
type DumpStatus string

const (
	DumpStatusNew     DumpStatus = "new"
	DumpStatusRefined DumpStatus = "refined"
	DumpStatusDone    DumpStatus = "done"
)

// DumpDocument is the top-level JSON export container.
type DumpDocument struct {
	Tasks []DumpNode `json:"tasks"`
}

// DumpNode represents one task in the merged export tree.
type DumpNode struct {
	Type        DumpTaskType `json:"type"`
	Title       string       `json:"title"`
	Status      DumpStatus   `json:"status"`
	ProjectRoot string       `json:"projectRoot"`
	Path        string       `json:"path,omitempty"`
	UUID        string       `json:"uuid,omitempty"`
	Subtasks    []DumpNode   `json:"subtasks"`
}

// ValidationError carries located semantic findings that prevent a dump.
type ValidationError struct {
	Issues []Issue
}

// Dump performs a read-only, fully validated export of the
// merged task tree reachable from the store's configured
// directory scope. File scopes are rejected.
func (s *Store) Dump() (DumpDocument, error) {
	if err := s.load(); err != nil {
		return DumpDocument{}, err
	}
	if !s.scope.directory {
		return DumpDocument{}, fmt.Errorf(
			"dump requires a directory scope, not a file: %s",
			s.scope.path)
	}

	tasks, err := s.reachableTasks()
	if err != nil {
		return DumpDocument{}, err
	}

	issues, err := s.validateAllTasks(tasks)
	if err != nil {
		return DumpDocument{}, err
	}
	if err := validationIssuesToError(issues); err != nil {
		return DumpDocument{}, err
	}

	return buildDumpDocument(tasks), nil
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "validation failed"
	}
	var b strings.Builder
	for i, issue := range e.Issues {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s:%d: %s", issue.Path, issue.Line, issue.Message)
	}
	return b.String()
}

// validationIssuesToError converts semantic findings into a dump failure.
func validationIssuesToError(issues []Issue) error {
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// buildDumpDocument constructs the merged export tree from
// validated, reached tasks.
func buildDumpDocument(tasks []*Task) DumpDocument {
	// Index tasks by their document path for lookup.
	byDocPath := make(map[string]*Task)
	for _, t := range tasks {
		byDocPath[t.document.path] = t
	}

	// Track which document paths are owned (appear as
	// source targets in some parent's subtask list).
	owned := make(map[string]bool)
	for _, t := range tasks {
		for _, sub := range t.Subtasks {
			if sub.sourceRef != "" {
				owned[sub.sourceRef] = true
			}
		}
	}

	// Build nodes recursively.
	var buildNode func(entry *Task, definingDoc *Task) DumpNode
	buildNode = func(entry *Task, definingDoc *Task) DumpNode {
		// Check if the source target exists as a loaded document.
		if entry.sourceRef != "" {
			if child, ok := byDocPath[entry.sourceRef]; ok {
				// Use the child document's authoritative metadata.
				node := DumpNode{
					Type:        toDumpType(child.Type),
					Title:       child.Title,
					Status:      toDumpStatus(child.Status),
					ProjectRoot: child.ProjectRoot,
					Path:        child.document.path,
					UUID:        child.uuid,
					Subtasks:    make([]DumpNode, 0),
				}
				// Recursively merge child's subtasks.
				for i := range child.Subtasks {
					node.Subtasks = append(node.Subtasks,
						buildNode(&child.Subtasks[i], child))
				}
				return node
			}
			// Missing source target: retain entry metadata.
			node := DumpNode{
				Type:        toDumpType(entry.Type),
				Title:       entry.Title,
				Status:      toDumpStatus(entry.Status),
				ProjectRoot: definingDoc.ProjectRoot,
				Path:        entry.sourceRef,
				Subtasks:    make([]DumpNode, 0),
			}
			return node
		}
		// Inline entry (no source): retain metadata, omit path.
		node := DumpNode{
			Type:        toDumpType(entry.Type),
			Title:       entry.Title,
			Status:      toDumpStatus(entry.Status),
			ProjectRoot: definingDoc.ProjectRoot,
			Subtasks:    make([]DumpNode, 0),
		}
		if entry.uuid != "" {
			node.UUID = entry.uuid
		}
		return node
	}

	// Roots: every reached document not owned by another.
	var roots []DumpNode
	var rootPaths []string
	for _, t := range tasks {
		if !owned[t.document.path] {
			rootPaths = append(rootPaths, t.document.path)
		}
	}
	sort.Strings(rootPaths)

	// Deduplicate root paths (already sorted).
	seen := make(map[string]bool)
	for _, p := range rootPaths {
		if seen[p] {
			continue
		}
		seen[p] = true
		t := byDocPath[p]
		node := DumpNode{
			Type:        toDumpType(t.Type),
			Title:       t.Title,
			Status:      toDumpStatus(t.Status),
			ProjectRoot: t.ProjectRoot,
			Path:        t.document.path,
			UUID:        t.uuid,
			Subtasks:    make([]DumpNode, 0),
		}
		for i := range t.Subtasks {
			node.Subtasks = append(node.Subtasks,
				buildNode(&t.Subtasks[i], t))
		}
		roots = append(roots, node)
	}

	if roots == nil {
		roots = make([]DumpNode, 0)
	}
	return DumpDocument{Tasks: roots}
}

func toDumpType(t TaskType) DumpTaskType {
	switch t {
	case TypeTopLevel:
		return DumpTypeTopLevel
	case TypeTask:
		return DumpTypeTask
	default:
		return DumpTaskType(t)
	}
}

func toDumpStatus(s string) DumpStatus {
	switch s {
	case "new":
		return DumpStatusNew
	case "refined":
		return DumpStatusRefined
	case "done":
		return DumpStatusDone
	default:
		return DumpStatus(s)
	}
}
