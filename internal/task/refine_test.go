package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTopLevelTaskIncludesSubtasks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".shemiq", "tasks", "example")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "top-level.md")
	data := "# Example\n:::shemiq\ntype: top-level\nstatus: refined\n:::\n" +
		"\n## Tasks\n\n### First\n:::shemiq\ntype: task\n:::\n" +
		"\n### Second\n:::shemiq\ntype: task\nstatus: done\n:::\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	top, err := ReadTopLevelTask(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	if top.Type != TypeTopLevel || top.Path != path || len(top.Subtasks) != 2 {
		t.Fatalf("unexpected top-level type, path, or number of subtasks")
	}
	for i, child := range top.Subtasks {
		if child.Type != TypeTask || child.Path != "" || child.Title != []string{"First", "Second"}[i] || child.Status != []string{"new", "done"}[i] {
			t.Fatalf("unexpected subtask %q type, path, title, or status", child.Title)
		}
	}
}
