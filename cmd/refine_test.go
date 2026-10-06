package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/agent"
	"github.com/iNecas/shemiq/internal/utils"
)

func TestRefineLaunchAndSelection(t *testing.T) {
	root := t.TempDir()
	newPath := refineFixture(t, root, "tasks", "first", "First", "", "")
	otherName := filepath.Join(filepath.Dir(newPath), "other.MARKDOWN")
	if err := os.WriteFile(otherName, []byte(readTestFile(t, newPath)), 0644); err != nil {
		t.Fatal(err)
	}
	refineFixture(t, root, "tasks", "done", "Done", "status: done\n", "")
	subtasks := utils.Dedent(`
		### Already refined
		:::shemiq
		type: task
		status: refined
		source: ./not-created.md
		:::

		### A "quoted" title
		:::shemiq
		type: task
		source: ./also-not-created.md
		:::
		`)
	refined := refineFixture(t, root, "tasks", "second", "Second", "status: refined\n", subtasks)
	t.Chdir(root)
	title := `A "quoted" title`
	taskPrompt := "Select a task:\n1. First\n2. Second\nSelection: "
	subtaskPrompt := "Select a subtask:\n1. " + title + "\nSelection: "
	for _, tc := range []struct {
		name, input, path, title string
		args                     []string
		diag                     string
	}{
		{"explicit top-level", "", otherName, "", []string{"refine", otherName}, ""},
		{"exact title", "", refined, title,
			[]string{"refine", refined, "--subtask", title}, ""},
		{"interactive selection", "2\n1\n", refined, title,
			[]string{"refine"}, taskPrompt + subtaskPrompt},
		{"pick task then exact subtask", "2\n", refined, title,
			[]string{"refine", "--subtask", title}, taskPrompt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher := &recordingLauncher{}
			out, diag, err := runRefineCommand(t, launcher, tc.input, tc.args...)
			if err != nil || out != "" || diag != tc.diag {
				t.Fatalf("out=%q diag=%q err=%v; want diag=%q", out, diag, err, tc.diag)
			}
			flow := agent.RefineTopLevel
			if tc.title != "" {
				flow = agent.RefineSubtask
			}
			want := agent.Request{Flow: flow, DocumentPath: tc.path, ProjectRoot: root, SubtaskTitle: tc.title}
			if launcher.calls != 1 || launcher.request != want {
				t.Fatalf("calls=%d request=%+v; want %+v", launcher.calls, launcher.request, want)
			}
		})
	}
}

func TestRefineRejections(t *testing.T) {
	root := t.TempDir()
	newPath := refineFixture(t, root, "tasks", "new", "New", "", "")
	done := refineFixture(t, root, "tasks", "done", "Done", "status: done\n", "")
	refined := refineFixture(t, root, "tasks", "refined", "Refined", "status: refined\n", utils.Dedent(`
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
	invalid := refineFixture(t, root, "archive", "invalid", "Invalid", "status: progress\n", "")
	noNew := refineFixture(t, root, "archive", "no-new", "No new", "status: refined\n", utils.Dedent(`
		### Finished
		:::shemiq
		type: task
		status: done
		:::
		`))
	childPath := filepath.Join(filepath.Dir(newPath), "child.md")
	child := "# Child\n:::shemiq\ntype: task\n:::\n"
	if err := os.WriteFile(childPath, []byte(child), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, tc := range []struct {
		name, input, message string
		args                 []string
	}{
		{"done", "", "is done", []string{"refine", done}},
		{"new with subtask", "", "cannot specify --subtask", []string{"refine", newPath, "--subtask", "Same"}},
		{"ambiguous exact title", "", "ambiguous subtask",
			[]string{"refine", refined, "--subtask", "Same"}},
		{"ambiguous interactive title", "1\n", "ambiguous subtask", []string{"refine", refined}},
		{"ineligible", "", "is not new", []string{"refine", refined, "--subtask", "Finished"}},
		{"no eligible", "", "no new subtasks", []string{"refine", noNew}},
		{"not found", "", "subtask not found", []string{"refine", refined, "--subtask", "Other"}},
		{"invalid status", "", "unsupported top-level status", []string{"refine", invalid}},
		{"cancel selection", "", "selection cancelled", []string{"refine", refined}},
		{"wrong task type", "", "expected type: top-level", []string{"refine", childPath}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher := &recordingLauncher{}
			out, diag, err := runRefineCommand(t, launcher, tc.input, tc.args...)
			if err == nil || out != "" || !strings.Contains(err.Error(), tc.message) || !strings.Contains(diag, tc.message) || launcher.calls != 0 {
				t.Fatalf("out=%q diag=%q err=%v calls=%d; want %q and no launch", out, diag, err, launcher.calls, tc.message)
			}
		})
	}
}

func TestRefineStoreRouting(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	path := refineFixture(t, root, "tasks", "example", "Example", utils.Dedent(`
		status: refined
		uuid: invalid
		unknown: ignored by refinement
		`), utils.Dedent(`
		### Invalid status
		:::shemiq
		type: task
		status: typo
		:::

		### Repeated status
		:::shemiq
		type: task
		status: new
		status: done
		:::

		### Wrong type
		:::shemiq
		type: top-level
		:::

		### Usable
		:::shemiq
		type: task
		source: ./missing.md
		uuid: invalid
		:::
		`))
	for _, tc := range []struct {
		name, input, title, message string
	}{
		{"pick usable despite issues", "1\n", "", ""},
		{"exact usable despite issues", "", "Usable", ""},
		{"invalid never defaults to new", "", "Invalid status", "is not new"},
		{"repeated never defaults to new", "", "Repeated status", "is not new"},
		{"exact wrong type", "", "Wrong type", "is not type: task"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"refine", path}
			if tc.title != "" {
				args = append(args, "--subtask", tc.title)
			}
			launcher := &recordingLauncher{}
			out, diag, err := runRefineCommand(t, launcher, tc.input, args...)
			if tc.message != "" {
				if err == nil || !strings.Contains(err.Error(), tc.message) || launcher.calls != 0 {
					t.Fatalf("diag=%q err=%v calls=%d", diag, err, launcher.calls)
				}
			} else if err != nil || out != "" || launcher.calls != 1 || launcher.request.SubtaskTitle != "Usable" {
				t.Fatalf("out=%q diag=%q err=%v request=%+v", out, diag, err, launcher.request)
			}
			if tc.input != "" && diag != "Select a subtask:\n1. Usable\nSelection: " {
				t.Fatalf("ineligible entries leaked into picker: %q", diag)
			}
		})
	}
	// Parser syntax is fatal even when it is unrelated to routing fields.
	before := readTestFile(t, path)
	if err := os.WriteFile(path, []byte(before+"\n:::shemiq\nmalformed\n:::\n"), 0644); err != nil {
		t.Fatal(err)
	}
	launcher := &recordingLauncher{}
	_, _, err := runRefineCommand(t, launcher, "", "refine", path, "--subtask", "Usable")
	if err == nil || !strings.Contains(err.Error(), "malformed metadata field") || launcher.calls != 0 {
		t.Fatalf("syntax failure: err=%v calls=%d", err, launcher.calls)
	}
	// Fenced syntax examples are not metadata and cannot block refinement.
	fenced := before + "\n```markdown\n:::shemiq\nmalformed\n```\n"
	if err := os.WriteFile(path, []byte(fenced), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err = runRefineCommand(t, launcher, "", "refine", path, "--subtask", "Usable")
	if err != nil || launcher.calls != 1 || readTestFile(t, path) != fenced {
		t.Fatalf("fenced example or read-only failure: err=%v calls=%d", err, launcher.calls)
	}
}

func TestRefineResolvedTargetContainment(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	// The caller has no project; containment and launch ownership follow the target.
	t.Chdir(outside)
	target := filepath.Join(outside, "top-level.md")
	if err := os.WriteFile(target, []byte("# Outside\n:::shemiq\ntype: top-level\n:::\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := refineFixture(t, root, "tasks", "alias", "Alias", "", "")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	launcher := &recordingLauncher{}
	_, _, err := runRefineCommand(t, launcher, "", "refine", path)
	if err == nil || !strings.Contains(err.Error(), "not inside a .shemiq project") || launcher.calls != 0 {
		t.Fatalf("lexical containment allowed escape: err=%v calls=%d", err, launcher.calls)
	}
	// An explicit alias to a different real project is allowed; launch there.
	other := t.TempDir()
	realPath := refineFixture(t, other, "archive", "real", "Real", "", "")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, path); err != nil {
		t.Fatal(err)
	}
	_, _, err = runRefineCommand(t, launcher, "", "refine", path)
	if err != nil || launcher.calls != 1 || launcher.request.DocumentPath != realPath || launcher.request.ProjectRoot != other {
		t.Fatalf("resolved launch: err=%v request=%+v", err, launcher.request)
	}
}

func TestRefineLaunchError(t *testing.T) {
	root := t.TempDir()
	path := refineFixture(t, root, "tasks", "new", "New", "", "")
	before := readTestFile(t, path)
	failure := errors.New("launcher failed")
	launcher := &recordingLauncher{err: failure}
	out, diag, err := runRefineCommand(t, launcher, "", "refine", path)
	if !errors.Is(err, failure) || out != "" || !strings.Contains(diag, failure.Error()) || launcher.calls != 1 {
		t.Fatalf("out=%q diag=%q err=%v calls=%d", out, diag, err, launcher.calls)
	}
	if after := readTestFile(t, path); after != before {
		t.Fatal("launch failure modified task content")
	}
}

func runRefineCommand(t *testing.T, launcher *recordingLauncher, input string, args ...string) (string, string, error) {
	t.Helper()
	stdin := strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	err := execute(stdin, &stdout, &stderr, args, launcher)
	if launcher.calls > 0 && (launcher.streams.Input != stdin || launcher.streams.Output != &stdout || launcher.streams.Diagnostics != &stderr) {
		t.Fatal("launcher did not receive the original input/result/diagnostic streams")
	}
	return stdout.String(), stderr.String(), err
}

type recordingLauncher struct {
	calls   int
	request agent.Request
	streams agent.Streams
	err     error
}

func (l *recordingLauncher) Launch(request agent.Request, streams agent.Streams) error {
	l.calls++
	l.request, l.streams = request, streams
	return l.err
}

func refineFixture(t *testing.T, root, category, dir, title, status, subtasks string) string {
	t.Helper()
	path := filepath.Join(root, ".shemiq", category, dir, "top-level.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
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
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
