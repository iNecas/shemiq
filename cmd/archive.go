package cmd

import (
	"fmt"
	"io"
	"os"

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
			path, err := task.ArchiveTask(cwd, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(result, path)
			return err
		},
	}
}
