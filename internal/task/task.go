package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iNecas/shemiq/internal/utils"
)

type TaskType string

const (
	TypeTopLevel TaskType = "top-level"
	TypeTask     TaskType = "task"
)

// Task represents a top-level document or an entry in its task list. Subtasks
// are read from the parent document; their own files may not exist yet, so
// Path and ProjectRoot are only set on the top-level task.
type Task struct {
	Type        TaskType
	Title       string
	Status      string
	Path        string
	ProjectRoot string
	Subtasks    []Task
}

// FindProjectDirectory returns the nearest .shemiq directory, or the path
// where one should be created if none exists above the invocation directory.
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

// CreateTopLevelTask writes a task independently of how its title was obtained.
// projectDir is the path to the project's .shemiq directory.
func CreateTopLevelTask(projectDir, description, title string) (string, error) {
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
		return "", fmt.Errorf("title must contain an ASCII letter or digit")
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
	}
	id, err := NewUUID()
	if err != nil {
		return "", err
	}
	tasksDir := filepath.Join(projectDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return "", fmt.Errorf("create tasks directory %s: %w", tasksDir, err)
	}
	taskDir := filepath.Join(tasksDir, slug)
	if err := os.Mkdir(taskDir, 0755); err != nil {
		return "", fmt.Errorf("create task directory %s: %w", taskDir, err)
	}
	path := filepath.Join(taskDir, "top-level.md")
	if err := os.WriteFile(path, []byte(topLevelDocument(title, description, id)), 0644); err != nil {
		return "", errors.Join(fmt.Errorf("write %s: %w", path, err), cleanupTask(path, taskDir))
	}
	return path, nil
}

func taskSlug(title string) string {
	var slug strings.Builder
	separator := false
	for i := 0; i < len(title); i++ {
		c := title[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			if separator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteByte(c)
			separator = false
		} else {
			separator = true
		}
	}
	return slug.String()
}

func topLevelDocument(title, description, id string) string {
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
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		failures = append(failures, fmt.Errorf("remove %s: %w", path, err))
	}
	if err := os.Remove(taskDir); err != nil {
		failures = append(failures, fmt.Errorf("remove %s: %w", taskDir, err))
	}
	return errors.Join(failures...)
}
