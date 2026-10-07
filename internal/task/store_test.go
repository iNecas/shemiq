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
		t.Fatalf("alias did not reuse snapshot: task=%+v err=%v",
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
	// External additions/deletions must not change cached candidates.
	storeFixture(t, active, "missing/other.md", utils.Dedent(`
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
		t.Fatalf("cached candidates: tasks=%+v err=%v", tasks, err)
	}
}

func TestStoreExplicitScopes(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "documents")
	a := storeFixture(t, selected, "a.MARKDOWN", utils.Dedent(`
		# A
		:::shemiq
		type: top-level
		:::
		`))
	storeFixture(t, selected, "nested/b.MD", utils.Dedent(`
		# B
		:::shemiq
		type: top-level
		:::
		`))
	child := storeFixture(t, selected, "nested/child.md", utils.Dedent(`
		# Child
		:::shemiq
		type: task
		:::
		`))
	storeFixture(t, selected, "ignored.txt", utils.Dedent(`
		:::shemiq
		malformed
		`))
	outside := storeFixture(t, root, "outside.md", utils.Dedent(`
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
	if err != nil || len(tasks) != 2 || tasks[0].Path != a || tasks[1].Title != "B" {
		t.Fatalf("explicit directory: tasks=%+v err=%v", tasks, err)
	}
	if _, err := store.TaskByPath(outside); err == nil {
		t.Fatal("directory scope allowed an outside query")
	}
	if err := os.Symlink(outside, filepath.Join(selected, "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TaskByPath("documents/escape.md"); err == nil {
		t.Fatal("symlink escaped directory scope")
	}
	fileStore, err := NewStore(root, "documents/a.MARKDOWN")
	if err != nil {
		t.Fatal(err)
	}
	tasks, err = fileStore.TopLevelTasks()
	if err != nil || len(tasks) != 1 || tasks[0].ProjectRoot != "" {
		t.Fatalf("generic explicit file: tasks=%+v err=%v", tasks, err)
	}
	if _, err := fileStore.TaskByPath(child); err == nil {
		t.Fatal("file scope allowed another document")
	}
	childStore, err := NewStore(root, child)
	if err != nil {
		t.Fatal(err)
	}
	if tasks, err := childStore.TopLevelTasks(); err != nil || len(tasks) != 0 {
		t.Fatalf("child is not a top-level candidate: tasks=%+v err=%v", tasks, err)
	}
}

func TestStoreEntryAndChildAreDistinct(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	child := storeFixture(t, other,
		".shemiq/archive/child.md",
		utils.Dedent(`
		# Child title
		:::shemiq
		type: task
		status: done
		:::
		`))
	parent := storeFixture(t, root,
		".shemiq/tasks/example/top-level.md",
		utils.Dedent(`
		# Parent

		:::shemiq
		type: top-level
		:::

		## Tasks

		### Entry title
		:::shemiq
		type: task
		source: ./child.md
		:::

		### Not created
		:::shemiq
		type: task
		source: ./missing.md
		status: refined
		:::

		### Ordinary section
		Just prose.
		`))
	if err := os.Symlink(child,
		filepath.Join(filepath.Dir(parent),
			"child.md")); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root, filepath.Dir(parent))
	if err != nil {
		t.Fatal(err)
	}
	top, err := store.TaskByPath(filepath.Dir(parent))
	// 3 subtasks: Entry title, Not created, Ordinary section
	// (directive-less subtask creates a valid new entry)
	if err != nil || top.Status != "new" ||
		top.Type != TypeTopLevel ||
		len(top.Subtasks) != 3 {
		t.Fatalf(
			"parent query: task=%+v err=%v", top, err)
	}
	entry := top.Subtasks[0]
	if top.document != store.tasks[parent].document ||
		entry.document != top.document ||
		top.Subtasks[1].document != top.document {
		t.Fatal(
			"parent and entries must reference " +
				"their defining document")
	}
	if entry.Title != "Entry title" ||
		entry.Status != "new" ||
		entry.Path != child ||
		entry.ProjectRoot != root {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if top.Subtasks[1].Path !=
		filepath.Join(filepath.Dir(parent), "missing.md") {
		t.Fatalf("uncreated target lost: %+v",
			top.Subtasks[1])
	}
	// Directive-less subtask: type task, status new, no path
	ordinary := top.Subtasks[2]
	if ordinary.Title != "Ordinary section" ||
		ordinary.Type != TypeTask ||
		ordinary.Status != "new" ||
		ordinary.Path != "" ||
		len(ordinary.Issues) != 0 {
		t.Fatalf("directive-less subtask: %+v", ordinary)
	}
	childStore, err := NewStore(root, child)
	if err != nil {
		t.Fatal(err)
	}
	standalone, err := childStore.TaskByPath(child)
	if err != nil ||
		standalone.Title != "Child title" ||
		standalone.Status != "done" ||
		standalone.ProjectRoot != other ||
		standalone.section == entry.section ||
		standalone.document !=
			childStore.tasks[child].document ||
		standalone.document == entry.document {
		t.Fatalf(
			"unexpected child: task=%+v err=%v",
			standalone, err)
	}
}

func TestStorePrimaryStructure(t *testing.T) {
	root := t.TempDir()
	// Non-adjacent directive and later heading with directive:
	// only primary heading's adjacent directive is accepted;
	// non-adjacent and later # heading directives are now
	// rejected by placement rules.
	path := storeFixture(t, root, "structure.md",
		utils.Dedent(`
		# First
		Prose prevents adjacency.
		`))
	store, err := NewStore(root, path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.TaskByPath(path)
	if err != nil || got.Title != "First" ||
		got.Status != "" || len(got.Issues) != 1 {
		t.Fatalf(
			"primary without directive: task=%+v err=%v",
			got, err)
	}
	if got.document != store.tasks[path].document {
		t.Fatal("task's document not retained")
	}
	// File without a heading: valid but no primary task
	path = storeFixture(t, root, "noheading.md",
		utils.Dedent(`
		Just prose, no metadata.
		`))
	store, err = NewStore(root, path)
	if err != nil {
		t.Fatal(err)
	}
	got, err = store.TaskByPath(path)
	if err != nil || got.Path != path ||
		len(got.Issues) != 1 ||
		got.document != store.tasks[path].document {
		t.Fatalf("headingless query: task=%+v err=%v",
			got, err)
	}
}

// Validation repairs refresh the store's snapshots so later
// task queries observe promoted status, while references
// loaded stay outside query scope.
func TestStoreValidateRefreshesWithoutExpandingScope(
	t *testing.T,
) {
	root := t.TempDir()
	active := filepath.Join(
		root, ".shemiq", "tasks", "example")
	storeFixture(t, active, "top-level.md",
		utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: 12345678-1234-4234-8234-123456789abc
		:::
		## Tasks
		### First
		:::shemiq
		type: task
		source: ./first.md
		status: done
		:::
		`))
	child := storeFixture(t, active, "first.md",
		utils.Dedent(`
		# First
		:::shemiq
		type: task
		parent: ./top-level.md
		status: new
		:::
		`))
	store, err := NewStore(root, active)
	if err != nil {
		t.Fatal(err)
	}
	issues, err := store.Validate(true)
	if err != nil {
		t.Fatalf("validate: issues=%+v err=%v",
			issues, err)
	}
	got, err := store.TaskByPath(child)
	if err != nil || got.Status != "done" {
		t.Fatalf(
			"query did not see refreshed status: "+
				"task=%+v err=%v", got, err)
	}
	// Out-of-scope reference remains unqueryable.
	outer := storeFixture(t, root, "outside.md",
		utils.Dedent(`
		# Outside
		:::shemiq
		type: top-level
		:::
		`))
	if _, err := store.loadTask(
		outer, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TaskByPath(outer); err == nil {
		t.Fatal("loaded reference widened query scope")
	}
}

func storeFixture(t *testing.T, root, relative, data string) string {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
