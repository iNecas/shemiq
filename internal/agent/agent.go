// Package agent launches interactive refinement sessions without owning task discovery.
package agent

import "io"

// Launcher starts a fresh refinement session and waits for it to finish.
// Session completion alone says nothing about the task's refinement status.
type Launcher interface {
	Launch(Request, Streams) error
}

// Request is the provider-neutral handoff from task selection. Paths are absolute.
type Request struct {
	Flow         Flow
	DocumentPath string
	ProjectRoot  string
	SubtaskTitle string
}

type Flow string

const (
	RefineTopLevel Flow = "refine-top-level"
	RefineSubtask  Flow = "refine-subtask"
)

// Streams must retain the original terminal files, not buffered readers or pipes.
type Streams struct {
	Input       io.Reader
	Output      io.Writer
	Diagnostics io.Writer
}
