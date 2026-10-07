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

func newImplementCommand(
	result io.Writer, launcher agent.Launcher,
) *cobra.Command {
	var subtask string
	command := &cobra.Command{
		Use:   "implement [path]",
		Short: "Implement a task interactively in Pi",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf(
					"get working directory: %w", err)
			}
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			ui := console.New(
				cmd.InOrStdin(), cmd.ErrOrStderr())
			resolved, err := resolveImplementation(
				cwd, path, subtask,
				cmd.Flags().Changed("subtask"), ui)
			if err != nil {
				return err
			}
			return launcher.Launch(resolved, agent.Streams{
				Input:       cmd.InOrStdin(),
				Output:      result,
				Diagnostics: cmd.ErrOrStderr(),
			})
		},
	}
	command.Flags().StringVar(&subtask, "subtask", "", "Exact subtask title to implement")
	return command
}

func resolveImplementation(
	cwd, path, subtask string, hasSubtask bool,
	ui *console.Console,
) (agent.Request, error) {
	store, err := task.NewStore(cwd, path)
	if err != nil {
		return agent.Request{}, err
	}
	if path != "" {
		return resolveImplementPath(store, cwd, path, subtask, hasSubtask, ui)
	}
	return resolveImplementDiscovery(store, subtask, hasSubtask, ui)
}

func resolveImplementPath(
	store *task.Store, cwd, path, subtask string,
	hasSubtask bool, ui *console.Console,
) (agent.Request, error) {
	selected, err := store.TaskByPath(path)
	if err != nil {
		return agent.Request{}, err
	}
	if selected.Type == task.TypeTopLevel {
		return resolveImplementTopLevel(selected, subtask, hasSubtask, ui)
	}
	if selected.Type == task.TypeTask {
		if hasSubtask {
			return agent.Request{}, fmt.Errorf("cannot specify --subtask with a subtask file path")
		}
		return resolveImplementDirect(store, selected)
	}
	return agent.Request{}, fmt.Errorf("unsupported task type: %s", selected.Path)
}

func resolveImplementDiscovery(
	store *task.Store, subtask string, hasSubtask bool,
	ui *console.Console,
) (agent.Request, error) {
	candidates, err := store.TopLevelTasks()
	if err != nil {
		return agent.Request{}, err
	}
	var choices []task.Task
	var labels []string
	for _, c := range candidates {
		if c.Status != "refined" {
			continue
		}
		if !hasEligibleSubtasks(c.Subtasks) {
			continue
		}
		choices = append(choices, c)
		labels = append(labels, c.Title)
	}
	if len(choices) == 0 {
		return agent.Request{}, fmt.Errorf("no refined tasks with eligible subtasks")
	}
	i, err := ui.Pick("Select a task:", labels)
	if err != nil {
		return agent.Request{}, err
	}
	return resolveImplementTopLevel(
		choices[i], subtask, hasSubtask, ui)
}

func resolveImplementTopLevel(
	topLevel task.Task, subtask string,
	hasSubtask bool, ui *console.Console,
) (agent.Request, error) {
	if topLevel.ProjectRoot == "" {
		return agent.Request{}, fmt.Errorf("task document is not inside a .shemiq project: %s", topLevel.Path)
	}
	if topLevel.Type != task.TypeTopLevel {
		return agent.Request{}, fmt.Errorf("expected type: top-level: %s", topLevel.Path)
	}
	if topLevel.Title == "" {
		return agent.Request{}, fmt.Errorf("empty top-level task title: %s", topLevel.Path)
	}
	if topLevel.Status != "refined" {
		return agent.Request{}, fmt.Errorf("top-level task is not refined: %s", topLevel.Path)
	}
	entry, err := chooseImplementSubtask(topLevel.Subtasks, subtask, hasSubtask, ui)
	if err != nil {
		return agent.Request{}, err
	}
	return implementRequest(topLevel, entry), nil
}

func resolveImplementDirect(
	store *task.Store, child task.Task,
) (agent.Request, error) {
	if child.ProjectRoot == "" {
		return agent.Request{}, fmt.Errorf(
			"task document is not inside a .shemiq project: %s",
			child.Path)
	}
	if child.Status == "done" {
		return agent.Request{}, fmt.Errorf(
			"subtask is done: %s", child.Path)
	}
	parentPath := child.ParentPath()
	switch child.Status {
	case "refined":
		return agent.Request{
			Flow:         agent.ImplementSubtask,
			DocumentPath: child.Path,
			ProjectRoot:  child.ProjectRoot,
		}, nil
	default: // new
		if parentPath == "" {
			return agent.Request{}, fmt.Errorf(
				"new subtask has no parent reference: %s",
				child.Path)
		}
		return agent.Request{
			Flow:         agent.ImplementSubtaskParent,
			DocumentPath: parentPath,
			ProjectRoot:  child.ProjectRoot,
			SubtaskTitle: child.Title,
		}, nil
	}
}

func implementRequest(
	parent task.Task, entry task.Task,
) agent.Request {
	projectRoot := parent.ProjectRoot
	if entry.ProjectRoot != "" {
		projectRoot = entry.ProjectRoot
	}
	switch entry.Status {
	case "refined":
		return agent.Request{
			Flow:         agent.ImplementSubtask,
			DocumentPath: entry.Path,
			ProjectRoot:  projectRoot,
		}
	default: // new
		return agent.Request{
			Flow:         agent.ImplementSubtaskParent,
			DocumentPath: parent.Path,
			ProjectRoot:  parent.ProjectRoot,
			SubtaskTitle: entry.Title,
		}
	}
}

func hasEligibleSubtasks(subtasks []task.Task) bool {
	for _, sub := range subtasks {
		if sub.Type == task.TypeTask &&
			sub.Title != "" &&
			(sub.Status == "new" || sub.Status == "refined") {
			return true
		}
	}
	return false
}

func chooseImplementSubtask(
	subtasks []task.Task, title string, exact bool, ui *console.Console) (task.Task, error) {
	if exact {
		return matchImplementSubtask(subtasks, title)
	}
	return pickImplementSubtask(subtasks, ui)
}

func matchImplementSubtask(subtasks []task.Task, title string) (task.Task, error) {
	var matches []task.Task
	for _, sub := range subtasks {
		if sub.Title == title {
			matches = append(matches, sub)
		}
	}
	if len(matches) == 0 {
		return task.Task{}, fmt.Errorf("subtask not found: %q", title)
	}
	if len(matches) != 1 {
		return task.Task{}, fmt.Errorf("ambiguous subtask title: %q", title)
	}
	match := matches[0]
	if match.Title == "" {
		return task.Task{}, fmt.Errorf("empty subtask title")
	}
	if match.Type != task.TypeTask {
		return task.Task{}, fmt.Errorf("subtask %q is not type: task", title)
	}
	if match.Status == "done" {
		return task.Task{}, fmt.Errorf("subtask %q is done", title)
	}
	return match, nil
}

func pickImplementSubtask(subtasks []task.Task, ui *console.Console) (task.Task, error) {
	var eligible []task.Task
	var labels []string
	for _, sub := range subtasks {
		if sub.Type == task.TypeTask && sub.Title != "" &&
			(sub.Status == "new" || sub.Status == "refined") {
			eligible = append(eligible, sub)
			labels = append(labels, fmt.Sprintf("%s [%s]", sub.Title, sub.Status))
		}
	}
	if len(eligible) == 0 {
		return task.Task{}, fmt.Errorf("no eligible subtasks available")
	}
	i, err := ui.Pick("Select a subtask:", labels)
	if err != nil {
		return task.Task{}, err
	}
	selected := eligible[i]
	for j, e := range eligible {
		if j != i && e.Title == selected.Title {
			return task.Task{}, fmt.Errorf("ambiguous subtask title: %q", selected.Title)
		}
	}
	return selected, nil
}
