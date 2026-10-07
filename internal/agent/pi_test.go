package agent

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/term"
)

func TestPiRequiresTerminalStreams(t *testing.T) {
	input, output := testFile(t), testFile(t)
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { readPipe.Close(); writePipe.Close() })
	// If the terminal check were skipped, PATH lookup would give a different error.
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		name    string
		streams Streams
		peer    *os.File
	}{
		{"stdin file", Streams{Input: input, Output: output}, output},
		{"stdin pipe", Streams{Input: readPipe, Output: output}, output},
		{"stdout file", Streams{Input: input, Output: output}, input},
		{"stdout pipe", Streams{Input: input, Output: writePipe}, input},
		{"wrapped stdin", Streams{Input: strings.NewReader(""), Output: output}, output},
		{"wrapped stdout", Streams{Input: input, Output: &bytes.Buffer{}}, input},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Stand in for the other terminal; the redirected stream is probed
			// with the real portable terminal check, not just a file assertion.
			pi := Pi{IsTerminal: func(fd int) bool {
				return fd == int(tc.peer.Fd()) || term.IsTerminal(fd)
			}}
			err := pi.Launch(Request{}, tc.streams)
			if err == nil || !strings.Contains(
				err.Error(),
				"requires terminal stdin and stdout") {
				t.Fatalf("err=%v; expected terminal rejection", err)
			}
		})
	}
	// The production default also rejects redirected files.
	if err := (Pi{}).Launch(Request{}, Streams{Input: input, Output: output}); err == nil || !strings.Contains(err.Error(), "requires terminal") {
		t.Fatalf("default terminal probe: %v", err)
	}
}

func TestPiMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	err := (Pi{IsTerminal: func(int) bool { return true }}).Launch(
		Request{Flow: RefineTopLevel, ProjectRoot: root, DocumentPath: filepath.Join(root, "top-level.md")},
		Streams{Input: testFile(t), Output: testFile(t)},
	)
	if !errors.Is(err, exec.ErrNotFound) || !strings.Contains(err.Error(), "install Pi") || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("expected actionable wrapped PATH error: %v", err)
	}
}

func TestPiMessageQuoting(t *testing.T) {
	root := t.TempDir()
	request := Request{
		Flow: RefineSubtask, ProjectRoot: root,
		DocumentPath: filepath.Join(root, ".shemiq", "tasks", "space task", "top-level.md"),
		SubtaskTitle: `A "double" and 'single' \ path`,
	}
	message, err := piMessage(request)
	want := "/shemiq-refine-sub-task '" + filepath.Join(".shemiq", "tasks", "space task", "top-level.md") + `' 'A "double" and '"'"'single'"'"' \ path'`
	if err != nil || message != want {
		t.Fatalf("message=%q err=%v; want %q", message, err, want)
	}
	// The same encoding is used for paths as well as titles.
	if got := quotePiArgument(`a 'quote' "double" \ path`); got != `'a '"'"'quote'"'"' "double" \ path'` {
		t.Fatalf("quotePiArgument=%q", got)
	}
	request.Flow, request.SubtaskTitle = RefineTopLevel, ""
	message, err = piMessage(request)
	if err != nil || message != "/shemiq-refine-and-split '"+filepath.Join(".shemiq", "tasks", "space task", "top-level.md")+"'" {
		t.Fatalf("top-level message=%q err=%v", message, err)
	}
}

func TestPiImplementFlowMessages(t *testing.T) {
	root := t.TempDir()
	subPath := filepath.Join(
		root, ".shemiq", "tasks", "my-task", "sub.md")
	parentPath := filepath.Join(
		root, ".shemiq", "tasks", "my-task",
		"top-level.md")

	msg, err := piMessage(Request{
		Flow: ImplementSubtask, ProjectRoot: root,
		DocumentPath: subPath,
	})
	want := "/shemiq-implement-task '" +
		filepath.Join(".shemiq", "tasks", "my-task", "sub.md") +
		"'"
	if err != nil || msg != want {
		t.Fatalf("msg=%q err=%v; want %q", msg, err, want)
	}

	msg, err = piMessage(Request{
		Flow:         ImplementSubtaskParent,
		ProjectRoot:  root,
		DocumentPath: parentPath,
		SubtaskTitle: "My subtask",
	})
	want = "/shemiq-implement-task-parent '" +
		filepath.Join(
			".shemiq", "tasks", "my-task",
			"top-level.md") +
		"' 'My subtask'"
	if err != nil || msg != want {
		t.Fatalf("msg=%q err=%v; want %q", msg, err, want)
	}
}

func testFile(t *testing.T) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stream")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}
