package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/iNecas/shemiq/internal/agent"
	"github.com/iNecas/shemiq/internal/console"
	"github.com/iNecas/shemiq/internal/task"
	"github.com/spf13/cobra"
)

func newRefineCommand(result io.Writer, launcher agent.Launcher) *cobra.Command {
	var subtask string
	command := &cobra.Command{
		Use:   "refine [path]",
		Short: "Refine a task interactively in Pi",
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
			ui := console.New(cmd.InOrStdin(), cmd.ErrOrStderr())
			resolved, err := resolveRefinement(cwd, path, subtask, cmd.Flags().Changed("subtask"), ui)
			if err != nil {
				return err
			}
			return launcher.Launch(resolved, agent.Streams{
				Input: cmd.InOrStdin(), Output: result, Diagnostics: cmd.ErrOrStderr(),
			})
		},
	}
	command.Flags().StringVar(&subtask, "subtask", "", "Exact subtask title to refine")
	return command
}

func resolveRefinement(cwd, path, subtask string, hasSubtask bool, ui *console.Console) (agent.Request, error) {
	var selected *task.Task
	if path != "" {
		var err error
		selected, err = task.ReadTopLevelTask(cwd, path)
		if err != nil {
			return agent.Request{}, err
		}
	} else {
		choices, err := task.ListUnfinishedTopLevelTasks(cwd)
		if err != nil {
			return agent.Request{}, err
		}
		labels := make([]string, len(choices))
		for i, choice := range choices {
			labels[i] = choice.Title
		}
		i, err := ui.Pick("Select a task:", labels)
		if err != nil {
			return agent.Request{}, err
		}
		selected = choices[i]
	}

	resolved := agent.Request{DocumentPath: selected.Path, ProjectRoot: selected.ProjectRoot}
	switch selected.Status {
	case "new":
		if hasSubtask {
			return agent.Request{}, fmt.Errorf("cannot specify --subtask for a new top-level task")
		}
		resolved.Flow = agent.RefineTopLevel
	case "refined":
		// A top-level picker only needs titles/statuses; read subtasks of the
		// chosen document so unrelated unfinished tasks cannot block routing.
		if path == "" {
			var err error
			selected, err = task.ReadTopLevelTask(cwd, selected.Path)
			if err != nil {
				return agent.Request{}, err
			}
		}
		title, err := chooseSubtask(selected.Subtasks, subtask, hasSubtask, ui)
		if err != nil {
			return agent.Request{}, err
		}
		resolved.Flow, resolved.SubtaskTitle = agent.RefineSubtask, title
	case "done":
		return agent.Request{}, fmt.Errorf("top-level task is done: %s", selected.Path)
	default:
		return agent.Request{}, fmt.Errorf("unsupported top-level status: %s", selected.Status)
	}
	return resolved, nil
}

func chooseSubtask(subtasks []task.Task, title string, exact bool, ui *console.Console) (string, error) {
	if exact {
		var matches []task.Task
		for _, sub := range subtasks {
			if sub.Title == title {
				matches = append(matches, sub)
			}
		}
		if len(matches) == 0 {
			return "", fmt.Errorf("subtask not found: %q", title)
		}
		if len(matches) != 1 {
			return "", fmt.Errorf("ambiguous subtask title: %q", title)
		}
		if matches[0].Status != "new" {
			return "", fmt.Errorf("subtask %q is not new", title)
		}
		return title, nil
	}
	var labels []string
	for _, sub := range subtasks {
		if sub.Status == "new" {
			labels = append(labels, sub.Title)
		}
	}
	if len(labels) == 0 {
		return "", fmt.Errorf("no new subtasks available")
	}
	i, err := ui.Pick("Select a subtask:", labels)
	if err != nil {
		return "", err
	}
	// Even interactive selection cannot identify duplicate titles uniquely.
	selected := labels[i]
	for j, label := range labels {
		if j != i && label == selected {
			return "", fmt.Errorf("ambiguous subtask title: %q", selected)
		}
	}
	return selected, nil
}
