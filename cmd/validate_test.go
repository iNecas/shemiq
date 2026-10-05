package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const testUUID = "12345678-1234-4234-8234-123456789abc"

func TestValidateScopeAndDuplicates(t *testing.T) {
	project := t.TempDir()
	dir := filepath.Join(project, ".shemiq")
	a := writeTestMarkdown(t, filepath.Join(dir, "a.md"), ":::shemiq\nuuid: "+testUUID+"\n:::\n")
	b := writeTestMarkdown(t, filepath.Join(dir, "nested", "b.md"), ":::shemiq\nuuid: "+testUUID+"\n:::\n")
	writeTestMarkdown(t, filepath.Join(dir, "nested", "plain.md"), "No metadata\n")
	cwd := filepath.Join(project, "subdir")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	for _, args := range [][]string{{"validate", a}, {"validate", filepath.Dir(b)}} {
		out, diag, err := runCommand("", args...)
		if err != nil || out != "" || diag != "" {
			t.Fatalf("validate %v: out=%q diag=%q err=%v", args, out, diag, err)
		}
	}
	for _, args := range [][]string{{"validate"}, {"validate", dir}} {
		out, diag, err := runCommand("", args...)
		if err == nil || out != "" || !strings.Contains(diag, b+":2: duplicate uuid") || strings.Contains(diag, "Error:") || strings.Contains(diag, "Usage:") {
			t.Fatalf("validate %v: out=%q diag=%q err=%v", args, out, diag, err)
		}
	}
}

func TestValidateFixPreservesBytesAndReportsRemainingErrors(t *testing.T) {
	dir := t.TempDir()
	path := writeTestMarkdown(t, filepath.Join(dir, "metadata.md"), "prefix\r\n    :::shemiq\r\n:::shemiq\r\nstatus: todo\r\n:::\r\nsuffix")
	t.Chdir(dir)
	out, diag, err := runCommand("", "validate", "--fix", path)
	if err == nil || out != "" || !strings.Contains(diag, path+":3: fixed: missing uuid") || !strings.Contains(diag, path+":4: invalid status: todo") || strings.Contains(diag, "Error:") {
		t.Fatalf("fix: out=%q diag=%q err=%v", out, diag, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^prefix\r\n    :::shemiq\r\n:::shemiq\r\nstatus: todo\r\nuuid: [0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\r\n:::\r\nsuffix$`)
	if !pattern.Match(content) {
		t.Fatalf("unexpected repair: %q", content)
	}
	_, diag, err = runCommand("", "validate", path, "--fix")
	if err == nil || strings.Contains(diag, "fixed:") || !strings.Contains(diag, "invalid status: todo") {
		t.Fatalf("second fix: diag=%q err=%v", diag, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(content) {
		t.Fatalf("second fix changed document: %q, %v", after, err)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(content), "status: todo", "status: done")), 0644); err != nil {
		t.Fatal(err)
	}
	out, diag, err = runCommand("", "validate", path)
	if err != nil || out != "" || diag != "" {
		t.Fatalf("valid document: out=%q diag=%q err=%v", out, diag, err)
	}
}

func TestValidateSuccessfulFixAndSourceException(t *testing.T) {
	dir := t.TempDir()
	path := writeTestMarkdown(t, filepath.Join(dir, "task.md"), ":::shemiq\ntype: task\n:::\n:::shemiq\nstatus: progress\n:::\n:::shemiq\nsource: ./future.md\n:::\n")
	t.Chdir(dir)
	out, diag, err := runCommand("", "validate", path, "--fix")
	if err != nil || out != "" || strings.Count(diag, "fixed: missing uuid") != 2 || strings.Contains(diag, "Error:") {
		t.Fatalf("fix: out=%q diag=%q err=%v", out, diag, err)
	}
	content, err := os.ReadFile(path)
	if err != nil || strings.Count(string(content), "uuid: ") != 2 {
		t.Fatalf("repair: %q, %v", content, err)
	}
	out, diag, err = runCommand("", "validate", path, "--fix")
	if err != nil || out != "" || diag != "" {
		t.Fatalf("repeat fix: out=%q diag=%q err=%v", out, diag, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(content) {
		t.Fatalf("repeat fix changed file: %q, %v", after, err)
	}
}

func TestValidateInvalidMetadataAndReferences(t *testing.T) {
	dir := t.TempDir()
	original := ":::shemiq\ntype: other\nstatus: todo\nstatus: done\nunknown: x\nuuid: INVALID\nparent: ./missing.md\n:::\n:::shemiq\nsource: ./not-yet-written.md\n:::\n:::shemiq\nsource: ./later.md\nuuid: " + testUUID + "\n:::\n:::shemiq\nnot a field\n:::\n"
	path := writeTestMarkdown(t, filepath.Join(dir, "bad.md"), original)
	t.Chdir(dir)
	out, diag, err := runCommand("", "validate", path, "--fix")
	if err == nil || out != "" {
		t.Fatalf("invalid metadata: out=%q diag=%q err=%v", out, diag, err)
	}
	for _, message := range []string{"invalid type", "invalid status", "repeated metadata field", "unknown metadata field", "invalid UUIDv4", "parent is not an existing file", "source and uuid are mutually exclusive", "malformed metadata field", "missing uuid"} {
		if !strings.Contains(diag, message) {
			t.Errorf("missing %q in %q", message, diag)
		}
	}
	if strings.Contains(diag, "fixed:") || strings.Contains(diag, "not-yet-written") {
		t.Fatalf("unexpected findings: %q", diag)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != original {
		t.Fatalf("invalid block changed unexpectedly: %q, %v", content, err)
	}
}

func TestValidateNoProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out, diag, err := runCommand("", "validate")
	if err == nil || out != "" || !strings.Contains(diag, "no .shemiq directory") {
		t.Fatalf("out=%q diag=%q err=%v", out, diag, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".shemiq")); !os.IsNotExist(statErr) {
		t.Fatalf("validation created project: %v", statErr)
	}
}

func writeTestMarkdown(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
