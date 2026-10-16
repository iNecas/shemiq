package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newNewSubtaskCommand(result io.Writer) *cobra.Command {
	var title, after string
	cmd := &cobra.Command{
		Use:   "new-subtask <parent-path> [--title \"Title\"] [--after \"Sibling\"] <description...>",
		Short: "Insert a subtask into a top-level task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parentArg := args[0]
			description := strings.TrimSpace(strings.Join(args[1:], " "))
			if description == "" {
				return fmt.Errorf("description is required")
			}

			// Resolve parent path: directory → top-level.md.
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			parentPath := parentArg
			if !filepath.IsAbs(parentPath) {
				parentPath = filepath.Join(cwd, parentPath)
			}
			parentPath = filepath.Clean(parentPath)
			info, err := os.Stat(parentPath)
			if err != nil {
				return fmt.Errorf("inspect parent: %w", err)
			}
			if info.IsDir() {
				parentPath = filepath.Join(parentPath, "top-level.md")
			}

			// Prompt for title before Store interaction.
			if !cmd.Flags().Changed("title") {
				title, err = readTitle(
					cmd.ErrOrStderr(), cmd.InOrStdin())
				if err != nil {
					return err
				}
			}

			store, err := task.NewStore(cwd, "")
			if err != nil {
				return err
			}
			path, err := store.CreateSubtask(parentPath, title, description, after)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(result, path)
			return err
		},
	}
	cmd.Flags().StringVar(
		&title, "title", "", "Title for the new subtask (prompts if omitted)")
	cmd.Flags().StringVar(
		&after, "after", "", "Insert after this sibling (exact title, case-sensitive)")
	return cmd
}
