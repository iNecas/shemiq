package console

import (
	"strings"

	"github.com/iNecas/shemiq/internal/task"
)

// RenderTree returns a human-readable tree of titles and statuses.
// When color is true, status words are wrapped in ANSI escapes.
// An empty document returns "No tasks.\n".
func RenderTree(document task.TreeDocument, color bool) string {
	if len(document.Tasks) == 0 {
		return "No tasks.\n"
	}
	var b strings.Builder
	for _, root := range document.Tasks {
		renderNode(&b, root, "", true, true, color)
	}
	return b.String()
}

func renderNode(b *strings.Builder, node task.TreeNode, prefix string, isRoot bool, isLast bool, color bool) {
	if isRoot {
		b.WriteString(nodeLabel(node, color))
		b.WriteByte('\n')
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		b.WriteString(prefix)
		b.WriteString(connector)
		b.WriteString(nodeLabel(node, color))
		b.WriteByte('\n')
	}

	var childPrefix string
	if isRoot {
		childPrefix = prefix
	} else if isLast {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}

	for i, child := range node.Subtasks {
		last := i == len(node.Subtasks)-1
		renderNode(b, child, childPrefix, false, last, color)
	}
}

func nodeLabel(node task.TreeNode, color bool) string {
	return node.Title + " [" + ColorStatus(string(node.Status), color) + "]"
}
