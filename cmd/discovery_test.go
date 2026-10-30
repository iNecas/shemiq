package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

// TestEntryPointDiscoveryIgnoresUnrelatedMarkdown is the representative
// regression for entry-point-based discovery: only top-level.md files and
// documents reachable through references are tasks. Supporting Markdown in
// task directories (direct or nested) is ignored even when it contains
// valid or malformed Shemiq content.
func TestEntryPointDiscoveryIgnoresUnrelatedMarkdown(t *testing.T) {
	project := t.TempDir()
	tasks := filepath.Join(project, ".shemiq", "tasks", "example")
	writeTestMarkdown(t, filepath.Join(tasks, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Example
		:::shemiq
		type: top-level
		uuid: %s
		status: done
		:::

		## Tasks

		### Child
		:::shemiq
		type: task
		source: ./child.md
		status: done
		:::
		`), testUUID))
	writeTestMarkdown(t, filepath.Join(tasks, "child.md"),
		utils.Dedent(`
		# Child
		:::shemiq
		type: task
		parent: ./top-level.md
		uuid: 87654321-4321-4321-8321-abcdefabcdef
		status: done
		:::
		`))
	// Unrelated supporting Markdown: direct and nested.
	plain := writeTestMarkdown(t, filepath.Join(tasks, "notes.md"),
		utils.Dedent(`
		# Meeting notes

		Ordinary supporting content.
		`))
	draft := writeTestMarkdown(t, filepath.Join(tasks, "notes", "draft.md"),
		utils.Dedent(`
		# Draft
		:::shemiq
		type: task
		:::
		`))
	malformed := writeTestMarkdown(t, filepath.Join(tasks, "notes", "broken.md"),
		utils.Dedent(`
		:::shemiq
		not a field
		:::
		`))
	ignored := make(map[string]string, 3)
	for _, path := range []string{plain, draft, malformed} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		ignored[path] = string(data)
	}
	t.Chdir(project)

	out, diag, err := runCommand("", "validate")
	if err != nil || out != "" || diag != "" {
		t.Fatalf("validate: out=%q diag=%q err=%v",
			out, diag, err)
	}
	out, diag, err = runCommand("", "validate", "--fix")
	if err != nil || out != "" || diag != "" {
		t.Fatalf("validate --fix: out=%q diag=%q err=%v",
			out, diag, err)
	}
	for path, before := range ignored {
		after, err := os.ReadFile(path)
		if err != nil || string(after) != before {
			t.Fatalf(
				"ignored file changed: %s: %q, %v",
				path, after, err)
		}
	}
	out, diag, err = runCommand("", "tree")
	if err != nil || diag != "" {
		t.Fatalf("tree: out=%q diag=%q err=%v",
			out, diag, err)
	}
	expected := utils.Dedent(`
		Example [done]
		└── Child [done]
		`)
	if out != expected {
		t.Fatalf("tree output mismatch:\ngot:  %q\nwant: %q",
			out, expected)
	}
}
