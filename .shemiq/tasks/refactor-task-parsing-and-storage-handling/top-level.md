# Refactor task parsing and storage handling
:::shemiq
type: top-level
uuid: 62f017d6-7eae-48d9-8a42-50fb8ae3408b
status: refined
:::

## Description

refactor tasks parsing and storage handling

There are multiple in internal/task that handle some parts
of the parsing and it's not combined well together. For example:

- internal/task/refine.go and internal/task/outline.go
  doing it's own outline parsing and top level path detection
- internal/task/metadata.go
  doing raw parsing
- internal/task/validate.go
  doing more semantic validation, but not providing way to work with the data once validated

Overall, the API is terrible and leads to bloated and hard to maintain codebase.

Let's refactor the code and split it to the following:

- internal/task/parse.go
  - focues on parsing single markdown documents with focus on the structure we are interested in:
     - the nodes should be just sections with title, position, directives and children sections
     - the parsing will not go beyond single document
     - the parsing errors on this level should focus mostly on syntax errors (not semantics).
- internal/task/store.go
  - higher level parsing
  - will leverage parse for low-level data extraction, will keep Task representation of the loaded tasks, will provide an API to validate, create and query the tasks (in the scope of the current functionality).

Ideally, once refactoring is done, the most of the business logic from internal/{metadata,outline,archive,refine,validate}  would be either in parse.go or store.go, or inside corresponding cmd files.

Let's start first with refining the desired interface for store and parse, while considering the current code-base.

## Context

The current `internal/task` package has several overlapping representations and loading paths:

- `metadata.go` parses directives and retains byte offsets for metadata repairs.
- `outline.go` separately pairs headings with directives and identifies the top-level task list.
- `refine.go` resolves task paths, loads top-level summaries and subtasks, and checks metadata needed for refinement.
- `validate.go` selects files, follows references, validates metadata and links, and repairs UUIDs and statuses, without exposing a reusable task representation.
- `task.go` contains the task model, project discovery, and document creation; `archive.go` performs filesystem-only moves.
- `cmd/refine.go`, `cmd/validate.go`, `cmd/new.go`, and `cmd/archive.go` call these separate entry points.

The goal is a coherent store API and one private document parser, not new workflow features. The store owns task data and its interpretation; commands own interaction, workflow routing, launch requests, and output. The final implementation is split across five files rather than concentrating the entire store component in `store.go`.

Existing behavior to retain includes refinement before UUIDs or child files are available, explicit archived and cross-project refinement targets, exact-title selection and ambiguity rejection, byte-preserving metadata repairs, creation collision rejection, and archiving without reading task metadata. Whole-project loading intentionally changes to active `.shemiq/tasks/` only; archives require an explicit directory or file scope.

Existing command-level tests provide the main regression coverage. Keep new tests focused and avoid a broad new testing framework.

## Interview

Agreed decisions:

- Use a focused store for task loading, queries, validation, and creation; keep CLI interaction and workflow policy in commands.
- Only the store component uses the private parser. Configure the store for the active project, one directory, or one file; validation may load additional referenced files.
- Syntax errors stop loading. There is no partial-document API.
- Initialization and loading do not require comprehensive semantic validity. Task conversion records local issues, and validation is an explicit operation.
- Support ATX headings and column-zero Shemiq directives, ignoring fenced code blocks and indented examples; do not introduce a full Markdown parser.
- Whole-project scope excludes archives. Archiving remains outside the parsed-store path.
- Expose children through `Task.Subtasks`, not a separate subtask-loading method, and use the generic `TaskByPath` lookup rather than `TopLevelTask`.
- A parent-list entry's path is its referenced child file, even when that file does not exist. Its title and status come from the parent entry, not the child document.
- Each `Task` links privately to one parsed section. A parent-list entry and a standalone child task are distinct representations, not a merged object.
- Invalid metadata produces empty/unusable fields plus located conversion issues; it does not fail task queries or silently substitute valid values. An omitted status still means `new`.
- Use `parse.go`, `task.go`, `store.go`, `validate.go` including repair logic, and `create.go`.

## Design

### Ownership and organization

Keep the implementation in the existing `internal/task` package:

- `parse.go`: private single-document parsing, section structure, and source positions.
- `task.go`: the domain task representation and private section-to-task conversion.
- `store.go`: configuration, scoped document loading, caching, and task queries.
- `validate.go`: store-based validation, reference checks, repair orchestration, and targeted text edits.
- `create.go`: store-based creation, slug generation, and the existing document template.

The store is a component spanning these files, not a requirement to place every method in `store.go`. Parser types remain private and are not used by commands. Validation and repairs reuse the store's documents rather than creating another parsing/loading pipeline. Keep path helpers near their owners unless sharing justifies extracting them.

Move archive orchestration into `cmd/archive.go`, retaining its filesystem-only behavior and existing safety checks. Remove superseded implementations and old public entry points once their callers have migrated; temporary adapters during migration must not become a permanent parallel API.

### Store API and loading scope

The agreed high-level API is:

```go
NewStore(options StoreOptions) (*Store, error)
(*Store).TopLevelTasks() ([]Task, error)
(*Store).TaskByPath(path string) (Task, error)
(*Store).Validate(fix bool) ([]Issue, error)
(*Store).CreateTopLevel(title, description string) (string, error)
```

These are API sketches; refine exact configuration types in the corresponding subtask without changing these responsibilities.

Configuration records the invocation directory and a scope: active project tasks, one directory, or one Markdown file. Directory scopes recursively select Markdown documents. Explicit scopes support archived tasks and other projects. Resolve relative selection paths against the invocation directory and metadata references against their containing document.

Configuration does not write files. Operations load the selected scope when they need document data; creation must work without an existing project and without parsing unrelated tasks. Read and validation operations must not create a project. Cache loaded documents and do not parse a file repeatedly unless a repair changes its contents. Task queries do not silently expand their configured scope; validation may do so by following existing local references.

`TopLevelTasks` enumerates immediate active task directories in directory-name order. `TaskByPath` returns a document's primary task without imposing a top-level type or refinement eligibility; a directory argument resolves to its `top-level.md`. A missing document, filesystem failure, or syntax error remains an error. Semantic problems are represented by conversion issues instead.

### Single-document parsing

Parse original bytes into a synthetic document root and nested heading sections. Each section retains its title, source position/span, directives, and child sections. Retain precise directive and field positions, including field-value offsets, insertion positions, and line endings, so repairs can change metadata without reformatting surrounding Markdown.

Recognize column-zero ATX headings from `#` through `######` and column-zero `:::shemiq` directives. Ignore fenced code blocks and indented examples. Do not interpret ordinary Markdown beyond the structure needed for tasks, and do not resolve references or access the filesystem here.

Malformed openers, malformed field syntax, nested directives, and unterminated directives stop loading with file/line diagnostics. Preserve field occurrences rather than silently overwriting repeated keys. Unknown keys, repeated metadata fields, invalid types/statuses, and UUID validity belong to semantic interpretation rather than fatal syntax handling.

The store recognizes heading-adjacent task directives, allowing intervening blank lines, and converts entries under `## Tasks` into children. Other directives, including headingless and status-only metadata, remain available for validation even when they do not represent a task.

### Task representation and conversion

`Task` exposes type, title, status, path, project root, subtasks, and conversion issues. It retains a private reference to its one defining parsed section for diagnostics and repairs. Commands never need the parser's structures.

A standalone task's path identifies its containing file. A parent-list entry's path is its resolved `source:` target, which may not exist yet; without `source:`, its path is empty. Its title and status are taken from the parent entry. Looking up the existing child file returns that document's separate task representation. Do not merge the two representations or replace parent-list metadata with child metadata.

Conversion fills fields that can be interpreted and records located issues for those that cannot. In particular, omitted status becomes `new`; a typo such as `refiend` produces an empty status and an associated issue, not a query error or fallback to `new`. Apply the same principle to other invalid fields. Findings must be attributable to the field that is unusable. Keep issues from metadata-only directives in the store as well.

Queries return task trees and their issues without requiring a prior `Validate` call. An invalid child does not prevent listing or selecting an unrelated top-level task.

### Command integration

`cmd/refine.go` uses store queries, checks the selected task's `Type`, and routes according to its usable status. It selects children through `Task.Subtasks`, preserving exact-title matching, duplicate-title rejection, eligibility checks, interactive selection, and Pi launch requests.

Commands check the fields required for the current workflow step and report their corresponding conversion issues when unusable. Invalid statuses must not silently disappear during filtering or become eligible defaults. Unrelated findings, such as missing UUIDs, do not require a comprehensive validation pass before refinement. Uncreated child files remain compatible with selecting parent-list entries.

Preserve explicit refinement target containment after symlink resolution and derive the launch project root from the selected target, not merely the caller's working directory. Archive moves retain their own filesystem containment and symlink checks without involving parsing or validation.

### Validation and repair

`Store.Validate` collects conversion issues once and performs checks not covered by local conversion, including cross-document UUID uniqueness, reference validity, reciprocal task links, and linked status agreement. Retain validation of standalone metadata directives without requiring headings or a task type on every directive.

Follow existing local `source:` and `parent:` references, including targets outside the initial selection. Handle cycles without repeatedly loading files. A missing source file is allowed; a missing parent is reported. Keep syntax/loading failures separate from semantic findings, including when encountered in additionally loaded files.

With `fix`, retain the existing repairs: insert missing UUIDs on eligible source-less directives and promote the less advanced status only for valid reciprocal pairs. Do not guess invalid statuses, repair broken links, or rewrite unrelated text. Preserve original bytes and line endings outside targeted edits. Refresh affected parsed documents and task representations after repairs, then report repaired findings and remaining issues from the resulting state.

### Creation and testing

Creation preserves project discovery, title/description requirements, slug generation, directory collision rejection, generated UUIDs, and the current document layout. Creation alone may establish a project. Archive preserves whole-directory moves, UTC-dated destinations, collision rejection, and the ability to move a directory with invalid or absent task metadata.

Use existing command-level tests as the primary regression suite. Add only focused coverage for invalid-status conversion and reporting without mandatory validation, fatal syntax errors, and exclusion of fenced examples. Retain end-to-end coverage of selection, explicit paths, missing child files, creation, archiving, reciprocal status repairs, byte preservation, and repair idempotence. Update default validation-scope expectations to active tasks only.

## Current status

High-level refinement and task split are approved. Single-document parsing is implemented and marked `done`: `internal/task/parse.go` provides the private section tree, original-byte spans, ordered metadata fields, fatal syntax diagnostics, and fenced-example exclusion. Focused parser tests, `go test ./...`, and `go vet ./...` pass. Existing callers and legacy parsing are intentionally unchanged; the scoped-store and validation subtasks will integrate the parser and retire those paths. The remaining subtasks are still `new` and will be refined separately.

## Tasks

### Unify single-document parsing
:::shemiq
type: task
source: ./unify-single-document-parsing.md
status: done
:::

Introduce the private section-based parser with byte-preserving source positions, fatal syntax errors, and exclusion of fenced and indented examples.

Refinement note: introduce the parser independently without migrating existing callers. The scoped-store and validation subtasks will integrate it and remove the corresponding legacy parsing paths.

### Introduce the scoped store and task model
:::shemiq
type: task
source: ./introduce-scoped-store-and-task-model.md
status: new
:::

Implement scoped loading, single-node task conversion with attached issues, `TaskByPath` and `TopLevelTasks`, and migrate refinement to the store API.

Parser handoff: consume `parseMarkdownDocument` and its section/heading spans; legacy outline parsing remains until this migration.

### Consolidate validation and repairs
:::shemiq
type: task
source: ./consolidate-validation-and-repairs.md
status: new
:::

Implement store-based validation that collects conversion issues, loads referenced documents, checks cross-document semantics, and preserves existing targeted repair behavior.

Parser handoff: ordered field occurrences and repair offsets are ready; retain or relocate the shared syntax helpers when retiring `metadata.go` parsing.

### Consolidate creation and archive boundaries
:::shemiq
type: task
source: ./consolidate-creation-and-archive-boundaries.md
status: new
:::

Move creation behind the store API, retain filesystem-only archive orchestration in its command, and remove superseded implementations and public entry points.
