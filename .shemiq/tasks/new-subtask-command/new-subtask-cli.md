# Expose and document the new-subtask command
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 9e65730b-e8ab-4375-aff0-fd9fde37da4d
:::

## Context

- Expose the existing `Store.CreateSubtask(parentPath, title, description, after string) (string, error)` through the CLI. Empty `after` means append; success returns the edited parent's absolute path.
- `cmd/new.go` supplies description joining, `readTitle`, and output conventions; `cmd/root.go` registers commands. Keep document editing and insertion checks in the Store.
- `internal/task/create_subtask.go` implements insertion; `./store-subtask-insertion.md` records its approved contract. The Store accepts a Markdown file, not a directory.
- Creation inserts only a directive-less `### <title>` heading and description under `## Tasks`. Separate subtask documents are still created later during refinement.

## Interview

- Include minimal Store fixes needed to satisfy the approved contract, rather than shipping known exceptions or splitting them into separate work.
- The Store currently ignores `after` when `## Tasks` is missing or contains only `[TBD]`; a requested nonexistent sibling must instead be rejected.
- Resolve directory arguments before prompting; leave document parsing, metadata checks, and insertion validation to the Store.
- Use compact command-level integration tests through the existing harness, plus README and skill updates. Leave workflow prompts unchanged.

## Implementation plan

### Command and integration

- Add and register `shemiq new-subtask <parent-path> [--title "Title"] [--after "Sibling"] <description...>` alongside `new`.
- Require the parent argument and a nonblank description. Join description arguments with spaces, matching `shemiq new`.
- Reuse `readTitle` when `--title` is omitted. An explicitly blank title fails without falling back to prompting.
- Resolve relative paths from the invocation directory. A directory selects `top-level.md`; an explicit Markdown filename need not be named `top-level.md`. Resolve directory arguments before prompting, then pass a file path to the Store.
- Call `Store.CreateSubtask` without discovering or creating a project from the working directory, scanning unrelated documents, or invoking full validation.
- Print only the returned absolute parent path plus a newline to stdout. Send prompts and diagnostics to stderr; propagate command errors using existing conventions.

### Behavior and Store follow-up

- Require a parseable parent with primary `type: top-level` metadata inside a `.shemiq` project; leave these checks in the Store rather than duplicating them in the CLI.
- Append by default. With `--after`, match an exact, case-sensitive sibling title and insert after its complete section, including nested content.
- Reject duplicate titles, missing or ambiguous `--after` matches, and structurally disruptive Markdown. Rejected requests leave the existing document unchanged.
- Replace a sole `[TBD]` placeholder or create a missing `## Tasks` section when appending. A nonexistent `--after` sibling must fail in these layouts too.
- Preserve unrelated content, existing directives, and the parent's status, including `done`. Do not add metadata or create a separate subtask file.
- Make only minimal Store corrections exposed by integration coverage and required by the approved contract; keep the CLI thin.
- Record such corrections in a short follow-up note in `./store-subtask-insertion.md`, retaining the completed task.

### Testing

- Use `runCommand`/`Execute` with temporary projects; do not introduce a subprocess framework or duplicate CLI scenarios at the Store level.
- Combine related success checks to keep coverage small:
  - Prompted title, directory resolution, and replacement of the sole `[TBD]` placeholder.
  - Explicit title, joined description arguments, default append, and `--after` placement after a complete sibling section with nested content.
  - Creation of a missing `## Tasks` section.
- Verify stdout/stderr separation, preservation of existing directives and unrelated content, unchanged parent status including `done`, and no new directive or separate task file.
- Run `validate` on a representative valid result; validation is a test assertion, not part of command execution.
- Use a small table of representative failures: missing or blank input, invalid parent, duplicate title, missing or ambiguous `--after`, and disruptive Markdown. Assert unchanged parent bytes for rejected requests against an existing file.
- Include missing-sibling `--after` regressions for both placeholder and missing-section layouts.

### Documentation

- Update `README.md` with file and directory examples, optional title prompting, default append, and exact-title `--after` placement.
- Update `skills/shemiq/SKILL.md` to distinguish top-level creation from subtask insertion; remove the obsolete implication that subtasks can only be added manually.
- Explain that `new-subtask` writes only a heading and description; refinement still creates the separate task document later.
- Document absolute-path output and rejection of duplicate titles or structurally disruptive Markdown.
- Leave workflow prompts unchanged.

## Implementation notes

- Created `cmd/new_subtask.go` with thin CLI command: resolves directory to `top-level.md`, reuses `readTitle` for prompting, delegates to `Store.CreateSubtask`.
- Registered in `cmd/root.go` alongside existing commands.
- Fixed Store bug: `--after` with missing `## Tasks` section or `[TBD]` placeholder was silently ignored; now correctly rejects with "--after sibling not found".
- Review follow-up: CLI tests cover prompting, relative directory/custom filename resolution, flag and description forwarding, stdout/stderr separation, and error propagation.
- Detailed insertion checks live in `internal/task/create_subtask_test.go`: append, placeholder/missing-section handling, nested-section placement, byte/status/directive preservation, no extra files, and unchanged-file rejection regressions.
- Updated `README.md` (new step 3a) and `skills/shemiq/SKILL.md` (new "Adding a subtask" section).
