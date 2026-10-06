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
- Only the store component uses the private parser. Configure it through `NewStore(cwd, path string)` for the active project, one directory, or one file; validation may load additional referenced files.
- Syntax errors stop loading. There is no partial-document API.
- Initialization and loading do not require comprehensive semantic validity. Task conversion records local issues, and validation is an explicit operation.
- Support ATX headings and column-zero Shemiq directives, ignoring fenced code blocks and indented examples; do not introduce a full Markdown parser.
- Whole-project scope excludes archives. Archiving remains outside the parsed-store path.
- Expose children through `Task.Subtasks`, not a separate subtask-loading method, and use the generic `TaskByPath` lookup rather than `TopLevelTask`.
- A parent-list entry's path is its referenced child file, even when that file does not exist. Its title and status come from the parent entry, not the child document.
- Each `Task` links privately to one parsed section. A parent-list entry and a standalone child task are distinct representations, not a merged object.
- Invalid metadata produces empty/unusable fields plus located conversion issues; it does not fail task queries or silently substitute valid values. An omitted status still means `new`.
- Scoped-store refinement revision: commands do not inspect or report attached conversion issues. Refinement uses interpreted fields and ordinary workflow guards; validation reports issues. Interactive child selection includes only usable `task` entries with `new` status.
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
NewStore(cwd, path string) (*Store, error)
(*Store).TopLevelTasks() ([]Task, error)
(*Store).TaskByPath(path string) (Task, error)
(*Store).Validate(fix bool) ([]Issue, error)
(*Store).CreateTopLevel(title, description string) (string, error)
```

Pass the invocation directory and optional scope path directly to `NewStore`; no separate options type is needed. An empty scope path selects the nearest project's active tasks. The validation and creation subtasks will implement their respective methods without changing these responsibilities.

Configuration records the invocation directory and a scope: active project tasks, one directory, or one Markdown file. Directory scopes recursively select Markdown documents. Explicit scopes support archived tasks and other projects. Resolve relative selection paths against the invocation directory and metadata references against their containing document.

Configuration does not write files. Operations load the selected scope when they need document data; creation must work without an existing project and without parsing unrelated tasks. Read and validation operations must not create a project. Cache loaded documents and do not parse a file repeatedly unless a repair changes its contents. Task queries do not silently expand their configured scope; validation may do so by following existing local references.

Default-scope `TopLevelTasks` enumerates immediate active task directories in directory-name order, preserving errors for missing required `top-level.md` files. Explicit-directory scopes enumerate primary `top-level` tasks recursively in path order; explicit-file scopes return that task or an empty list. Queries do not filter statuses.

`TaskByPath` loads only the requested document within the configured scope and returns its primary task without imposing a top-level type or refinement eligibility; a directory argument resolves to its `top-level.md`, but file arguments need not have that basename. Generic loading supports documents outside `.shemiq`, leaving `ProjectRoot` empty. A missing document, filesystem failure, or syntax error remains an error. Semantic problems are represented by conversion issues instead.

### Single-document parsing

Parse original bytes into a synthetic document root and nested heading sections. Each section retains its title, source position/span, directives, and child sections. Retain precise directive and field positions, including field-value offsets, insertion positions, and line endings, so repairs can change metadata without reformatting surrounding Markdown.

Recognize column-zero ATX headings from `#` through `######` and column-zero `:::shemiq` directives. Ignore fenced code blocks and indented examples. Do not interpret ordinary Markdown beyond the structure needed for tasks, and do not resolve references or access the filesystem here.

Malformed openers, malformed field syntax, nested directives, and unterminated directives stop loading with file/line diagnostics. Preserve field occurrences rather than silently overwriting repeated keys. Unknown keys, repeated metadata fields, invalid types/statuses, and UUID validity belong to semantic interpretation rather than fatal syntax handling.

The store recognizes heading-adjacent task directives, allowing intervening blank lines, and converts entries under `## Tasks` into children. Other directives, including headingless and status-only metadata, remain available for validation even when they do not represent a task.

### Task representation and conversion

`Task` exposes type, title, status, path, project root, subtasks, and conversion
issues. It privately references its defining parsed document and section for
diagnostics and repairs. Parent-list entries reference their parent document,
not the source target. Commands never need the parser's structures. The store
caches only parsed documents. Its loader converts primary tasks and parent-list
entries on demand, without a stored-document wrapper or interpreted metadata
cache. Local directive checks are shared with validation.

A standalone task's path identifies its containing file. A parent-list entry's path is its resolved `source:` target, which may not exist yet; without `source:`, its path is empty. Its title and status are taken from the parent entry. Looking up the existing child file returns that document's separate task representation. Do not merge the two representations or replace parent-list metadata with child metadata.

Conversion fills fields that can be interpreted and records located issues for those that cannot. In particular, omitted status becomes `new`; a typo such as `refiend` produces an empty status and an associated issue, not a query error or fallback to `new`. Apply the same principle to other invalid fields. Findings must be attributable to the field that is unusable. Preserve metadata-only directives in parsed documents so validation can collect their local findings with the same conversion helper.

Queries return task trees and their issues without requiring a prior `Validate` call. An invalid child does not prevent listing or selecting an unrelated top-level task.

### Command integration

`cmd/refine.go` uses store queries, checks the selected task's `Type`, and routes according to its usable status. It selects children through `Task.Subtasks`, preserving exact-title matching, duplicate-title rejection, eligibility checks, interactive selection, and Pi launch requests.

Commands do not inspect, print, or filter by attached conversion issues; validation owns their reporting. Refinement retains ordinary type/status routing and eligibility guards. The top-level picker excludes `done` candidates, while interactive child selection includes only usable `task` entries with `new` status. Selecting an unusable type/status produces an ordinary workflow error, never a fallback to `new`. Unrelated findings, such as missing UUIDs, do not require a validation pass before refinement. Uncreated child files remain compatible with selecting parent-list entries. Explicit refinement accepts any Markdown filename and requires interpreted type `top-level`, rather than a conventional basename.

Preserve explicit refinement target containment after symlink resolution and derive the launch project root from the selected target, not merely the caller's working directory. Archive moves retain their own filesystem containment and symlink checks without involving parsing or validation.

### Validation and repair

`Store.Validate` collects conversion issues once and performs checks not covered by local conversion, including cross-document UUID uniqueness, reference validity, reciprocal task links, and linked status agreement. Retain validation of standalone metadata directives without requiring headings or a task type on every directive.

Follow existing local `source:` and `parent:` references, including targets outside the initial selection. Handle cycles without repeatedly loading files. A missing source file is allowed; a missing parent is reported. Keep syntax/loading failures separate from semantic findings, including when encountered in additionally loaded files.

With `fix`, retain the existing repairs: insert missing UUIDs on eligible source-less directives and promote the less advanced status only for valid reciprocal pairs. Do not guess invalid statuses, repair broken links, or rewrite unrelated text. Preserve original bytes and line endings outside targeted edits. Refresh affected parsed documents and task representations after repairs, then report repaired findings and remaining issues from the resulting state.

### Creation and testing

Creation preserves project discovery, title/description requirements, slug generation, directory collision rejection, generated UUIDs, and the current document layout. Creation alone may establish a project. Archive preserves whole-directory moves, UTC-dated destinations, collision rejection, and the ability to move a directory with invalid or absent task metadata.

Use existing command-level tests as the primary regression suite. Add only focused coverage for invalid-status conversion without mandatory validation, ordinary refinement routing guards, fatal syntax errors, and exclusion of fenced examples. Retain end-to-end coverage of selection, explicit paths, missing child files, creation, archiving, reciprocal status repairs, byte preservation, and repair idempotence. Update default validation-scope expectations to active tasks only.

## Current status

Single-document parsing and the scoped-store/task-model tasks are implemented
and marked `done`. The store now uses the private parser for lazy, scoped,
cached document loading, preserves original metadata and repair spans, and exposes
separate primary tasks and parent-list entries. Refinement uses store queries
and interpreted fields without inspecting attached issues; legacy refinement
loading and outline parsing have been removed. Review follow-up removed the
explicit filename check: refinement now relies on store loading and the
`top-level` type guard. Command tests now focus on workflow and launch behavior;
store tests cover overlapping scope/loading cases. Model review removed
`storedDocument`: tasks reference their defining documents. Further review
merged primary-task conversion into `loadDocument` and removed the metadata
cache and `interpretedDirective` type. Only parsed documents are cached;
local fields/findings are computed on demand. Focused regressions,
`go test ./...`, and `go vet ./...` pass.

Validation/repairs and creation/archive remain `new` and will be refined
separately. Legacy validation and repairs remain functional, sharing local
field checks with store conversion. Existing creation code has moved unchanged
to `create.go`; its public API and archive behavior are not yet migrated.
Validation can reuse the private loader, the returned task's defining document,
and `directiveFields` without imposing primary-task structure on metadata-only
documents. Refresh replaces parsed snapshots without a separate metadata cache.

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
status: done
:::

Implement scoped loading, single-node task conversion with attached issues, `TaskByPath` and `TopLevelTasks`, and migrate refinement to the store API.

Implementation handoff: store queries now consume the parser; legacy outline
and refinement loaders are retired.

### Consolidate validation and repairs
:::shemiq
type: task
source: ./consolidate-validation-and-repairs.md
status: new
:::

Implement store-based validation that collects conversion issues, loads referenced documents, checks cross-document semantics, and preserves existing targeted repair behavior.

Parser handoff: ordered field occurrences and repair offsets are ready; retain or relocate the shared syntax helpers when retiring `metadata.go` parsing.

Store handoff: `loadDocument(path, refresh)` returns a `Task` with its defining
parsed document. Traverse all directives in that document and collect local
findings with `directiveFields`; no metadata cache or interpreted-record type
remains. Refresh replaces parsed snapshots. Keep query-structure issues out of
validation, which must support metadata-only documents without task headings.

### Consolidate creation and archive boundaries
:::shemiq
type: task
source: ./consolidate-creation-and-archive-boundaries.md
status: new
:::

Move creation behind the store API, retain filesystem-only archive orchestration in its command, and remove superseded implementations and public entry points.

Store handoff: existing creation helpers now live unchanged in `create.go`;
`Store` retains invocation context and can be constructed before a project exists.
