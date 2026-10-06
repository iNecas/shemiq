package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

var newTaskUUIDLine = regexp.MustCompile(`(?m)^uuid: [0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewDiscoversProject(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".shemiq"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(project, "src", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	stdout, stderr, err := runCommand("  My New Task!  \n", "new", "Describe", "the task")
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
	if !newTaskUUIDLine.Match(content) {
		t.Fatalf("document lacks a canonical UUIDv4: %s", content)
	}
	want := utils.Dedent(`
		# My New Task!
		:::shemiq
		type: top-level
		uuid: <generated>
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
		`)
	if newTaskUUIDLine.ReplaceAllString(string(content), "uuid: <generated>") != want {
		t.Fatalf("document = %q, want %q (with generated UUID)", content, want)
	}
	out, diag, err := runCommand("", "validate", path)
	if err != nil || out != "" || diag != "" {
		t.Fatalf("new task failed validation: out=%q diag=%q err=%v", out, diag, err)
	}
}

func TestNewWithTitleFlag(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	stdout, stderr, err := runCommand("", "new", "--title", "Command Line Title", "Describe", "the task")
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
	if !newTaskUUIDLine.Match(content) ||
		!strings.HasPrefix(newTaskUUIDLine.ReplaceAllString(string(content), "uuid: <generated>"), utils.Dedent(`
			# Command Line Title
			:::shemiq
			type: top-level
			uuid: <generated>
			:::
			`)) ||
		!strings.Contains(string(content), utils.Dedent(`
			## Description

			Describe the task
			`)) {
		t.Fatalf("unexpected document: %s", content)
	}
}

func TestNewCreatesProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	stdout, stderr, err := runCommand("Héllo  123\n", "new", "--", "--some", "description")
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
	if !strings.Contains(string(content), utils.Dedent(`
		## Description

		--some description
		`)) {
		t.Fatalf("document does not contain description: %s", content)
	}
}

func TestNewRejectsMissingInput(t *testing.T) {
	for _, tc := range []struct {
		name, input, message string
		args                 []string
	}{
		{"missing description", "Unused\n", "requires at least 1 arg(s)", []string{"new"}},
		{"blank description", "Unused\n", "description is required", []string{"new", "  "}},
		{"blank title", "  \n", "title is required", []string{"new", "description"}},
		{"blank title flag", "Fallback\n", "title is required", []string{"new", "--title", "  ", "description"}},
		{"multiline title flag", "", "title must be one line", []string{"new", "--title", utils.Dedent(`
			First
			Second`), "description"}},
		{"EOF without title", "", "title is required", []string{"new", "description"}},
		{"empty slug", "é !\n", "ASCII letter or digit", []string{"new", "description"}},
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

func TestRemovedTaskRoute(t *testing.T) {
	stdout, stderr, err := runCommand("", "task", "new", "description")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("error = %v, want unknown command", err)
	}
	if stdout != "" || !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
	}
}

func TestBareCommandPointsToNew(t *testing.T) {
	stdout, stderr, err := runCommand("")
	if err == nil || !strings.Contains(err.Error(), "shemiq new <description...>") {
		t.Fatalf("error = %v, want shemiq new guidance", err)
	}
	if stdout != "" || !strings.Contains(stderr, "shemiq new <description...>") {
		t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
	}
}

func TestNewDoesNotOverwrite(t *testing.T) {
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
	stdout, stderr, err := runCommand("Same Task\n", "new", "new description")
	if err == nil || !strings.Contains(stderr, "file exists") || stdout != "" {
		t.Fatalf("error = %v, stdout = %q, stderr = %q", err, stdout, stderr)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "keep me" {
		t.Fatalf("document = %q, error = %v", content, err)
	}
}
