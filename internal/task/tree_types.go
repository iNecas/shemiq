package task

// This file's declarations, constants, and JSON tags are the source of truth for
// the generated TypeScript tree contract. Keep internal parser/validation types
// out of this file: root tygo.yaml includes only this file and generates named
// literal unions.

// TreeDocument is the top-level JSON tree container.
type TreeDocument struct {
	Tasks []TreeNode `json:"tasks"`
}

// TreeNode represents one task in the merged tree.
// Unlike the parser-oriented Task model, it exposes only integration-facing
// metadata, independent of parsing details and internal validation state.
type TreeNode struct {
	Type        TreeTaskType `json:"type"`
	Title       string       `json:"title"`
	Status      TreeStatus   `json:"status"`
	ProjectRoot string       `json:"projectRoot"`
	Path        string       `json:"path,omitempty"`
	UUID        string       `json:"uuid,omitempty"`
	Subtasks    []TreeNode   `json:"subtasks"`
}

// TreeTaskType is the exported task type for the JSON contract.
type TreeTaskType string

const (
	TreeTaskTypeTopLevel TreeTaskType = "top-level"
	TreeTaskTypeTask     TreeTaskType = "task"
)

// TreeStatus is the exported task status for the JSON contract.
type TreeStatus string

const (
	TreeStatusNew     TreeStatus = "new"
	TreeStatusRefined TreeStatus = "refined"
	TreeStatusDone    TreeStatus = "done"
)
