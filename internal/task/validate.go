package task

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Validate checks the configured scope and every document
// reachable through its usable local references. An empty
// scope selects the nearest project's active tasks; it never
// creates a project. With fix it inserts missing UUIDs and
// promotes linked statuses, then reports repairs followed by
// the remaining findings from the resulting state.
// Syntax/loading failures, including in referenced documents,
// are operation errors rather than semantic findings.
func (s *Store) Validate(fix bool) ([]Issue, error) {
	if err := s.load(); err != nil {
		return nil, err
	}
	tasks, err := s.reachableTasks()
	if err != nil {
		return nil, err
	}
	var repaired []Issue
	if fix {
		repaired, err = s.repair(tasks)
		if err != nil {
			return nil, err
		}
		tasks, err = s.reachableTasks()
		if err != nil {
			return nil, err
		}
	}
	unresolved, err := s.validateAllTasks(tasks)
	if err != nil {
		return nil, err
	}
	return append(repaired, unresolved...), nil
}

// scopedTasks returns cached tasks within the configured
// scope in path order.
func (s *Store) scopedTasks() []*Task {
	var paths []string
	for p := range s.tasks {
		if s.contains(p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	result := make([]*Task, 0, len(paths))
	for _, p := range paths {
		result = append(result, s.tasks[p])
	}
	return result
}

// reachableTasks returns scoped tasks plus any tasks
// reachable through usable source/parent references.
// Non-existent, directory, and non-Markdown targets are
// left for validation.
func (s *Store) reachableTasks() ([]*Task, error) {
	visited := make(map[string]bool)
	byPath := make(map[string]*Task)
	var queue []string

	for _, t := range s.scopedTasks() {
		resolved, _ := s.documentPath(t.document.path)
		if !visited[resolved] {
			visited[resolved] = true
			byPath[resolved] = t
			queue = append(queue,
				taskReferenceTargets(t)...)
		}
	}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		// Check existence before resolving symlinks;
		// a missing source target is allowed.
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}
		if !info.Mode().IsRegular() ||
			!isMarkdown(path) {
			continue
		}
		resolved, err := s.documentPath(path)
		if err != nil {
			return nil, err
		}
		if visited[resolved] {
			continue
		}
		visited[resolved] = true
		t, err := s.loadTask(path, false)
		if err != nil {
			return nil, err
		}
		byPath[resolved] = t
		queue = append(queue, taskReferenceTargets(t)...)
	}

	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	result := make([]*Task, 0, len(paths))
	for _, p := range paths {
		result = append(result, byPath[p])
	}
	return result, nil
}

// taskReferenceTargets returns source/parent reference paths
// from a task and its subtasks.
func taskReferenceTargets(t *Task) []string {
	var targets []string
	collect := func(task *Task) {
		if task.sourceRef != "" {
			targets = append(targets, task.sourceRef)
		}
		if task.parentRef != "" {
			targets = append(targets, task.parentRef)
		}
	}
	collect(t)
	for i := range t.Subtasks {
		collect(&t.Subtasks[i])
	}
	return targets
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func allDirectives(
	section *parsedSection,
) []*parsedDirective {
	directives := append([]*parsedDirective(nil), section.directives...)
	for _, child := range section.children {
		directives = append(directives, allDirectives(child)...)
	}
	return directives
}

// taskStatusRank returns a comparable rank for usable
// statuses. An omitted status means new; a present-but-
// unusable status has no rank.
func taskStatusRank(t *Task) (int, bool) {
	if t.directive == nil {
		return 0, true // directive-less subtask = new
	}
	if hasParsedField(t.directive, "status") {
		for rank, name := range statusNames {
			if name == t.Status {
				return rank, true
			}
		}
		return 0, false
	}
	return 0, true // omitted status = new
}

var statusNames = []string{"new", "refined", "done"}

// flatTasks returns a task and all its subtasks as a flat
// slice of pointers.
func flatTasks(tasks []*Task) []*Task {
	var result []*Task
	for _, t := range tasks {
		result = append(result, t)
		for i := range t.Subtasks {
			result = append(result, &t.Subtasks[i])
		}
	}
	return result
}

// validateAllTasks collects local directive findings and
// performs cross-document checks: UUID uniqueness, reference
// suitability, and reciprocal task links. Only tasks with
// directives contribute metadata findings; structural issues
// (missing heading/directive) are for queries, not validation.
func (s *Store) validateAllTasks(
	tasks []*Task,
) ([]Issue, error) {
	all := flatTasks(tasks)
	var issues []Issue

	for _, t := range all {
		if t.directive == nil {
			continue
		}
		// Local field findings from the directive.
		_, fieldIssues := directiveFields(
			t.document.path, t.directive)
		issues = append(issues, fieldIssues...)
		refs, err := taskReferenceIssues(t)
		if err != nil {
			return nil, err
		}
		issues = append(issues, refs...)
	}
	issues = append(issues, validateUUIDUniqueness(all)...)
	return append(issues,
		validateTaskLinks(all)...), nil
}

// validateUUIDUniqueness reports repeated usable UUIDs,
// retaining the location of their first occurrence.
func validateUUIDUniqueness(all []*Task) []Issue {
	var issues []Issue
	seen := make(map[string]Issue)
	for _, t := range all {
		if t.directive == nil || t.uuid == "" {
			continue
		}
		line := uuidLine(t)
		if first, exists := seen[t.uuid]; exists {
			issues = append(issues, Issue{
				t.document.path, line,
				fmt.Sprintf(
					"duplicate uuid %s (first at %s:%d)",
					t.uuid, first.Path, first.Line),
				false,
			})
		} else {
			seen[t.uuid] = Issue{
				Path: t.document.path,
				Line: line,
			}
		}
	}
	return issues
}

func uuidLine(t *Task) int {
	if t.directive == nil {
		return t.section.heading.line
	}
	for _, f := range t.directive.fields {
		if f.key == "uuid" {
			return f.span.line
		}
	}
	return t.directive.opening.line
}

// taskReferenceIssues reports reference suitability.
// Missing source awaits refinement; missing parent is
// reported. Directory and non-Markdown targets are findings.
func taskReferenceIssues(t *Task) ([]Issue, error) {
	if t.directive == nil {
		return nil, nil
	}
	var issues []Issue
	for _, ref := range []struct {
		key  string
		path string
	}{
		{"source", t.sourceRef},
		{"parent", t.parentRef},
	} {
		if ref.path == "" {
			continue
		}
		field := findField(t.directive, ref.key)
		info, err := os.Stat(ref.path)
		if os.IsNotExist(err) {
			if ref.key == "parent" {
				issues = append(issues, Issue{
					t.document.path,
					field.span.line,
					"parent is not an existing " +
						"file: " + field.value,
					false,
				})
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(
				"inspect %s %s: %w",
				ref.key, ref.path, err)
		}
		if info.IsDir() || !info.Mode().IsRegular() ||
			!isMarkdown(ref.path) {
			issues = append(issues, Issue{
				t.document.path,
				field.span.line,
				ref.key +
					" is not a Markdown file: " +
					field.value,
				false,
			})
		}
	}
	return issues, nil
}

func findField(
	d *parsedDirective, key string,
) parsedField {
	for _, f := range d.fields {
		if f.key == key {
			return f
		}
	}
	return parsedField{}
}

func statusLine(t *Task) int {
	if t.directive == nil {
		return t.section.heading.line
	}
	for _, f := range t.directive.fields {
		if f.key == "status" {
			return f.span.line
		}
	}
	return t.directive.opening.line
}

// validateTaskLinks reports status mismatches between
// usable reciprocal task pairs.
func validateTaskLinks(all []*Task) []Issue {
	pairs, issues := linkedTaskPairs(all)
	for _, pair := range pairs {
		pt, ct := pair[0], pair[1]
		pRank, pOK := taskStatusRank(pt)
		cRank, cOK := taskStatusRank(ct)
		if pOK && cOK && pRank != cRank {
			line := statusLine(pt)
			issues = append(issues, Issue{
				pt.document.path, line,
				fmt.Sprintf(
					"status mismatch with %s: "+
						"%s vs %s",
					ct.document.path,
					statusNames[pRank],
					statusNames[cRank]),
				false,
			})
		}
	}
	return issues
}

// linkedTaskPairs finds unambiguous reciprocal source/parent
// pairs and reports broken or ambiguous links.
func linkedTaskPairs(
	all []*Task,
) ([][2]*Task, []Issue) {
	byDocPath := make(map[string][]*Task)
	for _, t := range all {
		byDocPath[t.document.path] = append(
			byDocPath[t.document.path], t)
	}
	var pairs [][2]*Task
	var issues []Issue
	for _, t := range all {
		if t.Type != TypeTask || t.sourceRef == "" {
			continue
		}
		field := findField(t.directive, "source")
		children := byDocPath[t.sourceRef]
		if len(children) == 0 {
			continue
		}
		parents := matchingTasks(
			children, "parent", t.document.path)
		backlinks := matchingTasks(
			byDocPath[t.document.path],
			"source", t.sourceRef)
		if len(parents) != 1 || len(backlinks) != 1 {
			issues = append(issues, Issue{
				t.document.path,
				field.span.line,
				fmt.Sprintf(
					"source %s does not have "+
						"exactly one reciprocal "+
						"task parent",
					field.value),
				false,
			})
		} else {
			pairs = append(pairs,
				[2]*Task{t, parents[0]})
		}
	}
	for _, t := range all {
		if t.Type != TypeTask || t.parentRef == "" {
			continue
		}
		field := findField(t.directive, "parent")
		owner := byDocPath[t.parentRef]
		if len(owner) == 0 {
			continue
		}
		backlinks := matchingTasks(
			owner, "source", t.document.path)
		if len(backlinks) != 1 {
			issues = append(issues, Issue{
				t.document.path,
				field.span.line,
				fmt.Sprintf(
					"parent %s does not have "+
						"exactly one reciprocal "+
						"task source",
					field.value),
				false,
			})
		}
	}
	return pairs, issues
}

func matchingTasks(
	tasks []*Task, key, target string,
) []*Task {
	var matches []*Task
	for _, t := range tasks {
		if t.Type != TypeTask {
			continue
		}
		switch key {
		case "source":
			if t.sourceRef == target {
				matches = append(matches, t)
			}
		case "parent":
			if t.parentRef == target {
				matches = append(matches, t)
			}
		}
	}
	return matches
}

type textEdit struct {
	start, end int
	text       string
}

// repair plans targeted edits against original-byte
// snapshots, writes them per document, and refreshes
// affected tasks.
func (s *Store) repair(
	tasks []*Task,
) ([]Issue, error) {
	all := flatTasks(tasks)
	edits := make(map[*parsedDocument][]textEdit)
	var fixed []Issue

	for _, t := range all {
		if t.directive == nil {
			continue
		}
		if eligibleForUUID(t.directive) {
			id, err := NewUUID()
			if err != nil {
				return nil, err
			}
			edits[t.document] = append(
				edits[t.document],
				textEdit{
					t.directive.insertAt,
					t.directive.insertAt,
					"uuid: " + id +
						t.directive.lineEnding,
				})
			fixed = append(fixed, Issue{
				t.document.path,
				t.directive.opening.line,
				"missing uuid", true,
			})
		}
	}

	pairs, _ := linkedTaskPairs(all)
	for _, p := range planStatusPromotions(pairs) {
		edit, line := p.textEdit()
		edits[p.task.document] = append(
			edits[p.task.document], edit)
		fixed = append(fixed, Issue{
			p.task.document.path, line,
			"status promoted to " +
				statusNames[p.target],
			true,
		})
	}

	for doc, documentEdits := range edits {
		if err := applyEdits(doc, documentEdits); err != nil {
			return nil, err
		}
	}
	refreshed := make(map[string]bool)
	for doc := range edits {
		if !refreshed[doc.path] {
			refreshed[doc.path] = true
			if _, err := s.loadTask(
				doc.path, true); err != nil {
				return nil, err
			}
		}
	}
	return fixed, nil
}

// eligibleForUUID requires both source and uuid absent from
// the original directive.
func eligibleForUUID(directive *parsedDirective) bool {
	return !hasParsedField(directive, "source") &&
		!hasParsedField(directive, "uuid")
}

type statusPromotion struct {
	task   *Task
	target int
}

func (p statusPromotion) textEdit() (textEdit, int) {
	name := statusNames[p.target]
	if p.task.directive != nil {
		for _, f := range p.task.directive.fields {
			if f.key == "status" {
				return textEdit{
					f.valueSpan.start,
					f.valueSpan.end,
					name,
				}, f.span.line
			}
		}
	}
	d := p.task.directive
	return textEdit{
		d.insertAt, d.insertAt,
		"status: " + name + d.lineEnding,
	}, d.opening.line
}

// planStatusPromotions raises every member of each connected
// group of eligible reciprocal links to the group's maximum
// usable status.
func planStatusPromotions(
	pairs [][2]*Task,
) []statusPromotion {
	parent := make(map[*Task]*Task)
	var find func(*Task) *Task
	find = func(t *Task) *Task {
		if parent[t] != t {
			parent[t] = find(parent[t])
		}
		return parent[t]
	}
	add := func(t *Task) bool {
		if _, ok := taskStatusRank(t); !ok {
			return false
		}
		if _, seen := parent[t]; !seen {
			parent[t] = t
		}
		return true
	}
	for _, pair := range pairs {
		if add(pair[0]) && add(pair[1]) {
			parent[find(pair[0])] = find(pair[1])
		}
	}
	maxRank := make(map[*Task]int)
	for t := range parent {
		rank, _ := taskStatusRank(t)
		if root := find(t); rank > maxRank[root] {
			maxRank[root] = rank
		}
	}
	var promotions []statusPromotion
	for t := range parent {
		rank, _ := taskStatusRank(t)
		if target := maxRank[find(t)]; target != rank {
			promotions = append(promotions,
				statusPromotion{t, target})
		}
	}
	return promotions
}

// applyEdits rewrites a document from its original snapshot.
func applyEdits(
	doc *parsedDocument, edits []textEdit,
) error {
	sort.SliceStable(edits, func(i, j int) bool {
		return edits[i].start < edits[j].start
	})
	var output bytes.Buffer
	start := 0
	for _, edit := range edits {
		output.Write(doc.data[start:edit.start])
		output.WriteString(edit.text)
		start = edit.end
	}
	output.Write(doc.data[start:])
	if err := os.WriteFile(
		doc.path, output.Bytes(), 0644); err != nil {
		return fmt.Errorf(
			"repair %s: %w", doc.path, err)
	}
	return nil
}
