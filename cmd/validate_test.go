package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

const testUUID = "12345678-1234-4234-8234-123456789abc"

func TestValidateScopeAndDuplicates(t *testing.T) {
	project := t.TempDir()
	shemiq := filepath.Join(project, ".shemiq")
	dir := filepath.Join(shemiq, "tasks")
	withUUID := fmt.Sprintf(utils.Dedent(`
		# Task
		:::shemiq
		type: task
		uuid: %s
		:::
		`), testUUID)
	a := writeTestMarkdown(t,
		filepath.Join(dir, "a", "top-level.md"), withUUID)
	b := writeTestMarkdown(t,
		filepath.Join(dir, "nested", "b.md"), withUUID)
	writeTestMarkdown(t,
		filepath.Join(dir, "nested", "plain.md"),
		utils.Dedent(`
		No metadata
		`))
	// Archived metadata is excluded by default scope.
	writeTestMarkdown(t,
		filepath.Join(shemiq, "archive", "old", "top-level.md"),
		withUUID)
	cwd := filepath.Join(project, "subdir")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	for _, args := range [][]string{
		{"validate", a},
		{"validate", filepath.Dir(b)},
	} {
		out, diag, err := runCommand("", args...)
		if err != nil || out != "" || diag != "" {
			t.Fatalf("validate %v: out=%q diag=%q err=%v",
				args, out, diag, err)
		}
	}
	for _, args := range [][]string{
		{"validate"},
		{"validate", dir},
	} {
		out, diag, err := runCommand("", args...)
		if err == nil || out != "" ||
			!strings.Contains(diag, "duplicate uuid") ||
			strings.Contains(diag, "Error:") ||
			strings.Contains(diag, "Usage:") {
			t.Fatalf("validate %v: out=%q diag=%q err=%v",
				args, out, diag, err)
		}
	}
}

func TestValidateFixPreservesBytesAndReportsRemainingErrors(
	t *testing.T,
) {
	dir := t.TempDir()
	original := strings.ReplaceAll(utils.Dedent(`
		# Title
		:::shemiq
		type: task
		status: todo
		:::
		suffix`), "\n", "\r\n")
	path := writeTestMarkdown(t,
		filepath.Join(dir, "metadata.md"), original)
	t.Chdir(dir)
	out, diag, err := runCommand("",
		"validate", "--fix", path)
	// Status field is now on line 4 (after type: task)
	if err == nil || out != "" ||
		!strings.Contains(diag,
			path+":2: fixed: missing uuid") ||
		!strings.Contains(diag,
			"invalid status: todo") ||
		strings.Contains(diag, "Error:") {
		t.Fatalf("fix: out=%q diag=%q err=%v",
			out, diag, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	uuidLine := regexp.MustCompile(
		`uuid: [0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-` +
			`[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Find(content)
	if uuidLine == nil ||
		strings.Replace(string(content),
			string(uuidLine)+"\r\n", "", 1) != original {
		t.Fatalf("unexpected repair: %q", content)
	}
	_, diag, err = runCommand("",
		"validate", path, "--fix")
	if err == nil ||
		strings.Contains(diag, "fixed:") ||
		!strings.Contains(diag, "invalid status: todo") {
		t.Fatalf("second fix: diag=%q err=%v",
			diag, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(content) {
		t.Fatalf("second fix changed document: %q, %v",
			after, err)
	}
	if err := os.WriteFile(path, []byte(
		strings.ReplaceAll(string(content),
			"status: todo", "status: done")),
		0644); err != nil {
		t.Fatal(err)
	}
	out, diag, err = runCommand("", "validate", path)
	if err != nil || out != "" || diag != "" {
		t.Fatalf("valid document: out=%q diag=%q err=%v",
			out, diag, err)
	}
}

func TestValidateSuccessfulFixAndSourceException(
	t *testing.T,
) {
	dir := t.TempDir()
	path := writeTestMarkdown(t,
		filepath.Join(dir, "task.md"),
		utils.Dedent(`
		# Task
		:::shemiq
		type: task
		:::
		## Tasks
		### Sub
		:::shemiq
		source: ./future.md
		:::
		`))
	t.Chdir(dir)
	out, diag, err := runCommand("",
		"validate", path, "--fix")
	// Primary directive is eligible (no source, no uuid);
	// subtask directive has source, so not eligible.
	if err != nil || out != "" ||
		strings.Count(diag, "fixed: missing uuid") != 1 ||
		strings.Contains(diag, "Error:") {
		t.Fatalf("fix: out=%q diag=%q err=%v",
			out, diag, err)
	}
	content, err := os.ReadFile(path)
	if err != nil ||
		strings.Count(string(content), "uuid: ") != 1 {
		t.Fatalf("repair: %q, %v", content, err)
	}
	out, diag, err = runCommand("",
		"validate", path, "--fix")
	if err != nil || out != "" || diag != "" {
		t.Fatalf("repeat fix: out=%q diag=%q err=%v",
			out, diag, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(content) {
		t.Fatalf("repeat fix changed file: %q, %v",
			after, err)
	}
}

func TestValidateLinkedStatusLifecycle(t *testing.T) {
	dir := t.TempDir()
	parent := writeTestMarkdown(t,
		filepath.Join(dir, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### First
		:::shemiq
		type: task
		source: ./first.md
		status: new
		:::
		### Future
		:::shemiq
		type: task
		source: ./future.md
		:::
		`), testUUID))
	child := writeTestMarkdown(t,
		filepath.Join(dir, "first.md"),
		utils.Dedent(`
		# First
		:::shemiq
		type: task
		parent: ./top-level.md
		status: refined
		:::
		Some content to preserve.
		`))
	t.Chdir(dir)
	for _, path := range []string{parent, child} {
		_, diag, err := runCommand("",
			"validate", path)
		if err == nil ||
			strings.Count(diag, "status mismatch") != 1 ||
			!strings.Contains(diag, parent) ||
			strings.Contains(diag, "future.md") {
			t.Fatalf("validate %s: diag=%q err=%v",
				path, diag, err)
		}
	}
	_, diag, err := runCommand("",
		"validate", "--fix", child)
	if err != nil || !strings.Contains(diag,
		parent+":11: fixed: status promoted to refined") {
		t.Fatalf("fix from child: diag=%q err=%v",
			diag, err)
	}
	fixedParent, err := os.ReadFile(parent)
	if err != nil || !strings.Contains(
		string(fixedParent),
		"source: ./first.md\nstatus: refined\n") {
		t.Fatalf("parent not promoted: %q, %v",
			fixedParent, err)
	}
	// Promote an omitted child status to done while
	// retaining unrelated bytes.
	if err := os.WriteFile(child,
		[]byte(strings.ReplaceAll(utils.Dedent(`
		# First
		:::shemiq
		type: task
		parent: ./top-level.md
		:::
		Some content to preserve.
		`), "\n", "\r\n")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent,
		[]byte(strings.Replace(string(fixedParent),
			"status: refined", "status: done", 1)),
		0644); err != nil {
		t.Fatal(err)
	}
	_, diag, err = runCommand("",
		"validate", "--fix", parent)
	if err != nil || !strings.Contains(diag,
		child+":2: fixed: status promoted to done") {
		t.Fatalf("fix omitted child status: diag=%q err=%v",
			diag, err)
	}
	fixedChild, err := os.ReadFile(child)
	if err != nil ||
		!strings.Contains(string(fixedChild),
			"parent: ./top-level.md\r\nuuid: ") ||
		!strings.Contains(string(fixedChild),
			"\r\nstatus: done\r\n:::\r\n"+
				"Some content to preserve.") {
		t.Fatalf(
			"child not promoted or content changed: %q, %v",
			fixedChild, err)
	}
	_, diag, err = runCommand("",
		"validate", "--fix", child)
	if err != nil || diag != "" {
		t.Fatalf("second fix: diag=%q err=%v",
			diag, err)
	}
}

func TestValidateExistingSourceWithoutBacklink(
	t *testing.T,
) {
	dir := t.TempDir()
	parent := writeTestMarkdown(t,
		filepath.Join(dir, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### Child
		:::shemiq
		type: task
		source: ./child.md
		:::
		`), testUUID))
	writeTestMarkdown(t,
		filepath.Join(dir, "child.md"),
		utils.Dedent(`
		# Child
		:::shemiq
		type: task
		uuid: 87654321-4321-4321-8321-abcdefabcdef
		:::
		`))
	t.Chdir(dir)
	_, diag, err := runCommand("", "validate", parent)
	if err == nil || !strings.Contains(diag,
		"source ./child.md does not have "+
			"exactly one reciprocal task parent") {
		t.Fatalf("missing backlink: diag=%q err=%v",
			diag, err)
	}
}

func TestValidateOneSidedAndInvalidStatuses(
	t *testing.T,
) {
	dir := t.TempDir()
	parent := writeTestMarkdown(t,
		filepath.Join(dir, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### Child
		:::shemiq
		type: task
		source: ./child.md
		status: progress
		:::
		`), testUUID))
	writeTestMarkdown(t,
		filepath.Join(dir, "child.md"),
		utils.Dedent(`
		# Child
		:::shemiq
		type: task
		parent: ./other.md
		status: done
		:::
		`))
	writeTestMarkdown(t,
		filepath.Join(dir, "other.md"),
		fmt.Sprintf(utils.Dedent(`
		# Other
		:::shemiq
		type: top-level
		uuid: %s
		:::
		`), "87654321-4321-4321-8321-abcdefabcdef"))
	t.Chdir(dir)
	_, diag, err := runCommand("",
		"validate", "--fix", parent)
	if err == nil ||
		!strings.Contains(diag, "invalid status: progress") ||
		!strings.Contains(diag,
			"does not have exactly one reciprocal task") ||
		strings.Contains(diag, "status mismatch") ||
		strings.Contains(diag, "fixed: status") {
		t.Fatalf("one-sided: diag=%q err=%v",
			diag, err)
	}
	content, err := os.ReadFile(parent)
	if err != nil ||
		!strings.Contains(
			string(content), "status: progress") {
		t.Fatalf("invalid status changed: %q, %v",
			content, err)
	}
	// Even when links become reciprocal, an invalid status
	// cannot choose a winner.
	child := filepath.Join(dir, "child.md")
	if err := os.WriteFile(child, []byte(
		utils.Dedent(`
		# Child
		:::shemiq
		type: task
		parent: ./top-level.md
		status: done
		:::
		`)), 0644); err != nil {
		t.Fatal(err)
	}
	_, diag, err = runCommand("",
		"validate", "--fix", child)
	if err == nil ||
		!strings.Contains(diag, "invalid status: progress") ||
		strings.Contains(diag, "status mismatch") ||
		strings.Contains(diag, "fixed: status") {
		t.Fatalf("invalid reciprocal status: diag=%q err=%v",
			diag, err)
	}
}

func TestValidateInvalidMetadataAndReferences(
	t *testing.T,
) {
	dir := t.TempDir()
	original := fmt.Sprintf(utils.Dedent(`
		# Task
		:::shemiq
		type: other
		status: todo
		status: done
		unknown: x
		uuid: INVALID
		parent: ./missing.md
		:::
		## Tasks
		### Sub1
		:::shemiq
		source: ./not-yet-written.md
		:::
		### Sub2
		:::shemiq
		source: ./later.md
		uuid: %s
		:::
		`), testUUID)
	path := writeTestMarkdown(t,
		filepath.Join(dir, "bad.md"), original)
	t.Chdir(dir)
	out, diag, err := runCommand("",
		"validate", path, "--fix")
	if err == nil || out != "" {
		t.Fatalf("invalid metadata: out=%q diag=%q err=%v",
			out, diag, err)
	}
	for _, message := range []string{
		"invalid type",
		"invalid status",
		"repeated metadata field",
		"unknown metadata field",
		"invalid UUIDv4",
		"parent is not an existing file",
		"source and uuid are mutually exclusive",
	} {
		if !strings.Contains(diag, message) {
			t.Errorf("missing %q in %q", message, diag)
		}
	}
	if strings.Contains(diag, "fixed:") ||
		strings.Contains(diag, "not-yet-written") {
		t.Fatalf("unexpected findings: %q", diag)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != original {
		t.Fatalf(
			"invalid block changed unexpectedly: %q, %v",
			content, err)
	}
}

// An existing non-Markdown or directory target is a located
// invalid-reference finding. An archived file reached through
// a reference is loaded even though default scope excludes it.
func TestValidateNonMarkdownAndReachableArchive(
	t *testing.T,
) {
	project := t.TempDir()
	tasks := filepath.Join(
		project, ".shemiq", "tasks", "example")
	writeTestMarkdown(t,
		filepath.Join(project,
			".shemiq", "archive", "old", "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Old
		:::shemiq
		type: task
		uuid: %s
		:::
		`), testUUID))
	parent := writeTestMarkdown(t,
		filepath.Join(tasks, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### Notes
		:::shemiq
		type: task
		source: ./notes.txt
		:::
		### Archive ref
		:::shemiq
		type: task
		source: ../../../.shemiq/archive/old/top-level.md
		:::
		`), testUUID))
	if err := os.WriteFile(
		filepath.Join(tasks, "notes.txt"),
		[]byte("plain"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tasks)
	_, diag, err := runCommand("", "validate", parent)
	if err == nil ||
		!strings.Contains(diag,
			"source is not a Markdown file: ./notes.txt") ||
		!strings.Contains(diag, "duplicate uuid") {
		t.Fatalf("references: diag=%q err=%v",
			diag, err)
	}
}

// All eligible promotions complete in one run.
func TestValidateLinkedGroupConverges(t *testing.T) {
	dir := t.TempDir()
	parent := writeTestMarkdown(t,
		filepath.Join(dir, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### First
		:::shemiq
		type: task
		source: ./first.md
		status: done
		:::
		### Second
		:::shemiq
		type: task
		source: ./second.md
		status: new
		:::
		`), testUUID))
	writeTestMarkdown(t,
		filepath.Join(dir, "first.md"),
		utils.Dedent(`
		# First
		:::shemiq
		type: task
		parent: ./top-level.md
		status: new
		:::
		`))
	writeTestMarkdown(t,
		filepath.Join(dir, "second.md"),
		utils.Dedent(`
		# Second
		:::shemiq
		type: task
		parent: ./top-level.md
		status: refined
		:::
		`))
	t.Chdir(dir)
	_, diag, err := runCommand("",
		"validate", "--fix", parent)
	if err != nil ||
		strings.Count(diag,
			"fixed: status promoted to done") != 1 ||
		strings.Count(diag,
			"fixed: status promoted to refined") != 1 {
		t.Fatalf("one-run convergence: diag=%q err=%v",
			diag, err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "first.md"))
	second, _ := os.ReadFile(
		filepath.Join(dir, "second.md"))
	if strings.Count(string(first), "status: done") != 1 ||
		strings.Count(
			string(second), "status: refined") != 1 {
		t.Fatalf("unexpected convergence: %q %q",
			first, second)
	}
	_, diag, err = runCommand("",
		"validate", "--fix", parent)
	if err != nil || diag != "" {
		t.Fatalf("second run not idempotent: diag=%q err=%v",
			diag, err)
	}
}

func TestValidateNoProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out, diag, err := runCommand("", "validate")
	if err == nil || out != "" ||
		!strings.Contains(diag, "no .shemiq directory") {
		t.Fatalf("out=%q diag=%q err=%v",
			out, diag, err)
	}
	if _, statErr := os.Stat(
		filepath.Join(dir, ".shemiq"),
	); !os.IsNotExist(statErr) {
		t.Fatalf("validation created project: %v",
			statErr)
	}
}

// A malformed field is a fatal syntax error.
func TestValidateFatalSyntaxStopsValidation(
	t *testing.T,
) {
	dir := t.TempDir()
	t.Chdir(dir)
	broken := writeTestMarkdown(t,
		filepath.Join(dir, "broken.md"),
		utils.Dedent(`
		# Broken
		:::shemiq
		not a field
		:::
		`))
	_, diag, err := runCommand("",
		"validate", "--fix", broken)
	if err == nil ||
		!strings.Contains(diag,
			"malformed metadata field") ||
		strings.Contains(diag, "fixed:") {
		t.Fatalf("direct syntax error: diag=%q err=%v",
			diag, err)
	}
	before, _ := os.ReadFile(broken)
	parent := writeTestMarkdown(t,
		filepath.Join(dir, "top-level.md"),
		fmt.Sprintf(utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: %s
		:::
		## Tasks
		### Sub
		:::shemiq
		type: task
		source: ./broken.md
		:::
		`), testUUID))
	_, diag, err = runCommand("",
		"validate", "--fix", parent)
	if err == nil ||
		!strings.Contains(diag,
			"malformed metadata field") ||
		strings.Contains(diag, "fixed:") {
		t.Fatalf("referenced syntax error: diag=%q err=%v",
			diag, err)
	}
	after, _ := os.ReadFile(broken)
	if string(before) != string(after) {
		t.Fatalf("syntax failure performed repairs: %q",
			after)
	}
}

func writeTestMarkdown(
	t *testing.T, path, content string,
) string {
	t.Helper()
	if err := os.MkdirAll(
		filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
