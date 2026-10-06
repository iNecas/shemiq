package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ReadTopLevelTask accepts a task directory or its top-level.md, including in archives.
func ReadTopLevelTask(cwd, path string) (*Task, error) {
	return readTopLevelTask(cwd, path, true)
}

// ListUnfinishedTopLevelTasks reads immediate active task directories in name
// order. A missing or unroutable top-level document fails the whole listing.
func ListUnfinishedTopLevelTasks(cwd string) ([]*Task, error) {
	projectDir, err := FindProjectDirectory(cwd)
	if err != nil {
		return nil, err
	}
	if err := requireDirectory(projectDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no .shemiq directory found above %s", cwd)
		}
		return nil, err
	}
	tasksDir := filepath.Join(projectDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", tasksDir, err)
	}
	var tasks []*Task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Only titles and top-level statuses are needed for the picker. Do not
		// let an unchosen task's subtask metadata block selection.
		t, err := readTopLevelTask(cwd, filepath.Join(tasksDir, entry.Name()), false)
		if err != nil {
			return nil, err
		}
		if t.Status != "done" {
			tasks = append(tasks, t)
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no unfinished tasks in %s", tasksDir)
	}
	return tasks, nil
}

func readTopLevelTask(cwd, path string, withSubtasks bool) (*Task, error) {
	path, err := topLevelPath(cwd, path)
	if err != nil {
		return nil, err
	}
	projectRoot, err := projectRootForDocument(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	title, top, subtasks, err := parseTopLevelOutline(path, data)
	if err != nil {
		return nil, err
	}
	if err := requireDirectiveType(top, string(TypeTopLevel)); err != nil {
		return nil, fmt.Errorf("%s: top-level: %w", path, err)
	}
	status, err := directiveStatus(top)
	if err != nil {
		return nil, fmt.Errorf("%s: top-level: %w", path, err)
	}
	result := &Task{Type: TypeTopLevel, Path: path, ProjectRoot: projectRoot, Title: title, Status: status}
	if status == "refined" && withSubtasks {
		for _, sub := range subtasks {
			if sub.title == "" {
				return nil, fmt.Errorf("%s: empty subtask title", path)
			}
			if err := requireDirectiveType(sub.directive, string(TypeTask)); err != nil {
				return nil, fmt.Errorf("%s: subtask %q: %w", path, sub.title, err)
			}
			substatus, err := directiveStatus(sub.directive)
			if err != nil {
				return nil, fmt.Errorf("%s: subtask %q: %w", path, sub.title, err)
			}
			result.Subtasks = append(result.Subtasks, Task{Type: TypeTask, Title: sub.title, Status: substatus})
		}
	}
	return result, nil
}

func topLevelPath(cwd, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve task path: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.IsDir() {
		path = filepath.Join(path, "top-level.md")
	} else if filepath.Base(path) != "top-level.md" {
		return "", fmt.Errorf("expected top-level.md: %s", path)
	}
	info, err = os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular top-level.md: %s", path)
	}
	// Check containment on the resolved path, also used by the later launcher.
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	return path, nil
}

// The target itself must live below .shemiq, not merely next to one.
func projectRootForDocument(path string) (string, error) {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if filepath.Base(dir) == ".shemiq" {
			return filepath.Dir(dir), nil
		}
		if filepath.Dir(dir) == dir {
			return "", fmt.Errorf("top-level.md is not inside a .shemiq project: %s", path)
		}
	}
}
