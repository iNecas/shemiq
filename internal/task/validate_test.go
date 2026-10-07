package task

import (
	"path/filepath"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

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
