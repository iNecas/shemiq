package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newArchiveCommand(result io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <path>",
		Short: "Move a task directory into the dated archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			path, err := archiveTask(cwd, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(result, path)
			return err
		},
	}
}

// archiveTask moves one task directory from the nearest
// existing project's tasks directory into its archive,
// without reading or changing its contents.
func archiveTask(cwd, path string) (string, error) {
	projectDir, err := task.FindProjectDirectory(cwd)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(projectDir); os.IsNotExist(err) {
		return "", fmt.Errorf(
			"no .shemiq directory found above %s", cwd)
	} else if err != nil {
		return "", fmt.Errorf(
			"inspect %s: %w", projectDir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf(
			"not a directory: %s", projectDir)
	}

	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf(
			"resolve project directory: %w", err)
	}
	taskDir, err := selectArchiveTask(cwd, path, filepath.Join(projectDir, "tasks"))
	if err != nil {
		return "", err
	}
	archiveDir := filepath.Join(projectDir, "archive")
	if err := os.Mkdir(archiveDir, 0755); err != nil &&
		!os.IsExist(err) {
		return "", fmt.Errorf(
			"create archive directory %s: %w",
			archiveDir, err)
	}
	if err := requireDirectory(archiveDir); err != nil {
		return "", err
	}
	destination := filepath.Join(archiveDir,
		time.Now().UTC().Format("2006-01-02")+
			"-"+filepath.Base(taskDir))
	if _, err := os.Lstat(destination); err == nil {
		return "", fmt.Errorf(
			"archive destination already exists: %s",
			destination)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf(
			"inspect %s: %w", destination, err)
	}
	if err := os.Rename(taskDir, destination); err != nil {
		return "", fmt.Errorf(
			"move %s to %s: %w",
			taskDir, destination, err)
	}
	return destination, nil
}

func selectArchiveTask(cwd, path, tasksDir string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	path = filepath.Clean(path)
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Mode().IsRegular() &&
		filepath.Base(path) == "top-level.md" {
		path = filepath.Dir(path)
	} else if !info.IsDir() {
		return "", fmt.Errorf(
			"not a task directory or top-level.md "+
				"file: %s", path)
	}
	if filepath.Dir(path) != tasksDir {
		return "", fmt.Errorf(
			"task must be an immediate child of %s: %s",
			tasksDir, path)
	}
	if err := requireDirectory(tasksDir); err != nil {
		return "", err
	}
	if err := requireDirectory(path); err != nil {
		return "", err
	}
	return path, nil
}

// requireDirectory uses Lstat to prevent symlinked entries
// from escaping the selected project during the move.
func requireDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}
