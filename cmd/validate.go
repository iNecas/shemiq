package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	var fix bool
	command := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate Shemiq metadata in Markdown files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			store, err := task.NewStore(cwd, path)
			if err != nil {
				return err
			}
			issues, err := store.Validate(fix)
			if err != nil {
				return err
			}
			unresolved := false
			for _, issue := range issues {
				label := ""
				if issue.Fixed {
					label = "fixed: "
				} else {
					unresolved = true
				}
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s:%d: %s%s\n", issue.Path, issue.Line, label, issue.Message); err != nil {
					return err
				}
			}
			if unresolved {
				cmd.Root().SilenceErrors = true
				cmd.Root().SilenceUsage = true
				return errors.New("validation failed")
			}
			return nil
		},
	}
	command.Flags().BoolVar(&fix, "fix", false, "Insert missing UUIDs and promote mismatched statuses in linked tasks")
	return command
}
