package task

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

// Validate reports structural and hierarchy findings. Headingless documents
// and missing types are already covered by the command validation tests.
func TestStoreValidateHierarchyAndStructure(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		files         map[string]string
		count, line   int
	}{
		{
			name: "missing_directive", message: "missing heading-adjacent task directive", count: 1, line: 1,
			files: map[string]string{"top-level.md": utils.Dedent(`
				# First
				Prose prevents adjacency.
				`)},
		},
		{
			name: "empty_title", message: "empty task title", count: 1, line: 1,
			files: map[string]string{"top-level.md": utils.Dedent(`
				#
				:::shemiq
				type: task
				uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
				:::
				`)},
		},
		{
			name: "source_on_primary", message: "source is not permitted", count: 1, line: 4,
			files: map[string]string{"top-level.md": utils.Dedent(`
				# Title
				:::shemiq
				type: task
				source: ./future.md
				:::
				`)},
		},
		{
			name: "parent_on_entry", message: "parent is not permitted", count: 1, line: 11,
			files: map[string]string{"top-level.md": utils.Dedent(`
				# Top
				:::shemiq
				type: top-level
				uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
				:::
				## Tasks
				### Entry
				:::shemiq
				type: task
				uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
				parent: ./top-level.md
				:::
				`)},
		},
		{
			// No roots; C -> B must not become a spurious cycle after A <-> B.
			name: "cycle", message: "child-reference cycle", count: 1, line: 11,
			files: map[string]string{
				"a/top-level.md": utils.Dedent(`
					# A
					:::shemiq
					type: top-level
					parent: ../b/top-level.md
					uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
					:::
					## Tasks
					### B
					:::shemiq
					type: task
					source: ../b/top-level.md
					:::
					### C
					:::shemiq
					type: task
					source: ../c/top-level.md
					:::
					`),
				"b/top-level.md": utils.Dedent(`
					# B
					:::shemiq
					type: task
					parent: ../a/top-level.md
					uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
					:::
					## Tasks
					### A
					:::shemiq
					type: task
					source: ../a/top-level.md
					:::
					`),
				"c/top-level.md": utils.Dedent(`
					# C
					:::shemiq
					type: task
					parent: ../a/top-level.md
					uuid: cccccccc-cccc-4ccc-8ccc-cccccccccccc
					:::
					## Tasks
					### B
					:::shemiq
					type: task
					source: ../b/top-level.md
					:::
					`),
			},
		},
		{
			name: "multiple_owners", message: "multiple owning parents", count: 2, line: 10,
			files: map[string]string{
				"p1/top-level.md": utils.Dedent(`
					# Parent 1
					:::shemiq
					type: top-level
					uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
					:::
					## Tasks
					### Child
					:::shemiq
					type: task
					source: ../child.md
					:::
					`),
				"p2/top-level.md": utils.Dedent(`
					# Parent 2
					:::shemiq
					type: top-level
					uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
					:::
					## Tasks
					### Child
					:::shemiq
					type: task
					source: ../child.md
					:::
					`),
				"child.md": utils.Dedent(`
					# Child
					:::shemiq
					type: task
					parent: ./p1/top-level.md
					uuid: cccccccc-cccc-4ccc-8ccc-cccccccccccc
					:::
					`),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for path, content := range tc.files {
				storeFixture(t, root, path, content)
			}
			store, err := NewStore(root, root)
			if err != nil {
				t.Fatal(err)
			}
			issues, err := store.Validate(false)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			seen := make(map[Issue]bool)
			for _, issue := range issues {
				if seen[issue] {
					t.Fatalf("duplicate diagnostic: %+v", issue)
				}
				seen[issue] = true
				if strings.Contains(issue.Message, tc.message) {
					count++
					if issue.Line != tc.line || issue.Fixed {
						t.Fatalf("incorrect finding location or repair: %+v", issue)
					}
				}
			}
			if count != tc.count {
				t.Fatalf("expected %d %q findings, got %+v", tc.count, tc.message, issues)
			}
			again, err := store.Validate(false)
			if err != nil || !reflect.DeepEqual(issues, again) {
				t.Fatalf("non-deterministic validation: %+v, %v", again, err)
			}
		})
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
