package console

import (
	"strings"
	"testing"

	"github.com/iNecas/shemiq/internal/task"
	"github.com/iNecas/shemiq/internal/utils"
)

func TestRenderTreeEmpty(t *testing.T) {
	result := RenderTree(task.TreeDocument{Tasks: []task.TreeNode{}}, false)
	if result != "No tasks.\n" {
		t.Fatalf("expected 'No tasks.\\n', got %q", result)
	}
}

func TestRenderTreeNested(t *testing.T) {
	doc := task.TreeDocument{
		Tasks: []task.TreeNode{
			{
				Title:  "Root",
				Status: "refined",
				Subtasks: []task.TreeNode{
					{
						Title:  "Child A",
						Status: "done",
						Subtasks: []task.TreeNode{
							{Title: "Grandchild", Status: "new", Subtasks: []task.TreeNode{}},
						},
					},
					{Title: "Child B", Status: "new", Subtasks: []task.TreeNode{}},
				},
			},
		},
	}

	result := RenderTree(doc, false)
	expected := utils.Dedent(`
		Root [refined]
		├── Child A [done]
		│   └── Grandchild [new]
		└── Child B [new]
		`)
	if result != expected {
		t.Fatalf("mismatch:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestRenderTreeWithColor(t *testing.T) {
	doc := task.TreeDocument{
		Tasks: []task.TreeNode{
			{Title: "Task", Status: "new", Subtasks: []task.TreeNode{}},
		},
	}
	result := RenderTree(doc, true)
	if !strings.Contains(result, ansiCyan+"new"+ansiReset) {
		t.Fatalf("expected colored 'new' in %q", result)
	}
	// Title and brackets should not contain ANSI escapes.
	before := strings.Index(result, "\033")
	if !strings.HasPrefix(result, "Task [") || before != len("Task [") {
		t.Fatalf("ANSI should only appear in status: %q", result)
	}
}
