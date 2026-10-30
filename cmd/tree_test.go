package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

// TestTreeHumanAndJSON covers the representative scenario with both output
// modes using a shared fixture: authoritative child titles, missing-source
// child, external descendant, nesting, deterministic ordering, and absence
// of duplicate roots.
func TestTreeHumanAndJSON(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, ".shemiq", "tasks")

	writeTestMarkdown(t, filepath.Join(active, "example/top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Example
		:::shemiq
		type: top-level
		uuid: %s
		status: refined
		:::

		## Tasks

		### Created child
		:::shemiq
		type: task
		source: ./child.md
		status: refined
		:::

		### Not yet created
		:::shemiq
		type: task
		source: ./missing.md
		status: new
		:::
		`), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"))

	writeTestMarkdown(t, filepath.Join(active, "example/child.md"),
		utils.Dedent(`
		# Authoritative title
		:::shemiq
		type: task
		parent: ./top-level.md
		uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
		status: refined
		:::

		## Tasks

		### Grandchild
		:::shemiq
		type: task
		source: ./grandchild.md
		status: refined
		:::
		`))

	writeTestMarkdown(t, filepath.Join(active, "example/grandchild.md"),
		utils.Dedent(`
		# Grandchild doc
		:::shemiq
		type: task
		parent: ./child.md
		uuid: cccccccc-cccc-4ccc-8ccc-cccccccccccc
		status: refined
		:::
		`))

	t.Chdir(root)

	// Human output
	out, diag, err := runCommand("", "tree")
	if err != nil {
		t.Fatalf("tree: out=%q diag=%q err=%v", out, diag, err)
	}
	if diag != "" {
		t.Fatalf("unexpected stderr: %q", diag)
	}
	expected := utils.Dedent(`
		Example [refined]
		├── Authoritative title [refined]
		│   └── Grandchild doc [refined]
		└── Not yet created [new]
		`)
	if out != expected {
		t.Fatalf("human output mismatch:\ngot:  %q\nwant: %q", out, expected)
	}

	// JSON output
	out, diag, err = runCommand("", "tree", "--json")
	if err != nil {
		t.Fatalf("tree --json: out=%q diag=%q err=%v", out, diag, err)
	}
	if diag != "" {
		t.Fatalf("unexpected stderr: %q", diag)
	}
	// Check two-space indentation and trailing newline.
	if !strings.HasSuffix(out, "}\n") {
		t.Fatalf("JSON should end with trailing newline: %q", out)
	}
	if !strings.Contains(out, "  ") {
		t.Fatalf("JSON should use two-space indentation: %q", out)
	}
	// Parse and verify structure.
	var doc struct {
		Tasks []struct {
			Title    string `json:"title"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Subtasks []struct {
				Title string `json:"title"`
			} `json:"subtasks"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].Title != "Example" {
		t.Fatalf("unexpected JSON root: %+v", doc.Tasks)
	}
	if len(doc.Tasks[0].Subtasks) != 2 ||
		doc.Tasks[0].Subtasks[0].Title != "Authoritative title" {
		t.Fatalf("unexpected JSON subtasks: %+v", doc.Tasks[0].Subtasks)
	}
}

func TestTreeEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	if err := os.MkdirAll(empty, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	out, diag, err := runCommand("", "tree", empty)
	if err != nil || diag != "" {
		t.Fatalf("tree empty: out=%q diag=%q err=%v", out, diag, err)
	}
	if out != "No tasks.\n" {
		t.Fatalf("expected 'No tasks.\\n', got %q", out)
	}

	out, diag, err = runCommand("", "tree", "--json", empty)
	if err != nil || diag != "" {
		t.Fatalf("tree --json empty: out=%q diag=%q err=%v", out, diag, err)
	}
	var doc2 struct {
		Tasks []interface{} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(out), &doc2); err != nil || len(doc2.Tasks) != 0 {
		t.Fatalf("unexpected empty JSON: %q", out)
	}
}

func TestTreeValidationError(t *testing.T) {
	dir := t.TempDir()
	writeTestMarkdown(t, filepath.Join(dir, "top-level.md"),
		utils.Dedent(`
		# Bad
		:::shemiq
		type: task
		:::
		`))
	t.Chdir(dir)

	out, diag, err := runCommand("", "tree", dir)
	if err == nil {
		t.Fatal("expected error for invalid metadata")
	}
	if out != "" {
		t.Fatalf("stdout should be empty on failure: %q", out)
	}
	if !strings.Contains(diag, "missing uuid") {
		t.Fatalf("expected validation finding in stderr: %q", diag)
	}
	// No usage text on runtime failure.
	if strings.Contains(diag, "Usage:") {
		t.Fatalf("should not show usage: %q", diag)
	}
}

func TestTreeFileArgRejected(t *testing.T) {
	dir := t.TempDir()
	f := writeTestMarkdown(t, filepath.Join(dir, "task.md"),
		fmt.Sprintf(utils.Dedent(`
		# Task
		:::shemiq
		type: task
		uuid: %s
		:::
		`), testUUID))
	t.Chdir(dir)

	out, diag, err := runCommand("", "tree", f)
	expected := "tree requires a directory scope, not a file: " + f
	if err == nil || err.Error() != expected {
		t.Fatalf("expected error %q, got %v", expected, err)
	}
	if out != "" {
		t.Fatalf("stdout should be empty: %q", out)
	}
	if diag != "Error: "+expected+"\n" {
		t.Fatalf("unexpected stderr: %q", diag)
	}
}

func TestTreeMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	missing := filepath.Join(dir, "nonexistent")
	out, diag, err := runCommand("", "tree", missing)
	if !errors.Is(err, os.ErrNotExist) ||
		!strings.HasPrefix(err.Error(), "inspect "+missing+": ") {
		t.Fatalf("expected missing-directory error for %q, got %v", missing, err)
	}
	if out != "" {
		t.Fatalf("stdout should be empty: %q", out)
	}
	if diag != "Error: "+err.Error()+"\n" {
		t.Fatalf("unexpected stderr: %q", diag)
	}
}
