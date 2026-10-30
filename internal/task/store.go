package task

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store lazily loads a project, directory, or Markdown file scope.
// Directory scopes discover only top-level.md entry points; explicitly
// selected files are entry points regardless of name. Loading follows
// usable source:/parent: references transitively, so the cached task
// set is the closure of discovered documents. Tasks are snapshots for
// the store's lifetime; queries never follow metadata references.
// Construction does not inspect the filesystem or require an existing
// project.
type Store struct {
	cwd        string
	selection  string
	scope      *storeScope
	tasks      map[string]*Task
	aliases    map[string]string
	discovered map[string]bool
	loaded     bool
}

type storeScope struct {
	path      string // Resolved file or directory.
	directory bool
}

// entryPointName is the only filename discovered as a task
// entry point when loading a directory scope.
const entryPointName = "top-level.md"

func NewStore(cwd, path string) (*Store, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve invocation directory: %w", err)
	}
	if path != "" {
		path = invocationPath(cwd, path)
	}
	return &Store{
		cwd: cwd, selection: path,
		tasks:      make(map[string]*Task),
		aliases:    make(map[string]string),
		discovered: make(map[string]bool),
	}, nil
}

// TaskByPath returns the requested document's primary task,
// including semantic issues. A directory argument selects
// top-level.md; other filenames are valid. A query for a
// not-yet-cached document in scope opts it into the task set;
// TopLevelTasks still applies its discovery naming rule. The
// complete configured scope is loaded before the first query.
func (s *Store) TaskByPath(path string) (Task, error) {
	if err := s.load(); err != nil {
		return Task{}, err
	}
	path = invocationPath(s.cwd, path)
	if _, cached := s.aliases[path]; !cached {
		info, err := os.Stat(path)
		if err != nil {
			return Task{}, fmt.Errorf(
				"inspect %s: %w", path, err)
		}
		if info.IsDir() {
			path = filepath.Join(path, "top-level.md")
		}
	}
	if !isMarkdown(path) {
		return Task{}, fmt.Errorf(
			"not a Markdown file: %s", path)
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return Task{}, err
	}
	if !s.contains(resolved) {
		return Task{}, fmt.Errorf(
			"document is outside store scope %s: %s",
			s.scope.path, path)
	}
	t := s.tasks[resolved]
	if t == nil {
		// An explicit query opts the document into the task set.
		t, err = s.loadDiscovered(resolved)
		if err != nil {
			return Task{}, err
		}
	}
	return *t, nil
}

// TopLevelTasks returns discoverable top-level tasks in path
// order, without filtering statuses. Directory scopes report
// only top-level.md documents; an explicitly selected file is
// a candidate regardless of its name. Selection uses cached
// snapshots and never rescans the filesystem.
func (s *Store) TopLevelTasks() ([]Task, error) {
	if err := s.load(); err != nil {
		return nil, err
	}
	var paths []string
	for path := range s.discovered {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var tasks []Task
	for _, path := range paths {
		t := s.tasks[path]
		if t == nil || t.Type != TypeTopLevel {
			continue
		}
		// Directory discovery recognizes only top-level.md;
		// the explicitly selected scope file keeps its name.
		if s.scope.directory &&
			filepath.Base(path) != entryPointName {
			continue
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

// load initializes the store's scope, discovers entry points,
// and completes the transitive reference closure. Idempotent
// after success; a failed call can be retried without
// duplicating task records.
func (s *Store) load() error {
	if s.loaded {
		return nil
	}
	if err := s.resolveScope(); err != nil {
		return err
	}
	if !s.scope.directory {
		if _, err := s.loadDiscovered(s.scope.path); err != nil {
			return err
		}
	} else {
		err := filepath.WalkDir(
			s.scope.path,
			func(path string, entry fs.DirEntry,
				err error) error {
				if err != nil {
					return err
				}
				if entry.Type().IsRegular() &&
					filepath.Base(path) == entryPointName {
					if _, err := s.loadDiscovered(
						path); err != nil {
						return err
					}
				}
				return nil
			})
		if err != nil {
			return fmt.Errorf(
				"load scope %s: %w", s.scope.path, err)
		}
	}
	if _, err := s.reachableTasks(); err != nil {
		return err
	}
	s.loaded = true
	return nil
}

// loadDiscovered loads a deliberately selected document and
// records it in the discovered set: directory entry points,
// explicit scope files, and explicit TaskByPath queries all
// share this membership regardless of their source.
func (s *Store) loadDiscovered(path string) (*Task, error) {
	t, err := s.loadTask(path, false)
	if err != nil {
		return nil, err
	}
	s.discovered[t.document.path] = true
	return t, nil
}

// reachableTasks returns discovered documents and every task
// reachable from them through usable source/parent references,
// in path order. Non-existent, directory, and non-Markdown
// targets are left for validation.
func (s *Store) reachableTasks() ([]*Task, error) {
	var seeds []string
	for path := range s.discovered {
		seeds = append(seeds, path)
	}
	sort.Strings(seeds)

	visited := make(map[string]bool)
	byPath := make(map[string]*Task)
	var queue []string
	for _, path := range seeds {
		t := s.tasks[path]
		if t == nil || visited[path] {
			continue
		}
		visited[path] = true
		byPath[path] = t
		queue = append(queue,
			taskReferenceTargets(t)...)
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

func (s *Store) resolveScope() error {
	if s.scope != nil {
		return nil
	}
	path := s.selection
	if path == "" {
		project, err := FindProjectDirectory(s.cwd)
		if err != nil {
			return err
		}
		info, err := os.Stat(project)
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"no .shemiq directory found above %s",
				s.cwd)
		}
		if err != nil {
			return fmt.Errorf(
				"inspect %s: %w", project, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", project)
		}
		path = filepath.Join(project, "tasks")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() &&
		(!info.Mode().IsRegular() || !isMarkdown(path)) {
		return fmt.Errorf("not a Markdown file: %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}
	s.scope = &storeScope{
		path: resolved, directory: info.IsDir(),
	}
	return nil
}

func (s *Store) contains(path string) bool {
	if !s.scope.directory {
		return path == s.scope.path
	}
	relative, err := filepath.Rel(s.scope.path, path)
	return err == nil && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// loadTask loads a parsed snapshot, converts its primary task
// with embedded subtasks, and caches the result. It is
// independent of query scope so validation can follow
// references and refresh after repairs.
func (s *Store) loadTask(
	path string, refresh bool,
) (*Task, error) {
	path = invocationPath(s.cwd, path)
	if !isMarkdown(path) {
		return nil, fmt.Errorf("not a Markdown file: %s", path)
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return nil, err
	}
	cached := s.tasks[resolved]
	if cached != nil && !refresh {
		return cached, nil
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a Markdown file: %s", path)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", resolved, err)
	}
	doc, err := parseMarkdownDocument(resolved, data)
	if err != nil {
		return nil, err
	}
	// A failed read/parse leaves the previous snapshot intact.
	result := documentTask(doc)
	s.tasks[resolved] = &result
	return &result, nil
}

// documentTask converts a parsed document into a primary
// task with embedded subtasks.
func documentTask(doc *parsedDocument) Task {
	var primary *parsedSection
	for _, section := range doc.root.children {
		if section.level == 1 {
			primary = section
			break
		}
	}
	if primary == nil {
		return Task{
			Path:        doc.path,
			ProjectRoot: documentProjectRoot(doc.path),
			document:    doc,
			Issues: []Issue{{Path: doc.path, Line: 1,
				Message: "missing primary task " +
					"heading/directive pair"}},
		}
	}
	result := sectionTask(doc, primary, false)
	for _, section := range primary.children {
		if section.level != 2 ||
			section.title != "Tasks" {
			continue
		}
		for _, entry := range section.children {
			if entry.level == 3 {
				result.Subtasks = append(
					result.Subtasks,
					sectionTask(doc, entry, true))
			}
		}
	}
	return result
}

func (s *Store) documentPath(path string) (string, error) {
	if resolved, ok := s.aliases[path]; ok {
		return resolved, nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	s.aliases[path] = resolved
	s.aliases[resolved] = resolved
	return resolved, nil
}

func invocationPath(cwd, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	return filepath.Clean(path)
}

// Ownership follows the resolved defining document, not the caller or a source
// entry's target. Generic documents outside a project have no project root.
func documentProjectRoot(path string) string {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if filepath.Base(dir) == ".shemiq" {
			return filepath.Dir(dir)
		}
		if filepath.Dir(dir) == dir {
			return ""
		}
	}
}

// Resolve existing symlink prefixes even for an uncreated source target. Errors
// do not turn reference validity into a conversion/loading requirement.
func resolvedReferencePath(path, value string) string {
	path = invocationPath(filepath.Dir(path), value)
	prefix := path
	var suffix []string
	for {
		if resolved, err := filepath.EvalSymlinks(prefix); err == nil {
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return resolved
		}
		parent := filepath.Dir(prefix)
		if parent == prefix {
			return path
		}
		suffix = append(suffix, filepath.Base(prefix))
		prefix = parent
	}
}
