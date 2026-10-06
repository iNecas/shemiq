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

func resolveRefinement(
	cwd, path, subtask string, hasSubtask bool, ui *console.Console,
) (agent.Request, error) {
	store, err := task.NewStore(cwd, path)
	if err != nil {
		return agent.Request{}, err
	}
	var selected task.Task
	if path != "" {
		selected, err = store.TaskByPath(path)
		if err != nil {
			return agent.Request{}, err
		}
	} else {
		candidates, err := store.TopLevelTasks()
		if err != nil {
			return agent.Request{}, err
		}
		var choices []task.Task
		var labels []string
		for _, choice := range candidates {
			if choice.Status != "done" {
				choices = append(choices, choice)
				labels = append(labels, choice.Title)
			}
		}
		if len(choices) == 0 {
			return agent.Request{}, fmt.Errorf("no unfinished tasks")
		}
		i, err := ui.Pick("Select a task:", labels)
		if err != nil {
			return agent.Request{}, err
		}
		selected = choices[i]
	}

	// The store is generic; project containment and routing are workflow policy.
	// ProjectRoot comes from the resolved document, including explicit targets
	// in archives or other projects. Conversion issues belong to validation.
	if selected.ProjectRoot == "" {
		return agent.Request{}, fmt.Errorf(
			"task document is not inside a .shemiq project: %s", selected.Path,
		)
	}
	if selected.Type != task.TypeTopLevel {
		return agent.Request{}, fmt.Errorf("expected type: top-level: %s", selected.Path)
	}
	if selected.Title == "" {
		return agent.Request{}, fmt.Errorf("empty top-level task title: %s", selected.Path)
	}
	resolved := agent.Request{DocumentPath: selected.Path, ProjectRoot: selected.ProjectRoot}
	switch selected.Status {
	case "new":
		if hasSubtask {
			return agent.Request{}, fmt.Errorf("cannot specify --subtask for a new top-level task")
		}
		resolved.Flow = agent.RefineTopLevel
	case "refined":
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

func chooseSubtask(
	subtasks []task.Task, title string, exact bool, ui *console.Console,
) (string, error) {
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
		if matches[0].Title == "" {
			return "", fmt.Errorf("empty subtask title")
		}
		if matches[0].Type != task.TypeTask {
			return "", fmt.Errorf("subtask %q is not type: task", title)
		}
		if matches[0].Status != "new" {
			return "", fmt.Errorf("subtask %q is not new", title)
		}
		return title, nil
	}
	var labels []string
	for _, sub := range subtasks {
		if sub.Type == task.TypeTask && sub.Status == "new" && sub.Title != "" {
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
