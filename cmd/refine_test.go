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

func TestRefineExplicitPathsAndSelection(t *testing.T) {
	root := t.TempDir()
	newPath := refineFixture(t, root, "tasks", "first", "First", "", "")
	archived := refineFixture(t, root, "archive", "old one", "Old", "status: new\n", "")
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
	cwd := filepath.Join(root, "src")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	for _, tc := range []struct {
		name, input, path, title string
		args                     []string
		diag                     string
	}{
		{"directory", "", newPath, "", []string{"refine", filepath.Dir(newPath)}, ""},
		{"file from nested cwd", "", newPath, "", []string{"refine", filepath.Join("..", ".shemiq", "tasks", "first", "top-level.md")}, ""},
		{"archived", "", archived, "", []string{"refine", filepath.Dir(archived)}, ""},
		{"exact title", "", refined, "A \"quoted\" title", []string{"refine", refined, "--subtask", "A \"quoted\" title"}, ""},
		{"two picks", "2\n1\n", refined, "A \"quoted\" title", []string{"refine"}, "Select a task:\n1. First\n2. Second\nSelection: Select a subtask:\n1. A \"quoted\" title\nSelection: "},
		{"retry both picks", "\nno\n2\n0\n1\n", refined, "A \"quoted\" title", []string{"refine"}, "Select a task:\n1. First\n2. Second\nSelection: Please enter a number from 1 to 2.\nSelection: Please enter a number from 1 to 2.\nSelection: Select a subtask:\n1. A \"quoted\" title\nSelection: Please enter a number from 1 to 1.\nSelection: "},
		{"pick task then exact subtask", "2\n", refined, "A \"quoted\" title", []string{"refine", "--subtask", "A \"quoted\" title"}, "Select a task:\n1. First\n2. Second\nSelection: "},
		{"one subtask pick", "1\n", refined, "A \"quoted\" title", []string{"refine", refined}, "Select a subtask:\n1. A \"quoted\" title\nSelection: "},
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
	repeated := refineFixture(t, root, "archive", "repeated", "Repeated", "status: new\nstatus: refined\n", "")
	noNew := refineFixture(t, root, "archive", "no-new", "No new", "status: refined\n", utils.Dedent(`
		### Finished
		:::shemiq
		type: task
		status: done
		:::
		`))
	wrongFile := filepath.Join(filepath.Dir(newPath), "other.md")
	if err := os.WriteFile(wrongFile, []byte("wrong filename"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, tc := range []struct {
		name, input, message string
		args                 []string
	}{
		{"done", "", "is done", []string{"refine", done}},
		{"new with subtask", "", "cannot specify --subtask", []string{"refine", newPath, "--subtask", "Same"}},
		{"ambiguous", "", "ambiguous subtask", []string{"refine", refined, "--subtask", "Same"}},
		{"ineligible", "", "is not new", []string{"refine", refined, "--subtask", "Finished"}},
		{"no eligible", "", "no new subtasks", []string{"refine", noNew}},
		{"not found", "", "subtask not found", []string{"refine", refined, "--subtask", "Other"}},
		{"invalid status", "", "invalid status", []string{"refine", invalid}},
		{"repeated status", "", "repeated status", []string{"refine", repeated}},
		{"blank then EOF", "\n", "selection cancelled", []string{"refine", refined}},
		{"invalid then EOF", "3\n", "selection cancelled", []string{"refine", refined}},
		{"EOF", "", "selection cancelled", []string{"refine"}},
		{"EOF after partial line", "1", "selection cancelled", []string{"refine", refined}},
		{"wrong filename", "", "expected top-level.md", []string{"refine", wrongFile}},
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

func TestRefineProjectAndActiveDocumentRequirements(t *testing.T) {
	root := t.TempDir()
	valid := refineFixture(t, root, "tasks", "valid", "Valid", "", "")
	t.Chdir(root)
	missing := filepath.Join(root, ".shemiq", "tasks", "broken")
	if err := os.Mkdir(missing, 0755); err != nil {
		t.Fatal(err)
	}
	launcher := &recordingLauncher{}
	out, diag, err := runRefineCommand(t, launcher, "1\n", "refine")
	if err == nil || out != "" || !strings.Contains(diag, filepath.Join(missing, "top-level.md")) || strings.Contains(diag, "Select a task:") || launcher.calls != 0 {
		t.Fatalf("missing active document: out=%q diag=%q err=%v calls=%d", out, diag, err, launcher.calls)
	}
	// Explicit paths locate their own project, even when the caller has none.
	other := t.TempDir()
	t.Chdir(other)
	out, diag, err = runRefineCommand(t, launcher, "", "refine", valid)
	want := agent.Request{Flow: agent.RefineTopLevel, DocumentPath: valid, ProjectRoot: root}
	if err != nil || diag != "" || out != "" || launcher.calls != 1 || launcher.request != want {
		t.Fatalf("explicit cross-project path: out=%q diag=%q err=%v request=%+v", out, diag, err, launcher.request)
	}
	launcher.calls = 0
	out, diag, err = runRefineCommand(t, launcher, "", "refine")
	if err == nil || out != "" || !strings.Contains(diag, "no .shemiq directory") || launcher.calls != 0 {
		t.Fatalf("no project: out=%q diag=%q err=%v calls=%d", out, diag, err, launcher.calls)
	}
	outside := filepath.Join(other, "top-level.md")
	if err := os.WriteFile(outside, []byte(utils.Dedent(`
		# Outside
		:::shemiq
		type: top-level
		:::
		`)), 0644); err != nil {
		t.Fatal(err)
	}
	out, diag, err = runRefineCommand(t, launcher, "", "refine", outside)
	if err == nil || out != "" || !strings.Contains(diag, "not inside a .shemiq project") || launcher.calls != 0 {
		t.Fatalf("outside project: out=%q diag=%q err=%v calls=%d", out, diag, err, launcher.calls)
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
