package task

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

// TestStoreTreeMergedTree covers the representative success
// scenario: authoritative child metadata, missing-source and
// inline children, external references via parent, ordering,
// root deduplication, and child-directory selection recovering
// ancestors.
func TestStoreTreeMergedTree(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, ".shemiq", "tasks")

	// Parent top-level with three subtask entries.
	storeFixture(t, active, "example/top-level.md",
		utils.Dedent(`
		# Example
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
		status: refined
		:::

		## Tasks

		### Created child
		:::shemiq
		type: task
		source: ./child.md
		status: refined
		:::

		### Not yet created
		:::shemiq
		type: task
		source: ./missing.md
		status: new
		:::

		### Inline entry
		Just prose, no directive.
		`))

	// Child document with its own subtask.
	storeFixture(t, active, "example/child.md",
		utils.Dedent(`
		# Authoritative title
		:::shemiq
		type: task
		parent: ./top-level.md
		uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
		status: refined
		:::

		## Tasks

		### Grandchild
		:::shemiq
		type: task
		source: ./grandchild.md
		status: refined
		:::
		`))

	// Grandchild document.
	storeFixture(t, active, "example/grandchild.md",
		utils.Dedent(`
		# Grandchild doc
		:::shemiq
		type: task
		parent: ./child.md
		uuid: cccccccc-cccc-4ccc-8ccc-cccccccccccc
		status: refined
		:::
		`))

	// Scope from child directory: should recover parent
	// through parent: reference.
	store, err := NewStore(root,
		filepath.Join(active, "example"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := store.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	// Verify JSON serialization.
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip TreeDocument
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc, roundtrip) {
		t.Fatalf("JSON roundtrip changed data:\noriginal: %+v\nroundtrip: %+v",
			doc, roundtrip)
	}

	if len(doc.Tasks) != 1 {
		t.Fatalf("expected 1 root, got %d", len(doc.Tasks))
	}

	top := doc.Tasks[0]
	if top.Title != "Example" ||
		top.Type != TreeTaskTypeTopLevel ||
		top.Status != TreeStatusRefined ||
		top.UUID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" ||
		top.ProjectRoot != root {
		t.Fatalf("unexpected root: %+v", top)
	}
	if len(top.Subtasks) != 3 {
		t.Fatalf("expected 3 subtasks, got %d",
			len(top.Subtasks))
	}

	// First subtask: uses authoritative child metadata.
	child := top.Subtasks[0]
	if child.Title != "Authoritative title" ||
		child.UUID != "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" ||
		child.Status != TreeStatusRefined {
		t.Fatalf("child not authoritative: %+v", child)
	}
	if len(child.Subtasks) != 1 ||
		child.Subtasks[0].Title != "Grandchild doc" ||
		child.Subtasks[0].UUID != "cccccccc-cccc-4ccc-8ccc-cccccccccccc" {
		t.Fatalf("grandchild missing: %+v", child.Subtasks)
	}

	// Second subtask: missing source, entry metadata retained.
	missing := top.Subtasks[1]
	if missing.Title != "Not yet created" ||
		missing.Status != TreeStatusNew ||
		missing.UUID != "" {
		t.Fatalf("missing-source entry: %+v", missing)
	}
	if missing.Path == "" {
		t.Fatal("missing-source should have a target path")
	}

	// Third subtask: inline, no path.
	inline := top.Subtasks[2]
	if inline.Title != "Inline entry" ||
		inline.Path != "" {
		t.Fatalf("inline entry: %+v", inline)
	}

	// Verify empty arrays not null in JSON.
	if !strings.Contains(string(data), `"subtasks":[]`) {
		t.Fatalf("leaf subtasks should be []: %s", data)
	}

	// Verify input files unchanged.
	for _, rel := range []string{
		"example/top-level.md",
		"example/child.md",
		"example/grandchild.md",
	} {
		path := filepath.Join(active, rel)
		info, _ := os.Stat(path)
		if info == nil {
			t.Fatalf("file removed: %s", path)
		}
	}
}

// TestStoreTreeEmptyDirectory checks that a valid empty
// directory returns { "tasks": [] }.
func TestStoreTreeEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "empty")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := store.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"tasks":[]}` {
		t.Fatalf("empty dir: %s", data)
	}
}

// TestStoreTreeRejectsFileScope ensures file scopes fail.
func TestStoreTreeRejectsFileScope(t *testing.T) {
	root := t.TempDir()
	f := storeFixture(t, root, "single.md",
		utils.Dedent(`
		# Single
		:::shemiq
		type: top-level
		uuid: dddddddd-dddd-4ddd-8ddd-dddddddddddd
		:::
		`))
	store, err := NewStore(root, f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Tree(); err == nil {
		t.Fatal("expected error for file scope")
	}
}

// Tree must run full validation over the reached closure before constructing
// any tree, return located findings, and leave the original files untouched.
func TestStoreTreeRunsFullValidation(t *testing.T) {
	root := t.TempDir()
	parent := storeFixture(t, root, "selected/top-level.md", utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
		:::
		## Tasks
		### Child
		:::shemiq
		type: task
		source: ../external/child.md
		:::
		`))
	child := storeFixture(t, root, "external/child.md", utils.Dedent(`
		# Child
		:::shemiq
		parent: ../selected/top-level.md
		:::
		`))
	before := make(map[string][]byte)
	for _, path := range []string{parent, child} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = data
	}
	store, err := NewStore(root, filepath.Dir(parent))
	if err != nil {
		t.Fatal(err)
	}
	issues, err := store.Validate(false)
	if err != nil || len(issues) == 0 {
		t.Fatalf("expected validation findings in the fixture: %+v, %v", issues, err)
	}
	doc, err := store.Tree()
	var ve *ValidationError
	if !errors.As(err, &ve) || !reflect.DeepEqual(issues, ve.Issues) {
		t.Fatalf("tree did not return the same validation findings: doc=%+v err=%v", doc, err)
	}
	if !reflect.DeepEqual(doc, TreeDocument{}) {
		t.Fatalf("validation failure returned a usable document: %+v", doc)
	}
	for _, issue := range ve.Issues {
		if issue.Path != child || issue.Line != 2 || issue.Fixed {
			t.Fatalf("expected read-only findings in the reached child: %+v", issue)
		}
	}
	for path, original := range before {
		after, err := os.ReadFile(path)
		if err != nil || string(after) != string(original) {
			t.Fatalf("tree changed %s: %v", path, err)
		}
	}
}

// TestStoreTreeChildScopeRecoversAncestor verifies that
// scoping to a child directory recovers the ancestor and
// its sibling branches through parent: references.
func TestStoreTreeChildScopeRecoversAncestor(t *testing.T) {
	root := t.TempDir()
	active := filepath.Join(root, ".shemiq", "tasks")

	storeFixture(t, active, "parent/top-level.md",
		utils.Dedent(`
		# Parent
		:::shemiq
		type: top-level
		uuid: aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
		status: new
		:::

		## Tasks

		### Child A
		:::shemiq
		type: task
		source: ../child-a/top-level.md
		status: new
		:::

		### Child B
		:::shemiq
		type: task
		source: ../child-b/top-level.md
		status: new
		:::
		`))

	storeFixture(t, active, "child-a/top-level.md",
		utils.Dedent(`
		# Child A
		:::shemiq
		type: task
		parent: ../parent/top-level.md
		uuid: bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
		status: new
		:::
		`))

	storeFixture(t, active, "child-b/top-level.md",
		utils.Dedent(`
		# Child B
		:::shemiq
		type: task
		parent: ../parent/top-level.md
		uuid: cccccccc-cccc-4ccc-8ccc-cccccccccccc
		status: new
		:::
		`))

	// Scope to child-a directory.
	store, err := NewStore(root,
		filepath.Join(active, "child-a"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := store.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	// Should have one root (parent), with two children.
	if len(doc.Tasks) != 1 {
		t.Fatalf("expected 1 root, got %d: %+v",
			len(doc.Tasks), doc.Tasks)
	}
	if doc.Tasks[0].Title != "Parent" {
		t.Fatalf("expected Parent root, got %s",
			doc.Tasks[0].Title)
	}
	if len(doc.Tasks[0].Subtasks) != 2 {
		t.Fatalf("expected 2 subtasks, got %d",
			len(doc.Tasks[0].Subtasks))
	}
	if doc.Tasks[0].Subtasks[0].Title != "Child A" ||
		doc.Tasks[0].Subtasks[1].Title != "Child B" {
		t.Fatalf("unexpected subtask order: %+v",
			doc.Tasks[0].Subtasks)
	}
}
