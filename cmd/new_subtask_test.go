package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestNewSubtaskPromptedTitle(t *testing.T) {
	parentPath := newSubtaskParent(t, "top-level.md")
	stdout, stderr, err := runCommand("New Subtask\n", "new-subtask",
		filepath.Join(".shemiq", "tasks", "my-task"), "Do something important")
	if err != nil || stdout != parentPath+"\n" || stderr != "Title: " {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	content, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "### New Subtask\n\nDo something important\n") {
		t.Fatalf("prompted title or description not passed through:\n%s", content)
	}

	out, diag, err := runCommand("", "validate", parentPath)
	if err != nil || out != "" || diag != "" {
		t.Fatalf("validation: stdout=%q stderr=%q err=%v", out, diag, err)
	}
}

func TestNewSubtaskFlags(t *testing.T) {
	parentPath := newSubtaskParent(t, "feature.md")
	stdout, stderr, err := runCommand("Unused title\n", "new-subtask", parentPath,
		"--title", "New Subtask", "--after", "First Task", "Describe", "the task")
	if err != nil || stdout != parentPath+"\n" || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	content, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "### New Subtask\n\nDescribe the task\n### Second Task") {
		t.Fatalf("title, joined description or --after not passed through:\n%s", content)
	}
}

func TestNewSubtaskWithTitleOnly(t *testing.T) {
	for _, description := range [][]string{nil, {""}, {"  "}} {
		for _, after := range []string{"", "First Task"} {
			t.Run(fmt.Sprintf("description=%q/after=%s", description, after), func(t *testing.T) {
				parentPath := newSubtaskParent(t, "top-level.md")
				args := []string{"new-subtask", parentPath, "--title", "Title only"}
				if after != "" {
					args = append(args, "--after", after)
				}
				args = append(args, description...)
				stdout, stderr, err := runCommand("", args...)
				if err != nil || stdout != parentPath+"\n" || stderr != "" {
					t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
				}
				content, err := os.ReadFile(parentPath)
				if err != nil {
					t.Fatal(err)
				}
				inserted := strings.Index(string(content), "### Title only\n")
				second := strings.Index(string(content), "### Second Task\n")
				if inserted < 0 || (after != "" && inserted > second) || (after == "" && inserted < second) {
					t.Fatalf("subtask missing or in wrong position: %s", content)
				}
				out, diag, err := runCommand("", "validate", parentPath)
				if err != nil || out != "" || diag != "" {
					t.Fatalf("validation: stdout=%q stderr=%q err=%v", out, diag, err)
				}
			})
		}
	}
}

func TestNewSubtaskInputErrors(t *testing.T) {
	parentPath := newSubtaskParent(t, "top-level.md")
	for _, tc := range []struct {
		name, message string
		args          []string
	}{
		{"missing parent", "requires at least 1 arg(s)", nil},
		{"missing description", "description is required", []string{parentPath}},
		{"blank description", "description is required", []string{parentPath, " "}},
		{"blank title flag", "title is required", []string{parentPath, "--title", " ", "desc"}},
		{"missing path", "inspect parent", []string{parentPath + ".missing", "desc"}},
		{"store error", "duplicate subtask title", []string{parentPath, "--title", "First Task", "desc"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCommand("Fallback title\n", append([]string{"new-subtask"}, tc.args...)...)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error=%v, want %q", err, tc.message)
			}
			if stdout != "" || !strings.Contains(stderr, tc.message) || strings.Contains(stderr, "Title: ") {
				t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}

// newSubtaskParent creates a project and sets it as the invocation directory.
func newSubtaskParent(t *testing.T, filename string) string {
	t.Helper()
	project := t.TempDir()
	parentDir := filepath.Join(project, ".shemiq", "tasks", "my-task")
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatal(err)
	}
	parentPath := filepath.Join(parentDir, filename)
	if err := os.WriteFile(parentPath, []byte(utils.Dedent(`
		# My Task
		:::shemiq
		type: top-level
		uuid: 11111111-1111-4111-8111-111111111111
		:::

		## Tasks

		### First Task

		First description

		### Second Task

		Second description
		`)), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	return parentPath
}
