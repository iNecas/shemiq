package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/iNecas/shemiq/internal/console"
	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newTreeCommand(result io.Writer) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "tree [dir]",
		Short: "Show the task hierarchy",
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
				cmd.Root().SilenceUsage = true
				return err
			}
			doc, err := store.Tree()
			if err != nil {
				cmd.Root().SilenceUsage = true
				var ve *task.ValidationError
				if errors.As(err, &ve) {
					// Print each located finding on stderr.
					for _, issue := range ve.Issues {
						fmt.Fprintf(cmd.ErrOrStderr(), "%s:%d: %s\n",
							issue.Path, issue.Line, issue.Message)
					}
					cmd.Root().SilenceErrors = true
					return errors.New("validation failed")
				}
				return err
			}

			var output string
			if jsonOutput {
				data, err := json.MarshalIndent(doc, "", "  ")
				if err != nil {
					cmd.Root().SilenceUsage = true
					return fmt.Errorf("encode JSON: %w", err)
				}
				output = string(data) + "\n"
			} else {
				color := console.ColorEnabled(result)
				output = console.RenderTree(doc, color)
			}

			if _, err := fmt.Fprint(result, output); err != nil {
				cmd.Root().SilenceUsage = true
				return err
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON instead of human-readable tree")
	return command
}
