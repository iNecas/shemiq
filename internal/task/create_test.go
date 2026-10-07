package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestStoreCreateTopLevel(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, ".shemiq", "tasks")
	storeFixture(t, active, "existing/top-level.md",
		utils.Dedent(`
		# Existing
		:::shemiq
		type: top-level
		uuid: 12345678-1234-4234-8234-123456789abc
		:::
		`))

	// Store with explicit scope different from cwd's project.
	other := t.TempDir()
	scope := filepath.Join(other, "docs")
	if err := os.MkdirAll(scope, 0755); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root, scope)
	if err != nil {
		t.Fatal(err)
	}

	// Creation discovers from cwd, not read scope.
	path, err := store.CreateTopLevel(
		"New Task", "A description")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	want := filepath.Join(
		active, "new-task", "top-level.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("created file missing: %v", err)
	}

	// Store not loaded yet: no cache insertion.
	if len(store.tasks) != 0 {
		t.Fatal("unloaded store should not cache")
	}

	// Now test in-scope cache insertion after loading.
	inScope, err := NewStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	// Force load.
	if _, err := inScope.TopLevelTasks(); err != nil {
		t.Fatal(err)
	}
	countBefore := len(inScope.tasks)
	path2, err := inScope.CreateTopLevel(
		"Another", "desc")
	if err != nil {
		t.Fatalf("in-scope create: %v", err)
	}
	if len(inScope.tasks) != countBefore+1 {
		t.Fatalf(
			"expected cache insertion: before=%d after=%d",
			countBefore, len(inScope.tasks))
	}
	got, err := inScope.TaskByPath(path2)
	if err != nil || got.Type != TypeTopLevel {
		t.Fatalf(
			"cached task query: task=%+v err=%v",
			got, err)
	}
}
