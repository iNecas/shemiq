package task

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Store lazily loads a project, directory, or Markdown file scope. Documents
// are snapshots for its lifetime; queries never follow metadata references.
// Construction does not inspect the filesystem or require an existing project.
type Store struct {
	cwd       string
	selection string
	scope     *storeScope
	documents map[string]*parsedDocument
	aliases   map[string]string
}

type storeScope struct {
	path      string // Resolved file or directory.
	directory bool
}

func NewStore(cwd, path string) (*Store, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve invocation directory: %w", err)
	}
	if path != "" {
		path = invocationPath(cwd, path)
	}
	return &Store{
		cwd: cwd, selection: path,
		documents: make(map[string]*parsedDocument),
		aliases:   make(map[string]string),
	}, nil
}

// TaskByPath returns the requested document's primary task, including semantic
// issues. A directory argument selects top-level.md; other filenames are valid.
func (s *Store) TaskByPath(path string) (Task, error) {
	if err := s.resolveScope(); err != nil {
		return Task{}, err
	}
	path = invocationPath(s.cwd, path)
	if _, cached := s.aliases[path]; !cached {
		info, err := os.Stat(path)
		if err != nil {
			return Task{}, fmt.Errorf("inspect %s: %w", path, err)
		}
		if info.IsDir() {
			path = filepath.Join(path, "top-level.md")
		}
	}
	if !isMarkdown(path) {
		return Task{}, fmt.Errorf("not a Markdown file: %s", path)
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return Task{}, err
	}
	// Scope is enforced independently of the cache: validation may later load
	// references outside this selection through loadDocument.
	if !s.contains(resolved) {
		return Task{}, fmt.Errorf("document is outside store scope %s: %s", s.scope.path, path)
	}
	return s.loadDocument(path, false)
}

// TopLevelTasks returns candidates in path order without filtering statuses or
// requiring valid metadata. Default project scope uses conventional documents
// in immediate active task directories; explicit directories scan recursively.
func (s *Store) TopLevelTasks() ([]Task, error) {
	if err := s.resolveScope(); err != nil {
		return nil, err
	}
	var paths []string
	if s.selection == "" {
		entries, err := os.ReadDir(s.scope.path)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", s.scope.path, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				paths = append(paths, filepath.Join(s.scope.path, entry.Name(), "top-level.md"))
			}
		}
	} else if s.scope.directory {
		err := filepath.WalkDir(s.scope.path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && isMarkdown(path) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", s.scope.path, err)
		}
	} else {
		paths = []string{s.selection}
	}
	var tasks []Task
	for _, path := range paths {
		t, err := s.TaskByPath(path)
		if err != nil {
			return nil, err
		}
		if s.selection == "" || t.Type == TypeTopLevel {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
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
			return fmt.Errorf("no .shemiq directory found above %s", s.cwd)
		}
		if err != nil {
			return fmt.Errorf("inspect %s: %w", project, err)
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
	if !info.IsDir() && (!info.Mode().IsRegular() || !isMarkdown(path)) {
		return fmt.Errorf("not a Markdown file: %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}
	s.scope = &storeScope{path: resolved, directory: info.IsDir()}
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

// loadDocument loads a parsed snapshot and converts its primary task. It is
// independent of query scope so validation can follow references and refresh
// after repairs. All directives remain available through the task's document;
// query-structure findings are separate from directive metadata checks.
func (s *Store) loadDocument(path string, refresh bool) (Task, error) {
	path = invocationPath(s.cwd, path)
	if !isMarkdown(path) {
		return Task{}, fmt.Errorf("not a Markdown file: %s", path)
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return Task{}, err
	}
	doc := s.documents[resolved]
	if doc == nil || refresh {
		info, err := os.Stat(resolved)
		if err != nil {
			return Task{}, fmt.Errorf("inspect %s: %w", resolved, err)
		}
		if !info.Mode().IsRegular() {
			return Task{}, fmt.Errorf("not a Markdown file: %s", path)
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return Task{}, fmt.Errorf("read %s: %w", resolved, err)
		}
		doc, err = parseMarkdownDocument(resolved, data)
		if err != nil {
			return Task{}, err
		}
		// A failed read/parse leaves the previous snapshot intact.
		s.documents[resolved] = doc
	}

	var primary *parsedSection
	for _, section := range doc.root.children {
		if section.level == 1 {
			primary = section
			break
		}
	}
	if primary == nil {
		return Task{
			Path: doc.path, ProjectRoot: documentProjectRoot(doc.path), document: doc,
			Issues: []Issue{{Path: doc.path, Line: 1,
				Message: "missing primary task heading/directive pair"}},
		}, nil
	}
	result := sectionTask(doc, primary, false)
	for _, section := range primary.children {
		if section.level != 2 || section.title != "Tasks" {
			continue
		}
		for _, entry := range section.children {
			if entry.level == 3 && adjacentDirective(doc, entry) != nil {
				result.Subtasks = append(result.Subtasks, sectionTask(doc, entry, true))
			}
		}
	}
	return result, nil
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
