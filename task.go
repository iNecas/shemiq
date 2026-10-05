package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createTopLevelTask writes a task independently of how its title was obtained.
// projectDir is the path to the project's .shemiq directory.
func createTopLevelTask(projectDir, description, title string) (string, error) {
	if strings.TrimSpace(description) == "" {
		return "", fmt.Errorf("description is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	slug := taskSlug(title)
	if slug == "" {
		return "", fmt.Errorf("title must contain an ASCII letter or digit")
	}

	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
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
	if err := os.WriteFile(path, []byte(topLevelDocument(title, description)), 0644); err != nil {
		return "", errors.Join(fmt.Errorf("write %s: %w", path, err), cleanupTask(path, taskDir))
	}
	return path, nil
}

// findProjectDirectory returns the nearest .shemiq directory, or the path
// where one should be created if none exists above the invocation directory.
func findProjectDirectory(cwd string) (string, error) {
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

func topLevelDocument(title, description string) string {
	return fmt.Sprintf(`# %s
:::shemiq
type: top-level
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
`, title, description)
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
