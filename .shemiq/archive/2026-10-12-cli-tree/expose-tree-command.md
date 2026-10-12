# Expose the tree command
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 73d3f115-f900-45bc-97a4-436e1e20c1a4
:::

## Context

- Add `shemiq tree [dir]`, with human-readable output by default and `--json` selecting the shared machine-readable contract.
- The completed store-export and naming tasks provide `Store.Tree() (TreeDocument, error)` and the `Tree*` DTOs in `internal/task/tree.go`; do not rebuild relationships or inspect parser internals in the command.
- Both modes consume the same fully validated, merged model and preserve its scope and ordering.
- `cmd/root.go` separates the result writer from diagnostics; reuse that boundary and the existing command test harness.
- Human rendering and color utilities belong in `internal/console`; JSON serialization and diagnostics belong in `cmd`.
- `golang.org/x/term` is already a dependency; no new dependency is needed for terminal detection.
- TypeScript generation remains a separate sibling task. Do not change the DTO or JSON contract for presentation needs.

## Interview

- Choose unprefixed roots, Unicode child connectors, and `Title [status]` labels; no synthetic root or blank separators.
- Choose two-space JSON indentation and a trailing newline.
- Print validation findings once as `path:line: message`; omit usage text for runtime failures while preserving normal Cobra help for argument/flag mistakes.
- Supersede the parent's original no-color rule: automatically color status words on terminal result streams, with no color flag.
- Use cyan for `new`, yellow for `refined`, and green for `done`. Keep brackets, titles, and connectors plain; no bold or background colors.
- Disable color for pipes/files, when `NO_COLOR` is set, or when `TERM=dumb`; JSON never contains color escapes.
- Choose human rendering and reusable color utilities in `internal/console`, with JSON serialization in the command. Consume the existing task DTO directly rather than adding a presentation model.
- Approve `ColorEnabled(output io.Writer) bool` and `RenderTree(document task.TreeDocument, color bool) string` as console entry points.
- Approve minimal command/console tests, usage documentation in `README.md` and `skills/shemiq/SKILL.md`, and unchanged TypeScript-generation scope.

## Implementation plan

### Command and scope

- Add `cmd/tree.go` and register the command in `cmd/root.go`.
- Accept at most one directory argument and a boolean `--json` flag. Add no aliases, additional formats, color flags, filtering, or interactive controls.
- Construct `task.NewStore(cwd, dir)` and call `Store.Tree()` once; keep command orchestration thin.
- With no argument, use the existing nearest-project discovery of active `.shemiq/tasks`. An explicit directory supplies the starting scope.
- Preserve source/parent closure behavior, including reached ancestors, sibling branches, and external documents. Preserve authoritative child metadata, root ordering, sibling ordering, and root deduplication.
- Do not prompt, launch Pi, create task files, insert UUIDs, repair statuses, or otherwise modify task files.

### Console placement and API

- Add `internal/console/tree.go` for root lines, recursive connectors, indentation, labels, status-color selection, and empty-tree text.
- Add `internal/console/color.go` for result-stream terminal detection, environment checks, and small reusable ANSI styling helpers.
- Expose `ColorEnabled(output io.Writer) bool`; return false for non-terminal or unrecognized writers, when `NO_COLOR` is set, or when `TERM=dumb`. Probe the actual result stream, not diagnostics or global stdout.
- Use the existing `golang.org/x/term` dependency for terminal detection.
- Expose `RenderTree(document task.TreeDocument, color bool) string`; return the complete human-readable output without writing it.
- The command passes `ColorEnabled(result)` to the human renderer. Explicit color selection keeps rendering deterministic and testable without a terminal.
- Consume the existing `task.TreeDocument` directly; task/store code remains independent of console rendering.
- Leave the existing interactive `console.Console` API unchanged. Do not introduce a generic rendering framework, theme system, or color-policy abstraction.

### Human and JSON output

- Human output uses `Title [status]` on each line, unprefixed roots, and Unicode child connectors (`├──`, `└──`, `│`) with indentation preserving parent-child relationships.
- Use no synthetic root or blank separators. End output with a newline.
- Show no paths, UUIDs, or additional metadata in human output. An empty tree prints `No tasks.` followed by a newline.
- Color only the status word: `new` cyan, `refined` yellow, `done` green. Reset ANSI styling immediately afterward, leaving brackets and subsequent output plain. Use no bold or background colors.
- Keep JSON serialization in `cmd/tree.go`, using the standard library and the existing `TreeDocument` contract.
- Use two-space JSON indentation and a trailing newline. Preserve existing JSON keys, field types, optional fields, and empty task/subtask arrays; an empty tree has an empty `tasks` array.
- JSON bypasses console/color helpers and never contains ANSI color escapes.

### Output boundary and diagnostics

- Finish loading, validation, and rendering/JSON serialization in memory before writing to the dedicated result stream.
- Preparation failures return an error/nonzero exit and leave stdout empty. Propagate output-writer failures; already written bytes cannot be rolled back.
- Use `task.ValidationError` for located findings and reuse the existing validation-command pattern to print each `path:line: message` once on stderr, without an additional duplicate error summary.
- Other runtime failures show their error without usage text. Argument/flag mistakes retain normal Cobra help.
- Apply this diagnostic policy only to the tree command; leave other commands unchanged.

### Minimal verification and documentation

- Add `cmd/tree_test.go` using the existing command harness.
- Reuse one representative fixture across both output modes, covering authoritative child titles, a missing-source child, an external descendant, nesting, deterministic ordering, and absence of duplicate roots.
- Assert exact plain human layout and final newline; check the shared JSON contract and two-space formatting. Successful commands leave stderr empty.
- Keep compact command failure coverage for validation errors, file-argument rejection, and missing directories: errors/nonzero exit, empty stdout, and concise stderr without runtime usage text.
- Keep individual validity-rule coverage in existing store/validation tests; do not duplicate parser or graph-validation tests.
- Check empty-directory success in both output modes.
- Add focused console tests for the three status colors and immediate resets using the renderer's explicit color boolean. Check automatic color disabling for buffered writers, `NO_COLOR`, and `TERM=dumb`; do not introduce pseudo-terminal infrastructure.
- Document default/explicit scope, reference expansion, human/JSON examples, automatic colors, and read-only failure behavior in `README.md` and `skills/shemiq/SKILL.md`.

### Sibling handoff

- Keep the TypeScript-generation sibling task separate and `new`; this task neither generates types nor changes its contract or requirements.
- The completed store-export and naming tasks remain complete. Their model is consumed unchanged by both output modes.

## Implementation notes

- Added `cmd/tree.go`: thin Cobra command registered in `cmd/root.go`, accepts `[dir]` and `--json` flag.
- Added `internal/console/color.go`: `ColorEnabled(output)` checks `Fd()` + `term.IsTerminal`, `NO_COLOR`, `TERM=dumb`; `ColorStatus()` wraps status words in ANSI escapes.
- Added `internal/console/tree.go`: `RenderTree(doc, color)` produces human-readable tree with Unicode connectors (`├──`, `└──`, `│`), `Title [status]` labels, empty output as `No tasks.\n`.
- JSON mode uses `json.MarshalIndent` with two-space indentation + trailing newline, bypassing color helpers.
- Validation errors printed as `path:line: message` on stderr; `SilenceUsage` set for runtime failures to suppress Cobra help.
- Tests in `cmd/tree_test.go`: human+JSON for representative fixture, empty directory both modes, validation error, file-argument rejection, missing directory.
- Tests in `internal/console/color_test.go`: three status colors with immediate resets, buffered writer, `NO_COLOR`, `TERM=dumb`.
- Tests in `internal/console/tree_test.go`: empty doc, nested tree layout, color in status only.
- Documented in `README.md` (step 5) and `skills/shemiq/SKILL.md` (new section before archiving).
