package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateSubtask inserts a directive-less subtask heading and
// description into an existing top-level document's ## Tasks
// section. Empty after appends; otherwise inserts after the
// named sibling. parentPath must identify a Markdown file; callers resolve
// directory arguments. Returns the edited parent's absolute path.
func (s *Store) CreateSubtask(
	parentPath, title, description, after string,
) (string, error) {
	parent, err := s.loadTask(parentPath, false)
	if err != nil {
		return "", err
	}
	absPath := parent.document.path
	if parent.ProjectRoot == "" {
		return "", fmt.Errorf("parent is not inside a .shemiq project: %s", absPath)
	}
	if parent.Type != TypeTopLevel {
		return "", fmt.Errorf(
			"parent is not a top-level task: %s",
			absPath)
	}
	title, description, err = validateSubtaskInput(parent, title, description)
	if err != nil {
		return "", err
	}
	edit, err := planSubtaskInsertion(parent, title, description, after)
	if err != nil {
		return "", err
	}
	stagingDir, err := os.MkdirTemp("", "shemiq-edit-*")
	if err != nil {
		return "", fmt.Errorf("stage edits for %s: %w", absPath, err)
	}
	defer os.RemoveAll(stagingDir)
	stagedPath := filepath.Join(stagingDir, "top-level.md")
	if err := applyEdits(parent.document, []textEdit{edit}, stagedPath); err != nil {
		return "", err
	}
	if err := verifyInsertion(parent, stagedPath, title); err != nil {
		return "", err
	}
	if err := replaceEditedDocument(absPath, stagedPath); err != nil {
		return "", fmt.Errorf("replace %s: %w", absPath, err)
	}
	if _, err := s.loadTask(absPath, true); err != nil {
		return absPath, fmt.Errorf(
			"edited %s but failed to refresh cache: %w",
			absPath, err)
	}
	return absPath, nil
}

func validateSubtaskInput(parent *Task, title, description string) (string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", fmt.Errorf("title is required")
	}
	if strings.ContainsAny(title, "\r\n") {
		return "", "", fmt.Errorf("title must be one line")
	}
	// Ensure the heading parses back to the intended title.
	headingLine := "### " + title
	_, parsed, ok := markdownHeading(headingLine)
	if !ok || parsed != title {
		return "", "", fmt.Errorf(
			"title would be changed by heading syntax: %q",
			title)
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return "", "", fmt.Errorf("description is required")
	}
	for _, sub := range parent.Subtasks {
		if sub.Title == title {
			return "", "", fmt.Errorf("duplicate subtask title: %s", title)
		}
	}
	return title, description, nil
}

// planSubtaskInsertion computes the targeted edit to insert
// a subtask heading and description.
func planSubtaskInsertion(
	parent *Task, title, description, after string,
) (textEdit, error) {
	doc := parent.document
	primary := parent.section
	le := documentLineEnding(doc)

	// Find the ## Tasks section.
	tasksSection, err := findTasksSection(primary)
	if err != nil {
		return textEdit{}, err
	}

	block := "### " + title + le + le + description + le

	if tasksSection == nil {
		if after != "" {
			return textEdit{}, fmt.Errorf("--after sibling not found: %s", after)
		}
		// Append ## Tasks at end of document.
		pos := len(doc.data)
		prefix := le
		if pos > 0 && doc.data[pos-1] != '\n' {
			prefix = le + le
		}
		text := prefix + "## Tasks" + le + le + block
		return textEdit{pos, pos, text}, nil
	}

	// Check for [TBD] placeholder.
	if isTasksPlaceholder(doc, tasksSection) {
		if after != "" {
			return textEdit{}, fmt.Errorf("--after sibling not found: %s", after)
		}
		// Replace the entire section content (after heading)
		// with the new subtask block.
		contentStart := tasksSection.heading.end
		contentEnd := tasksSection.span.end
		text := le + block
		return textEdit{contentStart, contentEnd, text}, nil
	}

	if after == "" {
		return planAppendSubtask(parent, tasksSection, block, le)
	}
	return planInsertAfter(parent, tasksSection, after, block, le)
}

func findTasksSection(
	primary *parsedSection,
) (*parsedSection, error) {
	var found []*parsedSection
	for _, child := range primary.children {
		if child.level == 2 && child.title == "Tasks" {
			found = append(found, child)
		}
	}
	if len(found) > 1 {
		return nil, fmt.Errorf(
			"ambiguous structure: multiple ## Tasks sections")
	}
	if len(found) == 0 {
		return nil, nil
	}
	return found[0], nil
}

// isTasksPlaceholder returns true when [TBD] is the sole
// non-whitespace content of the Tasks section body (after
// its heading, before any child sections).
func isTasksPlaceholder(doc *parsedDocument, tasks *parsedSection) bool {
	bodyStart := tasks.heading.end
	bodyEnd := tasks.span.end
	if len(tasks.children) > 0 {
		bodyEnd = tasks.children[0].span.start
	}
	body := strings.TrimSpace(
		string(doc.data[bodyStart:bodyEnd]))
	return body == "[TBD]"
}

func planAppendSubtask(
	parent *Task, tasks *parsedSection,
	block, le string,
) (textEdit, error) {
	pos := tasks.span.end
	// Ensure blank line before the new heading.
	prefix := ensureBlankLineBefore(
		parent.document.data, pos, le)
	return textEdit{pos, pos, prefix + block}, nil
}

func planInsertAfter(
	parent *Task, tasks *parsedSection,
	after, block, le string,
) (textEdit, error) {
	var match *parsedSection
	for _, child := range tasks.children {
		if child.level == 3 && child.title == after {
			if match != nil {
				return textEdit{}, fmt.Errorf(
					"ambiguous --after match: %s",
					after)
			}
			match = child
		}
	}
	if match == nil {
		return textEdit{}, fmt.Errorf(
			"--after sibling not found: %s", after)
	}
	pos := match.span.end
	prefix := ensureBlankLineBefore(
		parent.document.data, pos, le)
	return textEdit{pos, pos, prefix + block}, nil
}

// verifyInsertion reads the staged result and checks it adds exactly one new
// directive-less subtask without changing existing section relationships.
func verifyInsertion(parent *Task, stagedPath, title string) error {
	doc := parent.document
	proposed, err := os.ReadFile(stagedPath)
	if err != nil {
		return fmt.Errorf("read staged edits for %s: %w", doc.path, err)
	}

	newDoc, err := parseMarkdownDocument(doc.path, proposed)
	if err != nil {
		return fmt.Errorf("description would break document structure: %w", err)
	}
	newTask := documentTask(newDoc)
	if newTask.Type != TypeTopLevel {
		return fmt.Errorf("insertion changed document type")
	}

	// Check that exactly one new subtask was added.
	oldCount := len(parent.Subtasks)
	newCount := len(newTask.Subtasks)
	if newCount != oldCount+1 {
		return fmt.Errorf("description creates %d subtask(s) instead of 1",
			newCount-oldCount)
	}

	// Verify the new subtask has the intended title and
	// no directive.
	found := false
	for _, sub := range newTask.Subtasks {
		if sub.Title == title && sub.directive == nil {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("inserted subtask not found with expected title %q", title)
	}

	// Verify existing subtasks are preserved.
	for _, old := range parent.Subtasks {
		preserved := false
		for _, sub := range newTask.Subtasks {
			if sub.Title == old.Title {
				preserved = true
				break
			}
		}
		if !preserved {
			return fmt.Errorf("insertion would remove existing subtask %q", old.Title)
		}
	}

	// Check for accidental directives in the description.
	for _, d := range allDirectives(newDoc.root) {
		isExisting := false
		for _, od := range allDirectives(doc.root) {
			if d.opening.line == od.opening.line {
				isExisting = true
				break
			}
		}
		if !isExisting {
			return fmt.Errorf("description introduces a:::shemiq directive")
		}
	}

	return nil
}
