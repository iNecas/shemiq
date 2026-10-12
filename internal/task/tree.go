package task

// This file builds the validated, merged task hierarchy and defines its public
// JSON contract. Unlike the parser-oriented Task model in task.go, Tree* types
// expose only integration-facing metadata, keeping JSON and generated TypeScript
// types independent of parsing details and internal validation state.

import (
	"fmt"
	"sort"
	"strings"
)

// TreeTaskType is the exported task type for the JSON contract.
type TreeTaskType string

const (
	TreeTypeTopLevel TreeTaskType = "top-level"
	TreeTypeTask     TreeTaskType = "task"
)

// TreeStatus is the exported task status for the JSON contract.
type TreeStatus string

const (
	TreeStatusNew     TreeStatus = "new"
	TreeStatusRefined TreeStatus = "refined"
	TreeStatusDone    TreeStatus = "done"
)

// TreeDocument is the top-level JSON tree container.
type TreeDocument struct {
	Tasks []TreeNode `json:"tasks"`
}

// TreeNode represents one task in the merged tree.
type TreeNode struct {
	Type        TreeTaskType `json:"type"`
	Title       string       `json:"title"`
	Status      TreeStatus   `json:"status"`
	ProjectRoot string       `json:"projectRoot"`
	Path        string       `json:"path,omitempty"`
	UUID        string       `json:"uuid,omitempty"`
	Subtasks    []TreeNode   `json:"subtasks"`
}

// ValidationError carries located semantic findings that prevent tree construction.
type ValidationError struct {
	Issues []Issue
}

// Tree returns the read-only, fully validated merged task tree
// reachable from the store's configured directory scope.
// File scopes are rejected.
func (s *Store) Tree() (TreeDocument, error) {
	if err := s.load(); err != nil {
		return TreeDocument{}, err
	}
	if !s.scope.directory {
		return TreeDocument{}, fmt.Errorf(
			"tree requires a directory scope, not a file: %s",
			s.scope.path)
	}

	tasks, err := s.reachableTasks()
	if err != nil {
		return TreeDocument{}, err
	}

	issues, err := s.validateAllTasks(tasks)
	if err != nil {
		return TreeDocument{}, err
	}
	if err := validationIssuesToError(issues); err != nil {
		return TreeDocument{}, err
	}

	return buildTreeDocument(tasks), nil
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

// validationIssuesToError converts semantic findings into a tree failure.
func validationIssuesToError(issues []Issue) error {
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// buildTreeDocument constructs the merged tree from
// validated, reached tasks.
func buildTreeDocument(tasks []*Task) TreeDocument {
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
	var buildNode func(entry *Task, definingDoc *Task) TreeNode
	buildNode = func(entry *Task, definingDoc *Task) TreeNode {
		// Check if the source target exists as a loaded document.
		if entry.sourceRef != "" {
			if child, ok := byDocPath[entry.sourceRef]; ok {
				// Use the child document's authoritative metadata.
				node := TreeNode{
					Type:        toTreeType(child.Type),
					Title:       child.Title,
					Status:      toTreeStatus(child.Status),
					ProjectRoot: child.ProjectRoot,
					Path:        child.document.path,
					UUID:        child.uuid,
					Subtasks:    make([]TreeNode, 0),
				}
				// Recursively merge child's subtasks.
				for i := range child.Subtasks {
					node.Subtasks = append(node.Subtasks,
						buildNode(&child.Subtasks[i], child))
				}
				return node
			}
			// Missing source target: retain entry metadata.
			node := TreeNode{
				Type:        toTreeType(entry.Type),
				Title:       entry.Title,
				Status:      toTreeStatus(entry.Status),
				ProjectRoot: definingDoc.ProjectRoot,
				Path:        entry.sourceRef,
				Subtasks:    make([]TreeNode, 0),
			}
			return node
		}
		// Inline entry (no source): retain metadata, omit path.
		node := TreeNode{
			Type:        toTreeType(entry.Type),
			Title:       entry.Title,
			Status:      toTreeStatus(entry.Status),
			ProjectRoot: definingDoc.ProjectRoot,
			Subtasks:    make([]TreeNode, 0),
		}
		if entry.uuid != "" {
			node.UUID = entry.uuid
		}
		return node
	}

	// Roots: every reached document not owned by another.
	var roots []TreeNode
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
		node := TreeNode{
			Type:        toTreeType(t.Type),
			Title:       t.Title,
			Status:      toTreeStatus(t.Status),
			ProjectRoot: t.ProjectRoot,
			Path:        t.document.path,
			UUID:        t.uuid,
			Subtasks:    make([]TreeNode, 0),
		}
		for i := range t.Subtasks {
			node.Subtasks = append(node.Subtasks,
				buildNode(&t.Subtasks[i], t))
		}
		roots = append(roots, node)
	}

	if roots == nil {
		roots = make([]TreeNode, 0)
	}
	return TreeDocument{Tasks: roots}
}

func toTreeType(t TaskType) TreeTaskType {
	switch t {
	case TypeTopLevel:
		return TreeTypeTopLevel
	case TypeTask:
		return TreeTypeTask
	default:
		return TreeTaskType(t)
	}
}

func toTreeStatus(s string) TreeStatus {
	switch s {
	case "new":
		return TreeStatusNew
	case "refined":
		return TreeStatusRefined
	case "done":
		return TreeStatusDone
	default:
		return TreeStatus(s)
	}
}
