# New subtask command
:::shemiq
type: top-level
uuid: 6c3d15d4-6d09-4b6b-ac21-24afcc40e422
status: refined
:::

## Description

- Existing `shemiq new` creates a top-level task; add `shemiq new-subtask` to insert a subtask into an existing top-level document.
- Accept the parent Markdown file or its directory, with an optional title and insertion point:

```sh
shemiq new-subtask .shemiq/tasks/my-task/top-level.md [--title "Optional title"] [--after "Sibling Task"] <description...>
```

- Create only a `### <title>` heading and description under `## Tasks`, without a directive or separate subtask file.

## Context

- `cmd/new.go` provides the existing argument, title-prompt, and output conventions; `cmd/root.go` registers commands.
- `internal/task/create.go` owns existing creation behavior; document editing belongs in `internal/task.Store` rather than the CLI.
- `internal/task/parse.go` retains section source offsets for targeted edits; validation already uses targeted document edits and cache refresh.
- Directive-less `###` entries under `## Tasks` already default to `type: task` and `status: new`.
- Existing refinement creates the separate subtask document later; this command does not change that workflow.
- Existing subtask selection uses exact titles, making duplicate sibling titles ambiguous.
- Fresh top-level documents contain a sole `[TBD]` placeholder under `## Tasks`.

## Interview

- Creation: title and description only; no directive, UUID, link, or separate file.
- Title: optional `--title`; prompt when omitted, matching `shemiq new`.
- Placement: append by default; `--after` selects an exact, case-sensitive sibling title.
- Parent: explicit top-level Markdown file or directory resolving to `top-level.md`, inside a `.shemiq` project.
- Status: allow any parent status and leave it unchanged.
- Duplicates: reject an exact, case-sensitive match with an existing sibling title.
- Incomplete layout: replace a sole `[TBD]` placeholder or append a missing `## Tasks` section.
- Architecture: choose a Store operation with targeted edits over CLI-local editing or whole-document regeneration.
- Validation: check only what safe insertion requires, not full metadata or linked-document validity.
- Output: print the edited parent's absolute path to stdout; prompts and diagnostics go to stderr.
- Testing: minimal, primarily CLI end-to-end coverage, plus a focused Store cache-refresh test.
- Documentation: update README and skill usage; leave workflow prompts unchanged.

## Design

### Architecture and components

- Keep the CLI thin: handle arguments and prompting in a new command alongside `cmd/new.go`, registered in `cmd/root.go`.
- Put insertion, insertion-related checks, targeted file editing, and cache refresh in `internal/task.Store`.
- Reuse existing parsing and file-writing conventions; do not introduce a document renderer, transaction framework, or concurrency machinery.

### Data flow

- Require an explicit parent path; the `cmd` package accepts a Markdown file or resolves a directory to `top-level.md`. Pass the resulting file path to the Store, which does not resolve directory arguments.
- Use the explicit parent rather than discovering or creating a project from the working directory.
- Reuse the existing title prompt when `--title` is omitted; join description arguments as `shemiq new` does.
- Require a nonblank, single-line title and nonblank description; no slug generation or ASCII-character requirement is needed.
- Insert only the heading and description; do not add metadata or create other files.
- Append under `## Tasks` unless `--after` selects a sibling; insert after that sibling's complete section, including its description and nested content.
- Match `--after` and duplicate titles exactly and case-sensitively.
- Replace a sole `[TBD]` placeholder; append `## Tasks` at the document's end if missing; otherwise preserve existing section content.
- Use parser source offsets to preserve unrelated bytes, existing directives, and line-ending style.
- Refresh the cached parent after a successful write and print its absolute path to stdout.
- Preserve the parent's status exactly, including `done`; do not reopen or re-refine it.

### Error handling

- Require a parseable parent with primary `type: top-level` metadata inside a `.shemiq` project.
- Reject invalid input, duplicate sibling titles, missing or ambiguous `--after` matches, and ambiguous structure such as multiple `## Tasks` sections.
- Complete insertion-related checks before writing; rejected requests leave the document unchanged.
- Surface parsing, reading, and writing failures as command errors using existing conventions; prompts and diagnostics go to stderr.
- Do not invoke full validation, follow `source:` or `parent:` links, repair metadata, or reject unrelated metadata findings during insertion.

### Testing

- Keep coverage small and primarily end-to-end through the command, without duplicating CLI scenarios at the Store level.
- Cover prompted and explicit titles, default append and `--after`, directory resolution, placeholder replacement, and missing-section creation.
- Cover representative rejected requests and verify that they leave content unchanged.
- Verify unrelated content and parent-status preservation, and confirm a valid resulting document passes `shemiq validate`.
- Add focused Store-level coverage for cache refresh.

### Documentation

- Update `README.md` and `skills/shemiq/SKILL.md` with command usage and title-and-description-only creation behavior.
- Leave workflow prompts unchanged.

## Current status

- Store subtask insertion implemented in `internal/task/create_subtask.go`.
- `CreateSubtask` stages outside task directories, verifies before replacement, and refreshes the cache only after success; `applyEdits` simply writes to its explicit output path.
- `Store.CreateSubtask` API ready for CLI wiring; it loads an explicit Markdown file directly. Directory-to-`top-level.md` resolution belongs to the pending CLI task.
- CLI command implemented in `cmd/new_subtask.go`, registered in `cmd/root.go`.
- Fixed Store `--after` regression for TBD/missing-section layouts.
- Review follow-up: detailed insertion coverage lives in `internal/task`; CLI tests stay focused on command wiring and streams.
- README and skill file updated with `new-subtask` documentation.

## Tasks

### Add subtask insertion to the task Store
:::shemiq
type: task
source: ./store-subtask-insertion.md
status: done
:::

Implement the agreed parent-document insertion behavior, targeted preservation, insertion checks, and cache refresh with focused Store-level coverage.

### Expose and document the new-subtask command
:::shemiq
type: task
source: ./new-subtask-cli.md
status: done
:::

Wire the Store operation into the CLI with the agreed arguments, prompting, output, minimal end-to-end tests, and README and skill updates.

- Integration: use `Store.CreateSubtask(parentPath, title, description, after string) (string, error)`; empty `after` appends. The Store rejects structurally disruptive Markdown before writing; include representative rejection/unchanged-file checks in CLI end-to-end coverage. See `./store-subtask-insertion.md` for the approved contract.
- CLI follow-up: resolve directory arguments to `top-level.md` in `cmd` before prompting, and pass a Markdown file path to `CreateSubtask`. The Store rejects directories; keep directory-resolution end-to-end coverage in this CLI task.
