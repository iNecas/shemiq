package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestStoreScopeAndSnapshots(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, "")
	if err != nil || store.scope != nil ||
		len(store.tasks) != 0 {
		t.Fatalf(
			"construction must be lazy: store=%+v err=%v",
			store, err)
	}
	if _, err := store.TopLevelTasks(); err == nil {
		t.Fatal("read without a project succeeded")
	}
	if _, err := os.Stat(
		filepath.Join(root, ".shemiq"),
	); !os.IsNotExist(err) {
		t.Fatal("read created a project")
	}
	active := filepath.Join(root, ".shemiq", "tasks")
	a := storeFixture(t, active, "a/top-level.md",
		utils.Dedent(`
		# A
		:::shemiq
		type: task
		:::
		`))
	b := storeFixture(t, active, "b/top-level.md",
		utils.Dedent(`
		# B
		:::shemiq
		type: top-level
		status: done
		:::
		`))
	archived := storeFixture(t, root,
		".shemiq/archive/old/top-level.md",
		utils.Dedent(`
		# Archived
		:::shemiq
		type: top-level
		:::
		`))
	tasks, err := store.TopLevelTasks()
	if err != nil || len(tasks) != 1 || tasks[0].Path != b {
		t.Fatalf("default candidates: tasks=%+v err=%v",
			tasks, err)
	}
	aTask, err := store.TaskByPath(a)
	if err != nil {
		t.Fatal(err)
	}
	// Load out-of-scope reference; it must not widen queries.
	if _, err := store.loadTask(
		archived, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TaskByPath(archived); err == nil {
		t.Fatal("cached archive escaped default query scope")
	}
	// Alias and snapshot caching
	alias := filepath.Join(active, "b", "alias.md")
	if err := os.Symlink(b, alias); err != nil {
		t.Fatal(err)
	}
	cached, err := store.TaskByPath(alias)
	if err != nil || cached.Title != "B" ||
		cached.Status != "done" || cached.Path != b ||
		cached.document != tasks[0].document {
		t.Fatalf(
			"alias did not reuse snapshot: task=%+v err=%v",
			cached, err)
	}
	// Write bad content to test failed refresh
	if err := os.WriteFile(b, []byte(utils.Dedent(`
		# Bad
		:::shemiq
		malformed
		`)), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.loadTask(
		b, true); err == nil {
		t.Fatal("private refresh did not see syntax error")
	}
	if store.tasks[b] == nil ||
		store.tasks[b].document != cached.document {
		t.Fatal("failed refresh replaced the cached snapshot")
	}
	// Successful refresh
	storeFixture(t, active, "b/top-level.md",
		utils.Dedent(`
		# Updated
		:::shemiq
		type: top-level
		status: refined
		:::
		`))
	refreshed, err := store.loadTask(b, true)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.TaskByPath(alias)
	if err != nil || refreshed.Title != "Updated" ||
		updated.Status != "refined" ||
		updated.document != refreshed.document ||
		updated.document == cached.document {
		t.Fatalf(
			"refresh did not replace task: task=%+v err=%v",
			updated, err)
	}
	if len(store.tasks) != 3 ||
		store.tasks[a].document != aTask.document {
		t.Fatal("refresh lost other tasks' snapshots")
	}
	// External additions/deletions must not change
	// cached candidates.
	storeFixture(t, active, "missing/other.md",
		utils.Dedent(`
		# Added
		:::shemiq
		type: top-level
		:::
		`))
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	tasks, err = store.TopLevelTasks()
	if err != nil || len(tasks) != 1 ||
		tasks[0].Path != b || tasks[0].Title != "Updated" {
		t.Fatalf("cached candidates: tasks=%+v err=%v",
			tasks, err)
	}
}

func TestStoreExplicitScopes(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "documents")
	a := storeFixture(t, selected, "a.MARKDOWN",
		utils.Dedent(`
		# A
		:::shemiq
		type: top-level
		:::
		`))
	storeFixture(t, selected, "nested/b.MD",
		utils.Dedent(`
		# B
		:::shemiq
		type: top-level
		:::
		`))
	child := storeFixture(t, selected, "nested/child.md",
		utils.Dedent(`
		# Child
		:::shemiq
		type: task
		:::
		`))
	storeFixture(t, selected, "ignored.txt",
		utils.Dedent(`
		:::shemiq
		malformed
		`))
	outside := storeFixture(t, root, "outside.md",
		utils.Dedent(`
		# Outside
		:::shemiq
		type: top-level
		:::
		`))
	store, err := NewStore(root, "documents")
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := store.TopLevelTasks()
	if err != nil || len(tasks) != 2 ||
		tasks[0].Path != a || tasks[1].Title != "B" {
		t.Fatalf(
			"explicit directory: tasks=%+v err=%v",
			tasks, err)
	}
	if _, err := store.TaskByPath(outside); err == nil {
		t.Fatal("directory scope allowed an outside query")
	}
	if err := os.Symlink(outside,
		filepath.Join(selected, "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TaskByPath(
		"documents/escape.md"); err == nil {
		t.Fatal("symlink escaped directory scope")
	}
	fileStore, err := NewStore(root, "documents/a.MARKDOWN")
	if err != nil {
		t.Fatal(err)
	}
	tasks, err = fileStore.TopLevelTasks()
	if err != nil || len(tasks) != 1 ||
		tasks[0].ProjectRoot != "" {
		t.Fatalf(
			"generic explicit file: tasks=%+v err=%v",
			tasks, err)
	}
	if _, err := fileStore.TaskByPath(child); err == nil {
		t.Fatal("file scope allowed another document")
	}
	childStore, err := NewStore(root, child)
	if err != nil {
		t.Fatal(err)
	}
	if tasks, err := childStore.TopLevelTasks(); err != nil ||
		len(tasks) != 0 {
		t.Fatalf(
			"child is not a top-level candidate: "+
				"tasks=%+v err=%v",
			tasks, err)
	}
}

// storeFixture writes a test file and returns its absolute
// path. Shared across store, task, create, and validate tests.
func storeFixture(
	t *testing.T, root, relative, data string,
) string {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(
		filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

