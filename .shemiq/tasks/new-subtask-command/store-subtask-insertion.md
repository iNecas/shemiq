# Add subtask insertion to the task Store
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 31359542-1329-4f13-8992-0625dee1becb
:::

## Context

- Support `shemiq new-subtask` by inserting one directive-less subtask into an existing top-level document. This task owns the Store operation, insertion checks, targeted editing, and cache refresh; the sibling task owns CLI wiring, end-to-end coverage, and documentation.
- `internal/task/store.go` owns Store loading, path resolution, and cached task snapshots; `internal/task/create.go` provides the existing `CreateTopLevel` API convention.
- `internal/task/parse.go` retains heading and complete-section source offsets. `internal/task/validate.go` provides targeted-edit and post-write cache-refresh conventions.
- Directive-less `###` entries under `## Tasks` already default to `type: task` and `status: new`. Separate subtask documents are created later during refinement, not by this operation.

## Interview

- Choose an explicit-path Store operation, independent of its configured read scope, rather than requiring a scoped query or caller-supplied Task.
- Reject structurally disruptive Markdown rather than rewriting it or inserting it unchecked.
- Use the cached parent snapshot when available; read only when uncached. Reread and refresh only after a successful write, not before insertion. No external-edit protection is required.
- Keep Store testing focused on cache refresh; leave user-facing insertion scenarios to the sibling CLI task.

## Implementation plan

### API and loading

- Add `Store.CreateSubtask(parentPath, title, description, after string) (string, error)`; empty `after` means append. Return the edited parent's absolute path on success.
- Load an explicit Markdown file path, resolving relative paths from the Store's invocation directory. Directory-to-`top-level.md` resolution belongs to the sibling CLI task in `cmd`, not the Store.
- Operate independently of the configured read scope. Load only the explicit parent, without scanning unrelated documents or discovering/creating a project from the invocation directory.
- Reuse a cached parent snapshot when available; otherwise load it. After a successful write, reread and refresh the cached parent so subsequent queries see the insertion. Previously returned Tasks remain snapshots.
- Do not add locking, concurrency machinery, or a pre-insertion freshness check; external changes since caching may be overwritten.

### Preconditions and matching

- Require a parseable Markdown parent inside a `.shemiq` project, with an unambiguous primary heading and usable `type: top-level` metadata.
- Reject ambiguous Tasks structure, including multiple `## Tasks` sections.
- Trim surrounding title whitespace; require a nonblank, single-line title and a nonblank description. Do not require a slug or ASCII characters.
- Ensure the rendered heading parses back to the intended title, rather than silently changing its identity through Markdown heading syntax.
- Compare the new title and `after` against parsed sibling titles exactly and case-sensitively. Reject a duplicate new title or a missing/ambiguous `after` match.
- Do not invoke full validation, load linked documents, repair metadata, or reject unrelated semantic findings. Preserve the parent's status exactly, including `done`.

### Insertion and preservation

- Insert only `### <title>`, a blank separator, and the description. Do not add a directive, UUID, source link, or separate file.
- Append at the end of `## Tasks`, before a following peer section. With `after`, insert after the selected sibling's complete section, including its description and nested content.
- Replace `[TBD]` only when it is the sole non-whitespace content of the Tasks section; otherwise preserve existing content. Append a missing `## Tasks` section at the document's end.
- Use parser source offsets and existing targeted-edit conventions instead of whole-document regeneration. Preserve bytes outside the targeted edit, including existing directives and status.
- Use surrounding line-ending style for inserted content and add necessary blank-line separators.

### Structural safety and errors

- Build and parse the proposed result in memory before writing. Verify it adds exactly the intended directive-less sibling without changing existing section relationships.
- Reject descriptions that create extra tasks, escape the inserted subtask, introduce an actual `:::shemiq` directive, or cause fence interactions that swallow existing content.
- Support ordinary Markdown, nested headings (`####` or deeper), and fenced examples when they preserve the intended structure. Preserve accepted content rather than escaping or demoting it automatically.
- Finish insertion checks before writing; rejected requests leave file contents unchanged. Surface read, parse, and write failures as ordinary operation errors.
- Reuse existing parsing and writing conventions; do not introduce a document renderer or transaction framework.

### Testing and sibling integration

- Add focused Store coverage that preloads the parent, inserts a subtask, and queries the same Store to verify the refreshed subtasks.
- Keep default/`after` placement, directory resolution, placeholder and missing-section handling, preservation checks, and representative rejected Markdown in the sibling task's minimal CLI end-to-end coverage; do not duplicate those scenarios here.
- The CLI task should call the agreed Store API, print its returned absolute path, and leave structural-input checks in the Store. It remains a separate, unimplemented task.

## Implementation notes

- Created `internal/task/create_subtask.go` with `Store.CreateSubtask(parentPath, title, description, after string) (string, error)`.
- `CreateSubtask` loads the explicit Markdown file via `loadTask`, reusing its path resolution and cached snapshot; project ownership and top-level type checks stay inline before input validation.
- Review update: removed `resolveParentPath` and Store-side directory resolution. The next CLI task must resolve directory arguments to `top-level.md` in `cmd`; its entry in `top-level.md` records this handoff.
- Title validation ensures the heading round-trips through `markdownHeading` parser.
- Review update: shared `applyEdits(doc, edits, outPath)` in `internal/task/edit.go` only writes edited content to the explicit output path. `CreateSubtask` owns temporary staging outside the task directory, verification, cleanup, and replacement.
- Structural checks parse the staged file using the original document path; the Store cache refreshes only after successful replacement.
- Replacement preserves permissions, cleans up temporary files, and handles cross-filesystem moves via a local rename after verification.
- Handles `[TBD]` placeholder replacement, missing `## Tasks` section (appends), `after` placement, and blank-line separators.
- Validation repairs pass `doc.path` directly to `applyEdits`; `loadTask(path, true)` still refreshes changed snapshots.
- `edit_test.go` covers explicit output paths, in-place writes, byte preservation, and snapshots. Store tests cover staging cleanup on success/rejection, unchanged rejected files, and replacement permissions.
- `go test ./...` passes.
- Follow-up fix during CLI task: `--after` with missing `## Tasks` or `[TBD]` placeholder now correctly rejected instead of silently ignored.
