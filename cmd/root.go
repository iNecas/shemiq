package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute runs the CLI with the supplied arguments and streams.
func Execute(input io.Reader, result, diagnostics io.Writer, args []string) error {
	root := newRootCommand(result)
	root.SetIn(input)
	root.SetOut(diagnostics) // Cobra writes error usage to OutOrStdout.
	root.SetErr(diagnostics)
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand(result io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "shemiq",
		Short: "Manage Shemiq tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected 'shemiq new <description...>'")
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newNewCommand(result), newValidateCommand())
	return root
}
