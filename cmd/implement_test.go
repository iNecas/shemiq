package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/agent"
	"github.com/iNecas/shemiq/internal/utils"
)

func TestImplementLaunchAndSelection(t *testing.T) {
	root := t.TempDir()
	subtaskFile := implementSubtaskFile(t, root,
		"tasks", "proj", "sub-task",
		"Sub Task", "refined")
	parentPath := implementFixture(t, root,
		"tasks", "proj", "Project",
		"status: refined\n",
		utils.Dedent(`
			### New subtask
			:::shemiq
			type: task
			source: ./new-sub.md
			:::

			### Sub Task
			:::shemiq
			type: task
			status: refined
			source: ./sub-task.md
			:::

			### Done subtask
			:::shemiq
			type: task
			status: done
			source: ./done.md
			:::
			`))
	t.Chdir(root)
	taskPrompt := "Select a task:\n1. Project\nSelection: "
	subtaskPrompt := "Select a subtask:\n" +
		"1. New subtask [new]\n" +
		"2. Sub Task [refined]\nSelection: "
	for _, tc := range []struct {
		name, input string
		args        []string
		wantFlow    agent.Flow
		wantDoc     string
		wantTitle   string
		diag        string
	}{
		{
			"interactive new subtask",
			"1\n1\n", []string{"implement"},
			agent.ImplementSubtaskParent, parentPath,
			"New subtask",
			taskPrompt + subtaskPrompt,
		},
		{
			"interactive refined subtask",
			"1\n2\n", []string{"implement"},
			agent.ImplementSubtask, subtaskFile, "",
			taskPrompt + subtaskPrompt,
		},
		{
			"exact title new",
			"", []string{
				"implement", parentPath,
				"--subtask", "New subtask"},
			agent.ImplementSubtaskParent, parentPath,
			"New subtask", "",
		},
		{
			"exact title refined",
			"", []string{
				"implement", parentPath,
				"--subtask", "Sub Task"},
			agent.ImplementSubtask, subtaskFile, "", "",
		},
		{
			"direct subtask file refined",
			"", []string{"implement", subtaskFile},
			agent.ImplementSubtask, subtaskFile, "", "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher := &recordingLauncher{}
			out, diag, err := runImplementCommand(
				t, launcher, tc.input, tc.args...)
			if err != nil || out != "" || diag != tc.diag {
				t.Fatalf("out=%q diag=%q err=%v; want diag=%q",
					out, diag, err, tc.diag)
			}
			want := agent.Request{
				Flow:         tc.wantFlow,
				DocumentPath: tc.wantDoc,
				ProjectRoot:  root,
				SubtaskTitle: tc.wantTitle,
			}
			if launcher.calls != 1 ||
				launcher.request != want {
				t.Fatalf("calls=%d request=%+v; want %+v",
					launcher.calls,
					launcher.request, want)
			}
		})
	}
}

func TestImplementDirectNewSubtask(t *testing.T) {
	root := t.TempDir()
	// Create a new subtask file with parent reference.
	subtaskFile := implementSubtaskFile(t, root,
		"tasks", "proj", "new-sub",
		"New Sub", "new")
	implementFixture(t, root,
		"tasks", "proj", "Project",
		"status: refined\n",
		utils.Dedent(`
			### New Sub
			:::shemiq
			type: task
			source: ./new-sub.md
			:::
			`))
	parentPath := filepath.Join(
		root, ".shemiq", "tasks", "proj", "top-level.md")
	t.Chdir(root)
	launcher := &recordingLauncher{}
	out, diag, err := runImplementCommand(
		t, launcher, "", "implement", subtaskFile)
	if err != nil || out != "" || diag != "" {
		t.Fatalf("out=%q diag=%q err=%v", out, diag, err)
	}
	want := agent.Request{
		Flow:         agent.ImplementSubtaskParent,
		DocumentPath: parentPath,
		ProjectRoot:  root,
		SubtaskTitle: "New Sub",
	}
	if launcher.calls != 1 || launcher.request != want {
		t.Fatalf("calls=%d request=%+v; want %+v",
			launcher.calls, launcher.request, want)
	}
}

func TestImplementRejections(t *testing.T) {
	root := t.TempDir()
	newParent := implementFixture(t, root,
		"tasks", "new-parent", "New Parent", "", "")
	doneParent := implementFixture(t, root,
		"tasks", "done-parent", "Done",
		"status: done\n", "")
	refined := implementFixture(t, root,
		"tasks", "refined", "Refined",
		"status: refined\n",
		utils.Dedent(`
			### Same
			:::shemiq
			type: task
			:::

			### Same
			:::shemiq
			type: task
			:::

			### Finished
			:::shemiq
			type: task
			status: done
			:::
			`))
	noEligible := implementFixture(t, root,
		"tasks", "no-eligible", "No Eligible",
		"status: refined\n",
		utils.Dedent(`
			### Done
			:::shemiq
			type: task
			status: done
			:::
			`))
	_ = noEligible
	childPath := filepath.Join(
		filepath.Dir(newParent), "child.md")
	child := "# Child\n:::shemiq\ntype: task\n:::\n"
	if err := os.WriteFile(
		childPath, []byte(child), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, tc := range []struct {
		name, input, message string
		args                 []string
	}{
		{"not refined parent", "",
			"is not refined",
			[]string{"implement", newParent}},
		{"done parent", "",
			"is not refined",
			[]string{"implement", doneParent}},
		{"ambiguous exact title", "",
			"ambiguous subtask",
			[]string{"implement", refined,
				"--subtask", "Same"}},
		{"done subtask", "",
			"is done",
			[]string{"implement", refined,
				"--subtask", "Finished"}},
		{"not found", "",
			"subtask not found",
			[]string{"implement", refined,
				"--subtask", "Other"}},
		{"cancel selection", "",
			"selection cancelled",
			[]string{"implement", refined}},
		{"subtask flag with direct file", "",
			"cannot specify --subtask",
			[]string{"implement", childPath,
				"--subtask", "X"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher := &recordingLauncher{}
			out, diag, err := runImplementCommand(
				t, launcher, tc.input, tc.args...)
			if err == nil || out != "" ||
				!strings.Contains(
					err.Error(), tc.message) ||
				launcher.calls != 0 {
				t.Fatalf(
					"out=%q diag=%q err=%v calls=%d; "+
						"want %q",
					out, diag, err,
					launcher.calls, tc.message)
			}
		})
	}
}

func TestImplementStatusLabels(t *testing.T) {
	root := t.TempDir()
	implementFixture(t, root,
		"tasks", "proj", "Project",
		"status: refined\n",
		utils.Dedent(`
			### Alpha
			:::shemiq
			type: task
			:::

			### Beta
			:::shemiq
			type: task
			status: refined
			source: ./beta.md
			:::
			`))
	implementSubtaskFile(t, root,
		"tasks", "proj", "beta", "Beta", "refined")
	t.Chdir(root)
	launcher := &recordingLauncher{}
	// Pick the first (new) subtask.
	out, diag, err := runImplementCommand(
		t, launcher, "1\n1\n", "implement")
	if err != nil || out != "" {
		t.Fatalf("err=%v out=%q", err, out)
	}
	wantSubtaskPrompt := "Select a subtask:\n" +
		"1. Alpha [new]\n" +
		"2. Beta [refined]\nSelection: "
	if !strings.Contains(diag, wantSubtaskPrompt) {
		t.Fatalf("status labels missing: %q", diag)
	}
}

func runImplementCommand(
	t *testing.T, launcher *recordingLauncher,
	input string, args ...string,
) (string, string, error) {
	t.Helper()
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	err := execute(stdin, &stdout, &stderr, args, launcher)
	return stdout.String(), stderr.String(), err
}

func implementFixture(
	t *testing.T, root, category, dir, title,
	status, subtasks string,
) string {
	t.Helper()
	path := filepath.Join(
		root, ".shemiq", category, dir, "top-level.md")
	if err := os.MkdirAll(
		filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(utils.Dedent(`
		# %s
		:::shemiq
		type: top-level
		%s:::
		`), title, status)
	if subtasks != "" {
		content += "\n## Tasks\n\n" + subtasks
	}
	if err := os.WriteFile(
		path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func implementSubtaskFile(
	t *testing.T, root, category, dir, slug,
	title, status string,
) string {
	t.Helper()
	path := filepath.Join(
		root, ".shemiq", category, dir, slug+".md")
	if err := os.MkdirAll(
		filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(utils.Dedent(`
		# %s
		:::shemiq
		type: task
		parent: ./top-level.md
		status: %s
		:::
		`), title, status)
	if err := os.WriteFile(
		path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
