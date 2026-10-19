package task

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestCreateSubtaskCacheRefresh(t *testing.T) {
	root := t.TempDir()
	stagingRoot := t.TempDir()
	t.Setenv("TMPDIR", stagingRoot)
	active := filepath.Join(root, ".shemiq", "tasks")
	parentPath := storeFixture(t, active, "feat/top-level.md",
		utils.Dedent(`
		# Feature
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
		:::

		## Description

		A feature.

		## Tasks

		[TBD]
		`))

	store, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parentPath, 0640); err != nil {
		t.Fatal(err)
	}
	// Force load so cache is populated.
	if _, err := store.TopLevelTasks(); err != nil {
		t.Fatal(err)
	}
	parent, err := store.TaskByPath(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(parent.Subtasks) != 0 {
		t.Fatalf("expected 0 subtasks, got %d",
			len(parent.Subtasks))
	}

	// Insert first subtask (replaces [TBD]).
	result, err := store.CreateSubtask(parentPath, "First Task", "Do the first thing.", "")
	if err != nil {
		t.Fatalf("create subtask: %v", err)
	}
	if result != parentPath {
		t.Fatalf("path = %q, want %q", result, parentPath)
	}

	// Query the same store to verify cache refresh.
	refreshed, err := store.TaskByPath(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Subtasks) != 1 {
		t.Fatalf("expected 1 subtask after insert, got %d",
			len(refreshed.Subtasks))
	}
	if refreshed.Subtasks[0].Title != "First Task" {
		t.Fatalf("subtask title = %q, want %q",
			refreshed.Subtasks[0].Title, "First Task")
	}
	if refreshed.Subtasks[0].Status != "new" {
		t.Fatalf("subtask status = %q, want %q",
			refreshed.Subtasks[0].Status, "new")
	}

	// Insert second subtask (appends after first).
	_, err = store.CreateSubtask(
		parentPath, "Second Task",
		"Do the second thing.", "")
	if err != nil {
		t.Fatalf("create second subtask: %v", err)
	}
	refreshed, err = store.TaskByPath(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Subtasks) != 2 {
		t.Fatalf("expected 2 subtasks, got %d",
			len(refreshed.Subtasks))
	}
	if refreshed.Subtasks[1].Title != "Second Task" {
		t.Fatalf("second subtask title = %q",
			refreshed.Subtasks[1].Title)
	}

	// Insert with --after.
	_, err = store.CreateSubtask(parentPath, "Middle Task", "Goes between.", "First Task")
	if err != nil {
		t.Fatalf("create middle subtask: %v", err)
	}
	refreshed, err = store.TaskByPath(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Subtasks) != 3 {
		t.Fatalf("expected 3 subtasks, got %d",
			len(refreshed.Subtasks))
	}
	if refreshed.Subtasks[0].Title != "First Task" ||
		refreshed.Subtasks[1].Title != "Middle Task" ||
		refreshed.Subtasks[2].Title != "Second Task" {
		titles := make([]string, len(refreshed.Subtasks))
		for i, s := range refreshed.Subtasks {
			titles[i] = s.Title
		}
		t.Fatalf("unexpected order: %v", titles)
	}

	// Verify the written file passes validation.
	data, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "[TBD]") {
		t.Fatal("placeholder was not replaced")
	}
	_, parseErr := parseMarkdownDocument(parentPath, data)
	if parseErr != nil {
		t.Fatalf("written file does not parse: %v", parseErr)
	}
	info, err := os.Stat(parentPath)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("permissions not preserved: info=%v err=%v", info, err)
	}
	entries, err := os.ReadDir(stagingRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("staging files not cleaned up: entries=%v err=%v", entries, err)
	}
}

func TestCreateSubtaskRejectsInvalid(t *testing.T) {
	root := t.TempDir()
	stagingRoot := t.TempDir()
	t.Setenv("TMPDIR", stagingRoot)
	active := filepath.Join(root, ".shemiq", "tasks")
	parentPath := storeFixture(t, active, "feat/top-level.md",
		utils.Dedent(`
		# Feature
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
		:::

		## Tasks

		### Existing
		
		Already here.
		`))

	store, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}

	original, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}

	// Parent errors take precedence over invalid input.
	if _, err := store.CreateSubtask(
		filepath.Join(active, "missing.md"), "", "", ""); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected parent path error before input validation, got %v", err)
	}

	// Directory resolution belongs to the CLI, not the Store.
	if _, err := store.CreateSubtask(
		filepath.Dir(parentPath), "New", "desc", ""); err == nil {
		t.Fatal("expected directory parent rejection")
	}

	// Uniqueness is checked after trimming the title.
	if _, err := store.CreateSubtask(
		parentPath, " Existing ", "desc", ""); err == nil {
		t.Fatal("expected duplicate title error")
	}

	// Missing --after target.
	if _, err := store.CreateSubtask(
		parentPath, "New", "desc", "Nonexistent"); err == nil {
		t.Fatal("expected --after not found error")
	}

	// Empty title.
	if _, err := store.CreateSubtask(
		parentPath, "", "desc", ""); err == nil {
		t.Fatal("expected empty title error")
	}

	// Empty description.
	if _, err := store.CreateSubtask(
		parentPath, "New", "", ""); err == nil {
		t.Fatal("expected empty description error")
	}

	// Structural rejection happens while checking the staged file.
	if _, err := store.CreateSubtask(
		parentPath, "New", "Description.\n\n### Extra task\n", ""); err == nil {
		t.Fatal("expected disruptive Markdown error")
	}

	// Verify file unchanged after rejections.
	data, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("file was modified by rejected request")
	}
	entries, err := os.ReadDir(stagingRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected staging files not cleaned up: entries=%v err=%v", entries, err)
	}
	entries, err = os.ReadDir(filepath.Dir(parentPath))
	if err != nil || len(entries) != 1 {
		t.Fatalf("rejected edits polluted the task directory: entries=%v err=%v", entries, err)
	}
}

func TestCreateSubtaskMissingTasksSection(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, ".shemiq", "tasks")
	parentPath := storeFixture(t, active, "feat/top-level.md",
		utils.Dedent(`
		# Feature
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
		:::

		## Description

		No tasks section here.
		`))

	store, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateSubtask(
		parentPath, "New Task", "A new task.", "")
	if err != nil {
		t.Fatalf("missing tasks section: %v", err)
	}
	data, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !containsString(content, "## Tasks") {
		t.Fatal("## Tasks section not appended")
	}
	if !containsString(content, "### New Task") {
		t.Fatal("subtask not inserted")
	}
}

func TestCreateSubtaskPreservesContent(t *testing.T) {
	root := t.TempDir()
	original := utils.Dedent(`
		# Feature
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
		status: done
		:::

		## Description

		Unrelated content.

		## Tasks

		### First Task
		:::shemiq
		type: task
		source: ./first-task.md
		status: done
		:::

		First description.

		#### Nested heading

		Nested content.

		### Second Task

		Second description.

		## Notes

		Keep these notes.
		`)
	parentPath := storeFixture(t, root, ".shemiq/tasks/feat/top-level.md", original)
	store, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateSubtask(parentPath, "Middle Task", "Inserted description.", "First Task"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(original, "### Second Task",
		"### Middle Task\n\nInserted description.\n### Second Task", 1)
	if string(data) != want {
		t.Fatalf("insertion changed unrelated bytes or split the sibling section:\ngot: %q\nwant: %q", data, want)
	}
	entries, err := os.ReadDir(filepath.Dir(parentPath))
	if err != nil || len(entries) != 1 {
		t.Fatalf("insertion created extra files: entries=%v err=%v", entries, err)
	}
}

func TestCreateSubtaskInsertBeforeDirectiveSibling(t *testing.T) {
	root := t.TempDir()
	original := utils.Dedent(`
		# Feature
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
		:::

		## Tasks

		### First Task

		First description.

		### Second Task
		:::shemiq
		type: task
		source: ./second-task.md
		status: done
		:::

		Second description.
		`)
	parentPath := storeFixture(t, root, ".shemiq/tasks/feat/top-level.md", original)
	store, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	// The insertion shifts the Second Task directive to later lines;
	// verification must still recognize it as existing.
	if _, err := store.CreateSubtask(parentPath, "Middle Task", "Inserted description.", "First Task"); err != nil {
		t.Fatalf("insert before directive sibling: %v", err)
	}

	data, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(original, "### Second Task",
		"### Middle Task\n\nInserted description.\n### Second Task", 1)
	if string(data) != want {
		t.Fatalf("insertion changed unrelated bytes:\ngot: %q\nwant: %q", data, want)
	}
	refreshed, err := store.TaskByPath(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Subtasks) != 3 || refreshed.Subtasks[1].Title != "Middle Task" ||
		refreshed.Subtasks[2].Status != "done" {
		t.Fatalf("unexpected refreshed subtasks: %+v", refreshed.Subtasks)
	}
}

func TestCreateSubtaskAfterRequiresSibling(t *testing.T) {
	for _, tc := range []struct {
		name, tasks string
	}{
		{"placeholder", "## Tasks\n\n[TBD]\n"},
		{"missing section", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			original := utils.Dedent(`
				# Feature
				:::shemiq
				type: top-level
				uuid: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
				:::

				`) + tc.tasks
			parentPath := storeFixture(t, root, ".shemiq/tasks/feat/top-level.md", original)
			store, err := NewStore(root, "")
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.CreateSubtask(parentPath, "New", "Description.", "Nonexistent")
			if err == nil || !strings.Contains(err.Error(), "--after sibling not found") {
				t.Fatalf("error=%v, want missing sibling rejection", err)
			}
			data, err := os.ReadFile(parentPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != original {
				t.Fatal("rejected insertion modified the parent")
			}
		})
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		// Use strings.Contains equivalent.
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}
