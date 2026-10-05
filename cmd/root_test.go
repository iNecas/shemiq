package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskNewDiscoversProject(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".shemiq"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(project, "src", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	stdout, stderr, err := runCommand("  My New Task!  \n", "task", "new", "Describe", "the task")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}
	path := filepath.Join(project, ".shemiq", "tasks", "my-new-task", "top-level.md")
	if stdout != path+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, path+"\n")
	}
	if stderr != "Title: " {
		t.Fatalf("stderr = %q", stderr)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `# My New Task!
:::shemiq
type: top-level
:::

## Description

Describe the task

## Context

[TBD]

## Interview

[TBD]

## Design

[TBD]

## Current status

[TBD]

## Tasks

[TBD]
`
	if string(content) != want {
		t.Fatalf("document = %q, want %q", content, want)
	}
}

func TestTaskNewWithTitleFlag(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	stdout, stderr, err := runCommand("", "task", "new", "--title", "Command Line Title", "Describe", "the task")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}
	path := filepath.Join(dir, ".shemiq", "tasks", "command-line-title", "top-level.md")
	if stdout != path+"\n" || stderr != "" {
		t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(content), "# Command Line Title\n:::shemiq\ntype: top-level\n:::\n") ||
		!strings.Contains(string(content), "## Description\n\nDescribe the task\n") {
		t.Fatalf("unexpected document: %s", content)
	}
}

func TestTaskNewCreatesProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	stdout, stderr, err := runCommand("Héllo  123\n", "task", "new", "--", "--some", "description")
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
	}
	path := filepath.Join(dir, ".shemiq", "tasks", "h-llo-123", "top-level.md")
	if stdout != path+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, path+"\n")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "## Description\n\n--some description\n") {
		t.Fatalf("document does not contain description: %s", content)
	}
}

func TestTaskNewRejectsMissingInput(t *testing.T) {
	for _, tc := range []struct {
		name, input, message string
		args                 []string
	}{
		{"missing description", "Unused\n", "requires at least 1 arg(s)", []string{"task", "new"}},
		{"blank description", "Unused\n", "description is required", []string{"task", "new", "  "}},
		{"blank title", "  \n", "title is required", []string{"task", "new", "description"}},
		{"blank title flag", "Fallback\n", "title is required", []string{"task", "new", "--title", "  ", "description"}},
		{"multiline title flag", "", "title must be one line", []string{"task", "new", "--title", "First\nSecond", "description"}},
		{"EOF without title", "", "title is required", []string{"task", "new", "description"}},
		{"empty slug", "é !\n", "ASCII letter or digit", []string{"task", "new", "description"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			stdout, stderr, err := runCommand(tc.input, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error = %v, want %q", err, tc.message)
			}
			if stdout != "" || !strings.Contains(stderr, tc.message) {
				t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
			}
			if _, err := os.Stat(filepath.Join(dir, ".shemiq")); !os.IsNotExist(err) {
				t.Fatalf("invalid input created .shemiq: %v", err)
			}
		})
	}
}

func TestTaskNewDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, ".shemiq", "tasks", "same-task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(taskDir, "top-level.md")
	if err := os.WriteFile(path, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	stdout, stderr, err := runCommand("Same Task\n", "task", "new", "new description")
	if err == nil || !strings.Contains(stderr, "file exists") || stdout != "" {
		t.Fatalf("error = %v, stdout = %q, stderr = %q", err, stdout, stderr)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "keep me" {
		t.Fatalf("document = %q, error = %v", content, err)
	}
}

func runCommand(input string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := Execute(strings.NewReader(input), &stdout, &stderr, args)
	return stdout.String(), stderr.String(), err
}
