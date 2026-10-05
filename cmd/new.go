package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newTaskCommand(result io.Writer) *cobra.Command {
	taskCommand := &cobra.Command{Use: "task", Short: "Manage tasks"}
	var title string
	newCommand := &cobra.Command{
		Use:   "new <description...>",
		Short: "Create a top-level task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := strings.TrimSpace(strings.Join(args, " "))
			if description == "" {
				return fmt.Errorf("description is required")
			}
			if !cmd.Flags().Changed("title") {
				var err error
				title, err = readTitle(cmd.ErrOrStderr(), cmd.InOrStdin())
				if err != nil {
					return err
				}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			projectDir, err := task.FindProjectDirectory(cwd)
			if err != nil {
				return err
			}
			path, err := task.CreateTopLevelTask(projectDir, description, title)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(result, path)
			return err
		},
	}
	newCommand.Flags().StringVar(&title, "title", "", "Title for the new task (prompts if omitted)")
	taskCommand.AddCommand(newCommand)
	return taskCommand
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
