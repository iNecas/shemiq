package cmd

import (
	"fmt"
	"io"

	"github.com/iNecas/shemiq/internal/agent"
	"github.com/spf13/cobra"
)

// Execute runs the CLI with the supplied arguments and streams.
func Execute(input io.Reader, result, diagnostics io.Writer, args []string) error {
	return execute(input, result, diagnostics, args, agent.Pi{})
}

// execute keeps launcher injection local to command orchestration and tests.
func execute(input io.Reader, result, diagnostics io.Writer, args []string, launcher agent.Launcher) error {
	root := newRootCommand(result, launcher)
	root.SetIn(input)
	root.SetOut(diagnostics) // Cobra writes error usage to OutOrStdout.
	root.SetErr(diagnostics)
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand(result io.Writer, launcher agent.Launcher) *cobra.Command {
	root := &cobra.Command{
		Use:   "shemiq",
		Short: "Manage Shemiq tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected 'shemiq new <description...>'")
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		newNewCommand(result),
		newValidateCommand(),
		newArchiveCommand(result),
		newRefineCommand(result, launcher),
		newImplementCommand(result, launcher),
	)
	return root
}
