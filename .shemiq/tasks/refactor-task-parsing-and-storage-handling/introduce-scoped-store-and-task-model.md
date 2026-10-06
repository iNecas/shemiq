# Introduce the scoped store and task model
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 165bc98e-08a6-4440-8c0b-c0066d8d8fe2
:::

## Context

Introduce the scoped store and task model described by the parent task, and
migrate refinement to that API. The completed parser task provides
`parseMarkdownDocument` and its private section tree, ordered directive fields,
original bytes, and source spans in `internal/task/parse.go`.

Current refinement uses `internal/task/refine.go` and `outline.go`, with its own
loading and heading/directive pairing. Replace that path with the store, not
another adapter that survives the migration. Keep the existing `internal/task`
package and the parent's five-file organization. Validation, repairs, creation,
and archive migration remain separate sibling tasks.

One parent-level decision was revised during this refinement: commands do not
inspect or report conversion issues. Tasks retain them for validation, while
refinement uses interpreted fields and ordinary workflow guards. Missing UUIDs
and uncreated child files remain compatible with refinement.

## Interview

Approved decisions:

- Configure scope through the invocation directory and optional selection path.
  Use `NewStore(cwd, path string)`, not a separate `StoreOptions` type.
- Resolve scope lazily. Construction must work before a project exists and must
  not parse documents or write files.
- Support `TopLevelTasks` for explicit scopes as well as default project scope;
  explicit directories enumerate recursively in path order.
- Refinement ignores attached issues. Its top-level picker excludes `done`
  tasks; unusable selected fields cause ordinary routing errors. Its child
  picker includes only eligible `new` tasks, without issue-based filtering or
  warnings.
- Retain the existing public task fields and issue shape. Metadata interpretation
  and parser references stay private to the store component.
- Migrate refinement now, leaving validation and creation functional on their
  existing paths until their sibling tasks are implemented.

## Implementation plan

### Store API and scope

Provide these entry points in `internal/task`:

```go
NewStore(cwd, path string) (*Store, error)
(*Store).TaskByPath(path string) (Task, error)
(*Store).TopLevelTasks() ([]Task, error)
```

`Store.Validate` and `Store.CreateTopLevel` belong to the sibling tasks; do not
add placeholder implementations here.

- Normalize invocation-relative paths to absolute paths. Defer project
  discovery, selection-kind checks, and document loading until an operation
  needs them. Construction neither creates nor requires an existing project.
- Empty selection path means the nearest project's active `.shemiq/tasks/`.
  Archives are excluded from this default scope. Read operations must not
  create a missing project.
- An explicit directory selects Markdown documents recursively. An explicit
  file selects only that Markdown document. Retain `.md` and `.markdown`
  support, including the existing case-insensitive extension handling.
- `TaskByPath` loads only the requested document within the configured scope.
  Resolve relative query paths against the invocation directory and directory
  arguments to their `top-level.md`. Explicit file arguments need not have
  that basename. Queries never follow metadata references automatically.
- Default-scope `TopLevelTasks` reads `top-level.md` from every immediate active
  task directory, in directory-name order. Preserve errors for missing required
  documents. Return these conventional candidates without rejecting semantic
  issues; the command decides whether a selected candidate is routable.
- Explicit-directory `TopLevelTasks` enumerates documents whose primary task
  has interpreted type `top-level`, in path order. Explicit-file scope returns
  that primary task as a singleton if it is `top-level`, otherwise an empty
  list. Queries do not filter statuses or produce an unfinished-tasks error.
- Filesystem failures, missing requested documents, and parser syntax errors
  remain operation errors. Semantic issues do not prevent returning tasks.
- Generic scoped loading can read Markdown outside `.shemiq`; use an empty
  `ProjectRoot` if the defining document has no enclosing project. Refinement
  retains its stricter project containment requirement.

### Cached loading and validation handoff

Keep scope selection, document loading, and interpretation in `store.go`, with
the task model and section-to-task conversion in `task.go`.

- Cache parsed documents for the store's lifetime; do not reread/reparse them
  for each query. Interpret metadata on demand without a separate cache
  (revised during implementation review). No filesystem watcher or public
  refresh API is needed.
- Keep a small private document-loading boundary that later validation can use
  to load references outside the initial scope and refresh affected documents
  after repairs. Query scope enforcement must not depend merely on whether a
  document happens to be in the cache.
- Preserve original bytes and parser spans. Do not flatten parsed metadata into
  a representation that loses duplicate occurrences or repair positions.
- Resolve filesystem paths and symlink aliases consistently for loaded
  documents and cache identity. Preserve refinement's checks against the
  resolved target; do not treat lexical placement under `.shemiq` as sufficient.
- Loading a parent document must not read its child files. Its task tree comes
  from sections within that document, including children not yet created.

### Task model and primary-task recognition

Extend `Task` with `Issues []Issue`, retaining `Type`, `Title`, `Status`, `Path`,
`ProjectRoot`, and `Subtasks`. Keep status as a string. Reuse the current issue
shape (`Path`, `Line`, `Message`, `Fixed`) rather than adding a diagnostic
framework or public UUID/reference API.

- Each task privately references its defining parsed document and section.
  Parent-list entries reference the parent document, not the source target.
  Parser types remain unavailable to commands.
- Recognize a document's primary task from its first level-one heading and
  heading-adjacent directive, allowing intervening blank lines. Do not search
  for a later valid definition to hide an unusable first one.
- Missing primary-task structure yields an unusable task with located issues,
  not a parsing error. Preserve its containing-file path and any independently
  usable fields.
- This primary-task requirement applies to task queries, not to every Markdown
  document selected for validation. Keep query-structure findings separate from
  general directive findings so metadata-only or headingless documents do not
  acquire a global heading/type requirement.
- Convert heading-adjacent entries under the primary section's conventional
  `## Tasks` / `### Entry` structure into `Subtasks`, regardless of the primary
  task's status. Entries without a task directive are ordinary sections.
- A standalone task's `Path` identifies its loaded file. A parent-list entry's
  `Path` is its source target resolved against the containing document, even
  when that target does not exist; absent or unusable `source` gives an empty
  path. Resolve reference paths without loading targets.
- Parent-entry titles and statuses come only from that entry. Its project
  ownership comes from the defining document. Loading an existing child file
  returns a separate task representation with its own project ownership;
  never merge the two objects or substitute child metadata into the entry.

### Local metadata conversion

Interpret task directives on demand during conversion. Preserve all parsed
root-level, status-only, and other metadata-only directives for validation to
check with the same local field helper. No interpreted-record type or metadata
cache is needed (revised during implementation review).

- Local checks cover unknown and repeated keys, empty or invalid values, known
  types/statuses, UUID format and requirements, and mutually exclusive
  `source`/`uuid`. Reuse or relocate existing rules where practical, without
  introducing a second new semantic pipeline.
- Invalid or repeated fields are unusable. Do not take the first occurrence of
  a repeated field or silently replace an invalid value with a default.
- An omitted status on an actual directive means `new`; an invalid or repeated
  status is empty and has an associated issue. A missing defining directive is
  not an omitted-status shortcut to a usable new task.
- Each task exposes findings for its own definition; children retain their own
  findings. Metadata-only definitions remain in the parsed document. Later
  validation must collect local findings once per directive using the shared
  helper, not duplicate them by recursively aggregating task issues as well.
- Conversion does not require filesystem validity of references, reciprocal
  links, cross-document UUID uniqueness, or linked status agreement. Those
  checks remain for the validation sibling task.

### Refinement migration

- Replace `ReadTopLevelTask` and `ListUnfinishedTopLevelTasks` calls in
  `cmd/refine.go` with scoped store queries. Explicit targets use an explicit
  scope; interactive selection uses default project scope.
- Use the selected task's already-converted `Subtasks`; do not reread the chosen
  document through another loading path.
- Do not inspect, print, or filter by `Task.Issues`. Preserve ordinary guards:
  require a top-level task, route `new` and `refined`, reject `done` and unusable
  statuses, and reject `--subtask` for a new top-level task.
- The top-level picker excludes `done` candidates. Exact-title child selection
  retains not-found and ambiguity rejection, and requires usable `task` type
  and `new` status. Interactive child selection includes only such eligible
  children and retains duplicate-title rejection.
- Preserve exact-title matching, cancellation, interaction, and Pi launch
  requests. Preserve missing-child compatibility without a validation pass.
- Accept explicit Markdown filenames based on interpreted `top-level` type,
  not basename (revised during implementation review). Preserve archived and
  cross-project targets, containment after symlink resolution, and launch
  project root derived from the selected document. These workflow restrictions
  do not belong in generic `TaskByPath`.
- Retire the legacy outline parser and public refinement-loading entry points
  after migrating callers. Rehome useful path helpers according to ownership.
  Leave legacy validation/repair and creation entry points working until their
  sibling tasks migrate them; retain shared helpers still needed by those paths.

### Minimal verification

Use existing command-level tests as the main regression suite; do not create a
broad new test framework.

- Replace the old refinement-model test with focused store assertions for scope
  selection, query boundaries and cached loading, and distinct parent-entry /
  child-file representations, including uncreated source targets.
- Cover unusable invalid/repeated status with attached located issues and
  omitted status becoming `new`. Assert that queries do not require validation.
- Adjust command expectations for invalid metadata: refinement now emits
  ordinary routing failures rather than conversion diagnostics.
- Extend only a few command scenarios to show that attached issues do not block
  otherwise usable refinement, invalid statuses never become `new`, and fatal
  parser syntax errors stop loading through the new store path.
- Retain existing selection, ambiguity, cancellation, missing child,
  explicit-file/directory, archive-target, and cross-project scenarios. Preserve
  resolved-target containment checks and read-only refinement behavior.
- Run `go test ./...` and `go vet ./...`.

### Sibling boundaries

- **Consolidate validation and repairs:** reuse cached parsed documents and the
  on-demand `directiveFields` helper. Follow references through the same
  loader, perform filesystem and cross-document checks, and refresh affected
  documents after targeted repairs. Continue supporting metadata-only documents.
- **Consolidate creation and archive boundaries:** use `NewStore(cwd, path)` and
  the retained invocation context; creation must not require parsing unrelated
  tasks or an existing project. Keep creation and archive migration out of this
  implementation.

## Implementation notes

- Added lazy `NewStore(cwd, path)`, scoped `TaskByPath`, and `TopLevelTasks` in
  `internal/task/store.go`. Construction only normalizes invocation-relative
  paths. Default queries select active task directories in name order; explicit
  directories scan recursively and explicit Markdown files support arbitrary
  basenames and case-insensitive extensions. Queries neither create projects nor
  follow metadata references, and semantic findings do not fail loading.
- Parsed documents are cached by resolved filesystem identity, including
  symlink aliases. The private `loadDocument(path, refresh)`
  boundary can load outside the query scope and refresh snapshots after repairs;
  query containment remains independent of cache membership. Original parser
  bytes, ordered field occurrences, and repair spans remain intact.
- `internal/task/task.go` now holds `Task`, `Issue`, and section-to-task
  conversion. Primary recognition uses the first level-one heading and its
  adjacent directive, with blank lines allowed and no fallback to later valid
  definitions. Structural query issues remain separate from general directive
  findings, so metadata-only validation can reuse the latter without imposing
  task headings or types.
- Task directives are interpreted on demand. The `directiveFields` helper
  returns unique usable fields and local findings without storing another
  representation; validation can reuse it for root/status-only metadata.
  Invalid or repeated fields remain unusable with located findings. Omitted
  status becomes `new` only on an actual directive. Local checks share
  `localFieldMessage` with legacy validation;
  reference existence, reciprocal links, UUID uniqueness across documents, and
  status agreement remain for the validation sibling.
- Parent-list entries are converted regardless of the primary status. Their
  titles/statuses and project ownership come from the defining parent section;
  source paths resolve independently, including uncreated targets and symlink
  prefixes. Referenced files are not read or merged into entries. Standalone
  child queries produce separate task representations.
- Migrated `cmd/refine.go` to store queries and the already-converted `Subtasks`.
  Refinement does not inspect issues: type/status/title guards determine routing,
  the top-level picker excludes `done`, and the child picker includes usable
  `task` entries with `new` status and nonempty titles. Exact-title matching,
  ambiguity/cancellation handling, missing-child compatibility, launch requests,
  archive/cross-project targets, and resolved-target project containment are
  preserved. Invalid/repeated statuses now cause ordinary
  workflow errors instead of conversion diagnostics.
- Removed `internal/task/refine.go`, `outline.go`, and the old refinement-model
  test. Relocated the existing creation implementation unchanged to `create.go`
  to prepare the final file organization; its public entry point and the legacy
  validation/repair/archive paths remain functional until their sibling tasks
  migrate them. Syntax helpers in `metadata.go` are still shared by the parser.
- Replaced the model test with four focused store scenarios covering lazy scope,
  boundaries/cache identity/refresh, explicit Markdown scopes, distinct entry
  and child representations, uncreated sources, located status findings, and
  primary versus metadata-only structure. Command tests retain existing flows
  and add issue-tolerant routing, invalid child eligibility, fatal syntax,
  fenced examples, done-candidate exclusion, and resolved-target containment.
- Review follow-up: removed `requireRefinementPath` and its duplicate filesystem
  inspection. Explicit Markdown documents can use any basename; the existing
  `TypeTopLevel` guard determines eligibility. Command tests cover an arbitrary
  Markdown filename and rejection of a standalone `task` document.
- Test review follow-up: trimmed the command path/loading matrix now covered
  by store tests, along with repeated top-level status and console retry/EOF
  variants. Command coverage focuses on launch requests, interactive/exact
  selection, routing guards, ambiguity, cancellation, and issue-tolerant
  refinement. Cross-project launch ownership and containment are consolidated
  in the resolved-target test, including invocation without a local project.
- Model review follow-up: removed `storedDocument`. `Task` privately retains
  its defining `*parsedDocument` and section; parent entries share the parent's
  document, and standalone child tasks reference their own. `Store.documents`
  directly caches parsed documents without a second primary-task representation
  or derived project ownership.
- Conversion review follow-up: merged `primaryTask` into `loadDocument`, which
  now returns a converted `Task` directly. Removed `Store.metadata` and the
  entire `interpretedDirective` type. Conversion uses temporary fields/findings
  from `directiveFields`; all original directives remain in the syntax-only
  parsed tree. Validation should traverse that tree and call the same helper,
  ignoring task-query structure findings. Successful refreshes replace the
  parsed snapshot; failed refreshes preserve it. Existing store tests cover
  document identity, metadata-only conversion, alias reuse, and refresh behavior.
- Verification: `go test ./...`, `go vet ./...`, and `git diff --check` passed.
  Task metadata was checked with `shemiq validate --fix` and revalidated.
