# Launch Pi from `shemiq refine`
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 6ab8727c-7a7d-4b45-bebc-6eb2e419c0b8
:::

## Context

The parent task defines `shemiq refine [path] [--subtask "Title"]`. The status lifecycle and task-selection work are complete. `cmd/refine.go` already resolves the selected refinement into a flow, absolute top-level document path, project root, and optional exact subtask title. It currently prints a `TODO: launch pi ...` preview. Replace that preview with interactive Pi invocation; preserve existing discovery, routing, eligibility checks, and numbered pickers.

`cmd.Execute` accepts input, result, and diagnostic streams. Cobra's output stream is deliberately the diagnostic stream, so Pi's stdout must use the result stream, not `cmd.OutOrStdout()`. The picker in `internal/console` owns a buffered reader shared by consecutive selections. Pi must receive the original input stream, not that buffered reader or a copying pipe.

The installed Shemiq Pi package supplies `prompts/shemiq-refine-and-split.md` and `prompts/shemiq-refine-sub-task.md`. Those prompts already record approved refinement and synchronize parent/subtask statuses. Installing and enabling the templates is a prerequisite; this task must not bundle prompt copies, inspect Pi settings to find them, or install packages automatically.

Pi uses an initial positional message to start refinement. With terminal stdin and stdout it opens the interactive UI; redirecting either stream automatically selects print mode. The CLI must prevent that unintended noninteractive execution.

## Interview

- Require terminal stdin and stdout before launching Pi. Reject redirection rather than attaching to the controlling terminal or allowing Pi's automatic print mode.
- Use a small provider-neutral launcher boundary with Pi as its only implementation. Launch a child process without a shell, inherit streams and environment, and wait for it to exit.
- Start a fresh, normally persisted Pi session. Let Pi configuration control model, authentication, and project trust. Do not add resume, model, provider-selection, or argument-passthrough options.
- Preserve normal CLI error behavior: missing executable, terminal requirements, and process failures return errors; the CLI exits with its existing error code `1`, not the child's exact exit code.
- Use focused CLI and fake-process coverage, not live Pi, network, or model calls. Update README and the Shemiq skill while keeping direct slash-command usage available.

## Implementation plan

### Launcher boundary and command integration

- Introduce a small launcher interface in `internal/agent`, conceptually `Launch(request Request, streams Streams) error`. The request carries a provider-neutral flow (top-level refinement or subtask refinement), absolute document path, project root, and optional exact subtask title. Streams carry standard input, output, and diagnostics using the existing reader/writer conventions. The Pi adapter owns the translation to slash-command names.
- Reuse the existing resolved selection handoff, adapting or relocating its types only as needed. Keep project discovery and document parsing in `internal/task` and command orchestration in `cmd/refine.go`. Do not redesign task selection.
- Wire the production Pi launcher into the command and provide a narrow injection seam for command tests. Preserve the public `cmd.Execute` stream-based API. Selection errors and cancellation must return before invoking the launcher.
- Remove the TODO preview. Do not print a success result or mark refinement complete just because the child exits successfully. Neither command selection nor launch success/failure may edit task documents.

### Pi process contract

- Before starting Pi, verify that the supplied stdin and stdout are actual terminals. A file type assertion alone is insufficient: redirected files and pipes are not terminals. Use a small portable terminal check; do not introduce controlling-terminal attachment or a PTY framework. Return a clear error explaining that refinement requires terminal stdin and stdout.
- Resolve `pi` from `PATH` and invoke it directly using Go's process facilities, never through a shell. Set the child working directory to the request's project root, inherit the environment, connect the original input/result/diagnostic streams directly, and wait for process completion. In normal CLI use these must remain terminal file descriptors rather than Go-created copying pipes.
- Pass exactly one initial slash-command message: `/shemiq-refine-and-split <path>` for top-level refinement or `/shemiq-refine-sub-task <path> <title>` for subtask refinement. Make the document path relative to the project root; keep the absolute path in the request. Do not pass print, resume, session, model, or trust-override flags.
- Encode path and title for Pi's argument parser, independently of operating-system argv handling. Preserve spaces, both quote types, and backslashes. The current `strconv.Quote` implementation in `resolvedRefinement.prompt()` is unsuitable: Pi's installed parser treats quote delimiters specially but does not interpret backslash escapes, so Go-style escaping corrupts embedded quotes and backslashes. Move prompt construction into the Pi adapter and replace this encoding.
- A read-only experiment against the installed Pi parser confirmed that single-quoting a value and encoding an embedded single quote as adjacent single/double-quoted segments (`'"'"'`) preserves a mixed-quote title and literal backslashes. Use parser-compatible encoding rather than assuming full shell escaping semantics; no shell is executed.
- Report an absent `pi` with an actionable installation/PATH error. Return contextual process-start and nonzero-exit errors through the existing CLI diagnostics, retaining underlying errors where useful. Leave terminal interaction and project-trust decisions to Pi.

### Minimal coverage

- Keep existing selection/rejection coverage in `cmd/refine_test.go`. Replace preview-output assertions with a recording launcher's request/stream assertions while preserving picker diagnostics. Verify rejected or cancelled selections do not launch anything.
- Add one focused CLI-to-process scenario with a fake Pi executable or helper process. Use a space-containing task path and a subtask title containing quotes and backslashes. Assert the project-root working directory, a single initial-message argument for the correct template, and connected input/output/diagnostic streams. Bypass only the terminal probe for this scenario, not process invocation; keep that seam internal and narrow.
- Cover redirected stdin and redirected stdout preventing startup, a missing executable, and a nonzero child exit. Include a small regression for parser-compatible argument quoting. Validate compatibility with Pi's parser through a read-only experiment if needed; automated tests must not depend on an installed Pi or its source tree.
- Confirm the CLI does not mutate task content on successful or failed launch. Avoid an exhaustive parser matrix, a new PTY test framework, and live model calls. Run `make check` after implementation.

### User guidance and scope

- Update `README.md` and `skills/shemiq/SKILL.md` to describe installing/building the CLI, Pi on `PATH`, and the installed/enabled Shemiq Pi templates prerequisite.
- Show `shemiq refine`, an explicit task directory, an explicit `top-level.md`, and `--subtask "Exact title"`. Explain that new top-level tasks enter top-level refinement, refined top-level tasks select a new subtask, and stdin/stdout must be terminals. Approved prompt-driven refinement—not process exit—updates status.
- Keep direct slash-command examples as an alternative. Do not modify refinement prompts, add provider/session features, or reopen completed sibling tasks. This work consumes the selection task's existing handoff and replaces its planned TODO preview.

## Implementation notes

- Added `internal/agent.Launcher`, provider-neutral `Request`/`Flow`, and original-stream `Streams`. The Pi adapter owns slash-command translation. `cmd.Execute` retains its public stream-based API; an unexported orchestration helper injects launchers for command tests. Selection, discovery, eligibility checks, and pickers are unchanged; their handoff is now `agent.Request` instead of `resolvedRefinement`.
- Replaced the TODO preview with a direct, shell-free Pi child process. The adapter requires actual terminal stdin/stdout using `golang.org/x/term`, resolves `pi` through PATH (pinning relative executable paths before changing directories), launches at the selected project root with inherited environment and original input/result/diagnostic streams, and waits. It returns actionable missing-executable and contextual wrapped start/exit errors; the existing CLI exit code remains `1`. No session/model/trust flags, package installation, task edits, or success/status output were added.
- Initial messages use project-relative document paths and parser-compatible single-quoted segments, encoding apostrophes as adjacent `'"'"'` segments rather than Go-style escapes. A read-only experiment with the installed Pi `parseCommandArgs` confirmed exact round-trip preservation of spaces, both quote types, and literal backslashes in paths and titles. Automated tests do not depend on installed Pi.
- Updated selection tests to record requests and assert original-stream identity, including after consecutive pickers; rejected/cancelled selections never launch. A copied Go test executable acts as fake Pi for a CLI-to-process scenario with a space-containing path and mixed-quote/backslash title, checking one initial argument, working directory, environment, stdin/stdout/stderr, success and nonzero exit, and unchanged task content. Additional focused checks cover real terminal probes rejecting files/pipes/wrapped streams, production CLI redirection rejection, absent Pi, process-start failure, and quoting. Only the terminal probe is bypassed for process invocation tests; no live Pi, network, model calls, or PTY framework is used.
- Updated `README.md` and `skills/shemiq/SKILL.md` with CLI installation/building, Pi/PATH and enabled-template prerequisites, all refinement forms, eligibility/terminal rules, prompt-approved status updates, and direct slash-command alternatives. Refinement prompts and completed sibling implementations were not changed.
- Verification: `make check` passes (`go test ./...` and `go vet ./...`); Shemiq metadata validation passes for this task directory.
