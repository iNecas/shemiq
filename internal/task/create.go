package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iNecas/shemiq/internal/utils"
)

var slugCleanup = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// FindProjectDirectory returns the nearest .shemiq directory,
// or the path where one should be created if none exists
// above the invocation directory.
func FindProjectDirectory(cwd string) (string, error) {
	for dir := cwd; ; dir = filepath.Dir(dir) {
		projectDir := filepath.Join(dir, ".shemiq")
		info, err := os.Stat(projectDir)
		if err == nil && info.IsDir() {
			return projectDir, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect %s: %w", projectDir, err)
		}
		if parent := filepath.Dir(dir); parent == dir {
			return filepath.Join(cwd, ".shemiq"), nil
		}
	}
}

// CreateTopLevel writes a new top-level task document,
// discovering the project from the store's invocation
// directory independently of the configured read scope.
// It returns the absolute path to the created top-level.md.
func (s *Store) CreateTopLevel(
	title, description string,
) (string, error) {
	if strings.TrimSpace(description) == "" {
		return "", fmt.Errorf("description is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	if strings.ContainsAny(title, "\r\n") {
		return "", fmt.Errorf("title must be one line")
	}
	slug := taskSlug(title)
	if slug == "" {
		return "", fmt.Errorf(
			"title must contain an ASCII letter or digit")
	}

	projectDir, err := FindProjectDirectory(s.cwd)
	if err != nil {
		return "", err
	}
	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf(
			"resolve project directory: %w", err)
	}
	id, err := NewUUID()
	if err != nil {
		return "", err
	}
	tasksDir := filepath.Join(projectDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return "", fmt.Errorf(
			"create tasks directory %s: %w",
			tasksDir, err)
	}
	taskDir := filepath.Join(tasksDir, slug)
	if err := os.Mkdir(taskDir, 0755); err != nil {
		return "", fmt.Errorf(
			"create task directory %s: %w",
			taskDir, err)
	}
	path := filepath.Join(taskDir, "top-level.md")
	content := topLevelDocument(title, description, id)
	if err := os.WriteFile(
		path, []byte(content), 0644,
	); err != nil {
		return "", errors.Join(
			fmt.Errorf("write %s: %w", path, err),
			cleanupTask(path, taskDir))
	}

	if err := s.cacheCreated(path); err != nil {
		return path, fmt.Errorf(
			"created %s but failed to cache: %w",
			path, err)
	}
	return path, nil
}

// cacheCreated adds a newly created file to the store's
// task cache if the store is already loaded and the file
// is within scope. Does nothing if the store has not loaded.
func (s *Store) cacheCreated(path string) error {
	if !s.loaded {
		return nil
	}
	resolved, err := s.documentPath(path)
	if err != nil {
		return err
	}
	if !s.contains(resolved) {
		return nil
	}
	if _, err := s.loadTask(resolved, false); err != nil {
		return err
	}
	return nil
}

func taskSlug(title string) string {
	cleaned := slugCleanup.ReplaceAllString(title, "-")
	cleaned = strings.Trim(cleaned, "-")
	cleaned = strings.ToLower(cleaned)
	return cleaned
}

func topLevelDocument(
	title, description, id string,
) string {
	return fmt.Sprintf(utils.Dedent(`
		# %s
		:::shemiq
		type: top-level
		uuid: %s
		:::

		## Description

		%s

		## Context

		[TBD]

		## Interview

		[TBD]

		## Design

		[TBD]

		## Current status

		[TBD]

		## Tasks

		[TBD]
		`), title, id, description)
}

func cleanupTask(path, taskDir string) error {
	var failures []error
	if err := os.Remove(path); err != nil &&
		!os.IsNotExist(err) {
		failures = append(failures,
			fmt.Errorf("remove %s: %w", path, err))
	}
	if err := os.Remove(taskDir); err != nil {
		failures = append(failures,
			fmt.Errorf("remove %s: %w", taskDir, err))
	}
	return errors.Join(failures...)
}
