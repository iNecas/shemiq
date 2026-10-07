package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/utils"
)

func TestSectionTaskStatusConversion(t *testing.T) {
	for _, tc := range []struct {
		name, fields, status, message string
		line                          int
	}{
		{"omitted", "", "new", "missing uuid", 2},
		{"invalid", "status: refiend\n", "", "invalid status", 4},
		{"empty", "status: \n", "", "empty metadata field: status", 4},
		{"repeated", "status: new\nstatus: done\n", "", "repeated metadata field: status", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.name + ".md"
			data := []byte("# Title\n:::shemiq\ntype: top-level\n" + tc.fields + ":::\n")
			doc, err := parseMarkdownDocument(path, data)
			if err != nil {
				t.Fatal(err)
			}
			got := sectionTask(doc, doc.root.children[0], false)
			if got.Status != tc.status || got.Title != "Title" {
				t.Fatalf("conversion: task=%+v", got)
			}
			found := false
			for _, issue := range got.Issues {
				if issue.Path == path && issue.Line == tc.line &&
					strings.Contains(issue.Message, tc.message) {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing located issue %q: %+v", tc.message, got.Issues)
			}
		})
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
		filepath.Join(
			filepath.Dir(parent), "missing.md") {
		t.Fatalf("uncreated target lost: %+v",
			top.Subtasks[1])
	}
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

func TestDirectiveFieldsMetadataOnly(t *testing.T) {
	for _, status := range []string{"new", "invalid"} {
		t.Run(status, func(t *testing.T) {
			path := "metadata.md"
			// Parse a valid document with a primary heading
			// and test directiveFields on the directive.
			doc, err := parseMarkdownDocument(path,
				[]byte("# Title\n:::shemiq\nstatus: "+
					status+"\n:::\n"))
			if err != nil {
				t.Fatal(err)
			}
			directive := doc.root.children[0].
				directives[0]
			fields, issues := directiveFields(
				path, directive)
			if status == "invalid" {
				if fields["status"].value != "" ||
					len(issues) != 2 ||
					issues[0].Message !=
						"invalid status: invalid" ||
					issues[1].Message !=
						"missing uuid" {
					t.Fatalf(
						"metadata-only: fields=%+v "+
							"issues=%+v",
						fields, issues)
				}
			} else if fields["status"].value != "new" ||
				len(issues) != 1 ||
				issues[0].Message != "missing uuid" {
				t.Fatalf(
					"general metadata findings: %+v",
					issues)
			}
		})
	}
}
