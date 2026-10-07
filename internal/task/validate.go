package task

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Validate checks the configured scope and every document reachable through its
// usable local references. An empty scope selects the nearest project's active
// tasks; it never creates a project. With fix it inserts missing UUIDs and
// promotes linked statuses, then reports repairs followed by the remaining
// findings from the resulting state. Syntax/loading failures, including in
// referenced documents, are operation errors rather than semantic findings.
func (s *Store) Validate(fix bool) ([]Issue, error) {
	seeds, err := s.scopedSeeds()
	if err != nil {
		return nil, err
	}
	docs, err := s.reachableDocuments(seeds)
	if err != nil {
		return nil, err
	}
	var repaired []Issue
	if fix {
		repaired, err = s.repair(docs)
		if err != nil {
			return nil, err
		}
		// Re-enumerate from refreshed snapshots: all remaining findings describe
		// the final on-disk state, never stale pre-edit offsets.
		docs, err = s.reachableDocuments(seeds)
		if err != nil {
			return nil, err
		}
	}
	unresolved, err := s.validateDocuments(docs)
	if err != nil {
		return nil, err
	}
	return append(repaired, unresolved...), nil
}

// scopedSeeds enumerates documents for the configured scope independently of
// task queries. A file scope selects that document; a directory scope (default
// active tasks or an explicit directory) scans Markdown files recursively.
func (s *Store) scopedSeeds() ([]string, error) {
	if err := s.resolveScope(); err != nil {
		return nil, err
	}
	if !s.scope.directory {
		return []string{s.scope.path}, nil
	}
	var paths []string
	err := filepath.WalkDir(s.scope.path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && isMarkdown(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", s.scope.path, err)
	}
	return paths, nil
}

// reachableDocuments loads the seeds and follows usable source/parent references,
// including targets outside the initial scope. Documents are deduplicated by
// resolved filesystem identity so aliases and cycles load each file once.
// Non-existent, directory, and non-Markdown targets are left for validation.
func (s *Store) reachableDocuments(seeds []string) ([]*parsedDocument, error) {
	visited := make(map[string]bool)
	byPath := make(map[string]*parsedDocument)
	queue := append([]string(nil), seeds...)
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		resolved, err := s.documentPath(path)
		if err != nil {
			return nil, err
		}
		if visited[resolved] {
			continue
		}
		visited[resolved] = true
		doc, err := s.loadParsedDocument(path)
		if err != nil {
			return nil, err
		}
		byPath[doc.path] = doc
		for _, directive := range allDirectives(doc.root) {
			fields, _ := directiveFields(doc.path, directive)
			for _, key := range []string{"source", "parent"} {
				field, ok := fields[key]
				if !ok {
					continue
				}
				target := resolvedReferencePath(doc.path, field.value)
				info, err := os.Stat(target)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("inspect %s %s: %w", key, target, err)
				}
				if info.Mode().IsRegular() && isMarkdown(target) {
					queue = append(queue, target)
				}
			}
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	docs := make([]*parsedDocument, 0, len(paths))
	for _, path := range paths {
		docs = append(docs, byPath[path])
	}
	return docs, nil
}

// loadParsedDocument loads and caches a snapshot, returning the parsed document
// directly so validation can traverse every directive without a primary task.
func (s *Store) loadParsedDocument(path string) (*parsedDocument, error) {
	if _, err := s.loadDocument(path, false); err != nil {
		return nil, err
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return nil, err
	}
	return s.documents[resolved], nil
}

func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func allDirectives(section *parsedSection) []*parsedDirective {
	directives := append([]*parsedDirective(nil), section.directives...)
	for _, child := range section.children {
		directives = append(directives, allDirectives(child)...)
	}
	return directives
}

// directiveRecord pairs a parsed directive with its usable fields and local
// findings, computed once per validation or repair pass.
type directiveRecord struct {
	doc       *parsedDocument
	directive *parsedDirective
	fields    map[string]parsedField
	issues    []Issue
}

func collectRecords(docs []*parsedDocument) []directiveRecord {
	var records []directiveRecord
	for _, doc := range docs {
		for _, directive := range allDirectives(doc.root) {
			fields, issues := directiveFields(doc.path, directive)
			records = append(records, directiveRecord{doc, directive, fields, issues})
		}
	}
	return records
}

func (r directiveRecord) isTask() bool {
	return r.fields["type"].value == "task"
}

// statusRank reports a comparable rank for usable statuses only. An omitted
// status means new; a present-but-unusable status has no rank and cannot carry
// comparisons or promotions.
func (r directiveRecord) statusRank() (int, bool) {
	if field, ok := r.fields["status"]; ok {
		for rank, name := range statusNames {
			if name == field.value {
				return rank, true
			}
		}
		return 0, false
	}
	if hasParsedField(r.directive, "status") {
		return 0, false
	}
	return 0, true
}

var statusNames = []string{"new", "refined", "done"}

// validateDocuments collects local findings and the checks not covered by
// conversion: reference suitability, cross-document UUID uniqueness, and
// reciprocal task links with status agreement.
func (s *Store) validateDocuments(docs []*parsedDocument) ([]Issue, error) {
	records := collectRecords(docs)
	var issues []Issue
	seen := make(map[string]Issue)
	for _, record := range records {
		issues = append(issues, record.issues...)
		references, err := referenceIssues(record)
		if err != nil {
			return nil, err
		}
		issues = append(issues, references...)
		if field, ok := record.fields["uuid"]; ok {
			if first, exists := seen[field.value]; exists {
				issues = append(issues, Issue{record.doc.path, field.span.line,
					fmt.Sprintf("duplicate uuid %s (first at %s:%d)",
						field.value, first.Path, first.Line), false})
			} else {
				seen[field.value] = Issue{Path: record.doc.path, Line: field.span.line}
			}
		}
	}
	return append(issues, validateLinks(records)...), nil
}

// referenceIssues reports reference suitability for usable source/parent fields.
// A missing source awaits refinement; a missing parent is reported. Directory
// and existing non-Markdown targets are located invalid-reference findings.
func referenceIssues(record directiveRecord) ([]Issue, error) {
	var issues []Issue
	for _, key := range []string{"source", "parent"} {
		field, ok := record.fields[key]
		if !ok {
			continue
		}
		target := resolvedReferencePath(record.doc.path, field.value)
		info, err := os.Stat(target)
		if os.IsNotExist(err) {
			if key == "parent" {
				issues = append(issues, Issue{record.doc.path, field.span.line,
					"parent is not an existing file: " + field.value, false})
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s %s: %w", key, target, err)
		}
		if info.IsDir() || !info.Mode().IsRegular() || !isMarkdown(target) {
			issues = append(issues, Issue{record.doc.path, field.span.line,
				key + " is not a Markdown file: " + field.value, false})
		}
	}
	return issues, nil
}

// validateLinks reports status mismatches between usable reciprocal task pairs.
func validateLinks(records []directiveRecord) []Issue {
	pairs, issues := linkedTaskPairs(records)
	for _, pair := range pairs {
		parent, child := pair[0], pair[1]
		parentRank, parentOK := parent.statusRank()
		childRank, childOK := child.statusRank()
		if parentOK && childOK && parentRank != childRank {
			field, present := parent.fields["status"]
			line := parent.directive.opening.line
			if present {
				line = field.span.line
			}
			issues = append(issues, Issue{parent.doc.path, line,
				fmt.Sprintf("status mismatch with %s: %s vs %s",
					child.doc.path, statusNames[parentRank], statusNames[childRank]), false})
		}
	}
	return issues
}

// linkedTaskPairs reports broken or ambiguous reciprocal links and returns the
// unambiguous pairs. Each pair's first record is the source-bearing parent-list
// entry; the second is the child's matching parent directive.
func linkedTaskPairs(records []directiveRecord) ([][2]directiveRecord, []Issue) {
	byPath := make(map[string][]directiveRecord)
	for _, record := range records {
		byPath[record.doc.path] = append(byPath[record.doc.path], record)
	}
	var pairs [][2]directiveRecord
	var issues []Issue
	for _, record := range records {
		if !record.isTask() {
			continue
		}
		if source, ok := record.fields["source"]; ok {
			target := resolvedReferencePath(record.doc.path, source.value)
			children, exists := byPath[target]
			if exists { // A subtask file need not exist yet.
				parents := matchingTaskDirectives(children, "parent", record.doc.path)
				backlinks := matchingTaskDirectives(byPath[record.doc.path], "source", target)
				if len(parents) != 1 || len(backlinks) != 1 {
					issues = append(issues, Issue{record.doc.path, source.span.line,
						fmt.Sprintf("source %s does not have exactly one reciprocal task parent",
							source.value), false})
				} else {
					pairs = append(pairs, [2]directiveRecord{record, parents[0]})
				}
			}
		}
		if parent, ok := record.fields["parent"]; ok {
			target := resolvedReferencePath(record.doc.path, parent.value)
			if owner, exists := byPath[target]; exists {
				if len(matchingTaskDirectives(owner, "source", record.doc.path)) != 1 {
					issues = append(issues, Issue{record.doc.path, parent.span.line,
						fmt.Sprintf("parent %s does not have exactly one reciprocal task source",
							parent.value), false})
				}
			}
		}
	}
	return pairs, issues
}

func matchingTaskDirectives(records []directiveRecord, key, target string) []directiveRecord {
	var matches []directiveRecord
	for _, record := range records {
		if field, ok := record.fields[key]; ok && record.isTask() &&
			resolvedReferencePath(record.doc.path, field.value) == target {
			matches = append(matches, record)
		}
	}
	return matches
}

type textEdit struct {
	start, end int
	text       string
}

// repair plans targeted edits against the original-byte snapshots, writes them
// per document, and refreshes the affected parsed snapshots. UUID insertion and
// status promotion sharing a directive's closer are combined into one write.
func (s *Store) repair(docs []*parsedDocument) ([]Issue, error) {
	records := collectRecords(docs)
	edits := make(map[*parsedDocument][]textEdit)
	var fixed []Issue
	for _, record := range records {
		if !eligibleForUUID(record.directive) {
			continue
		}
		id, err := NewUUID()
		if err != nil {
			return nil, err
		}
		edits[record.doc] = append(edits[record.doc], textEdit{
			record.directive.insertAt, record.directive.insertAt,
			"uuid: " + id + record.directive.lineEnding})
		fixed = append(fixed, Issue{record.doc.path, record.directive.opening.line,
			"missing uuid", true})
	}
	pairs, _ := linkedTaskPairs(records)
	for _, promotion := range planStatusPromotions(pairs) {
		edit, line := promotion.textEdit()
		edits[promotion.record.doc] = append(edits[promotion.record.doc], edit)
		fixed = append(fixed, Issue{promotion.record.doc.path, line,
			"status promoted to " + statusNames[promotion.target], true})
	}
	for doc, documentEdits := range edits {
		if err := applyEdits(doc, documentEdits); err != nil {
			return nil, err
		}
	}
	for doc := range edits {
		if _, err := s.loadDocument(doc.path, true); err != nil {
			return nil, err
		}
	}
	return fixed, nil
}

// eligibleForUUID requires both source and uuid absent from the original
// directive; empty, invalid, and repeated occurrences still count as present.
func eligibleForUUID(directive *parsedDirective) bool {
	return !hasParsedField(directive, "source") && !hasParsedField(directive, "uuid")
}

type statusPromotion struct {
	record directiveRecord
	target int
}

func (p statusPromotion) textEdit() (textEdit, int) {
	name := statusNames[p.target]
	if field, ok := p.record.fields["status"]; ok {
		return textEdit{field.valueSpan.start, field.valueSpan.end, name}, field.span.line
	}
	directive := p.record.directive
	return textEdit{directive.insertAt, directive.insertAt,
		"status: " + name + directive.lineEnding}, directive.opening.line
}

// planStatusPromotions raises every member of each connected group of eligible
// reciprocal links to the group's maximum usable status. A directive that is
// already at the maximum receives no edit, so a second run makes no changes.
func planStatusPromotions(pairs [][2]directiveRecord) []statusPromotion {
	parent := make(map[*parsedDirective]*parsedDirective)
	ranks := make(map[*parsedDirective]directiveRecord)
	var find func(*parsedDirective) *parsedDirective
	find = func(directive *parsedDirective) *parsedDirective {
		if parent[directive] != directive {
			parent[directive] = find(parent[directive])
		}
		return parent[directive]
	}
	add := func(record directiveRecord) bool {
		if _, ok := record.statusRank(); !ok {
			return false
		}
		if _, seen := parent[record.directive]; !seen {
			parent[record.directive] = record.directive
			ranks[record.directive] = record
		}
		return true
	}
	for _, pair := range pairs {
		if add(pair[0]) && add(pair[1]) {
			parent[find(pair[0].directive)] = find(pair[1].directive)
		}
	}
	maxRank := make(map[*parsedDirective]int)
	for directive, record := range ranks {
		rank, _ := record.statusRank()
		if root := find(directive); rank > maxRank[root] {
			maxRank[root] = rank
		}
	}
	var promotions []statusPromotion
	for directive, record := range ranks {
		rank, _ := record.statusRank()
		if target := maxRank[find(directive)]; target != rank {
			promotions = append(promotions, statusPromotion{record, target})
		}
	}
	return promotions
}

// applyEdits rewrites a document from its original snapshot, combining
// insertions that share an offset while preserving surrounding bytes.
func applyEdits(doc *parsedDocument, edits []textEdit) error {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var output bytes.Buffer
	start := 0
	for _, edit := range edits {
		output.Write(doc.data[start:edit.start])
		output.WriteString(edit.text)
		start = edit.end
	}
	output.Write(doc.data[start:])
	// WriteFile retains the permissions of an existing file.
	if err := os.WriteFile(doc.path, output.Bytes(), 0644); err != nil {
		return fmt.Errorf("repair %s: %w", doc.path, err)
	}
	return nil
}
