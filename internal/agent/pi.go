package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// Pi launches the installed Pi CLI with its normally persisted session and configuration.
type Pi struct {
	// IsTerminal probes a file descriptor. Nil uses term.IsTerminal.
	// This dependency allows process tests to bypass only the terminal probe.
	IsTerminal func(int) bool
}

func (p Pi) Launch(request Request, streams Streams) error {
	probe := p.IsTerminal
	if probe == nil {
		probe = term.IsTerminal
	}
	if !isTerminalFile(streams.Input, probe) ||
		!isTerminalFile(streams.Output, probe) {
		return fmt.Errorf(
			"session requires terminal stdin and stdout; " +
				"run without input/output redirection")
	}
	message, err := piMessage(request)
	if err != nil {
		return err
	}
	executable, err := exec.LookPath("pi")
	if err != nil {
		return fmt.Errorf("find pi executable: install Pi (https://pi.dev/) and ensure pi is on PATH: %w", err)
	}
	// PATH entries may be relative to the caller. Pin the executable before
	// changing the child's working directory to the selected project.
	executable, err = filepath.Abs(executable)
	if err != nil {
		return fmt.Errorf("resolve pi executable path: %w", err)
	}
	child := exec.Command(executable, message)
	child.Dir = request.ProjectRoot
	// A nil Env inherits the environment. Direct *os.File streams preserve
	// terminal descriptors, so Pi does not silently switch into print mode.
	child.Stdin, child.Stdout, child.Stderr = streams.Input, streams.Output, streams.Diagnostics
	if err := child.Start(); err != nil {
		return fmt.Errorf("start pi in %q: %w", request.ProjectRoot, err)
	}
	if err := child.Wait(); err != nil {
		return fmt.Errorf("pi session failed: %w", err)
	}
	return nil
}

func piMessage(request Request) (string, error) {
	template, err := flowTemplate(request.Flow)
	if err != nil {
		return "", err
	}
	path, err := filepath.Rel(
		request.ProjectRoot, request.DocumentPath)
	if err != nil {
		return "", fmt.Errorf(
			"make document path relative to project root: %w",
			err)
	}
	message := "/" + template + " " + quotePiArgument(path)
	if request.SubtaskTitle != "" {
		message += " " + quotePiArgument(request.SubtaskTitle)
	}
	return message, nil
}

func flowTemplate(flow Flow) (string, error) {
	switch flow {
	case RefineTopLevel:
		return "shemiq-refine-and-split", nil
	case RefineSubtask:
		return "shemiq-refine-sub-task", nil
	case ImplementSubtask:
		return "shemiq-implement-task", nil
	case ImplementSubtaskParent:
		return "shemiq-implement-task-parent", nil
	default:
		return "", fmt.Errorf("unsupported flow: %q", flow)
	}
}

func isTerminalFile(stream any, probe func(int) bool) bool {
	file, ok := stream.(*os.File)
	return ok && file != nil && probe(int(file.Fd()))
}

// Pi's template parser joins adjacent quoted segments but does not interpret
// backslash escapes. Keep backslashes literal and quote apostrophes separately.
// This is encoding within a single argv message; no shell is involved.
func quotePiArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
