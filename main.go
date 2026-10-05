package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand(os.Stdout).Execute(); err != nil {
		os.Exit(1) // Cobra has already printed the error and usage to stderr.
	}
}

func newRootCommand(result io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "shemiq",
		Short: "Manage Shemiq tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected 'task new <description...>'")
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetOut(os.Stderr) // Cobra writes error usage to OutOrStdout.
	root.AddCommand(newTaskCommand(result))
	return root
}

func newTaskCommand(result io.Writer) *cobra.Command {
	task := &cobra.Command{Use: "task", Short: "Manage tasks"}
	task.AddCommand(&cobra.Command{
		Use:   "new <description...>",
		Short: "Create a top-level task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := strings.TrimSpace(strings.Join(args, " "))
			if description == "" {
				return fmt.Errorf("description is required")
			}
			title, err := readTitle(cmd.ErrOrStderr(), cmd.InOrStdin())
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			projectDir, err := findProjectDirectory(cwd)
			if err != nil {
				return err
			}
			path, err := createTopLevelTask(projectDir, description, title)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(result, path)
			return err
		},
	})
	return task
}

func readTitle(prompt io.Writer, input io.Reader) (string, error) {
	if _, err := fmt.Fprint(prompt, "Title: "); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read title: %w", err)
	}
	title := strings.TrimSpace(line)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	return title, nil
}
