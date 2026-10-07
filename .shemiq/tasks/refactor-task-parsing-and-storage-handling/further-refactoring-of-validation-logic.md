# Further refactoring of validation logic
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 9fbb73d5-9801-41e5-8c30-cd2bc91a86bf
:::

## Context

`internal/task/validate.go` uses the scoped store and private parser, but its
validation flow still centers on documents and a separate `directiveRecord`
representation. Simplify loading and validation around stored `Task` objects
and their embedded subtasks, retiring the parallel document/record pipeline.

This follow-up intentionally revises parts of the earlier parent design:

- Cache document-level tasks instead of a `Store.documents` map.
- Load the complete configured scope before queries, rather than loading only
  the requested document or top-level candidates.
- Reject directives without an immediate heading association or outside the
  supported task positions. Headingless metadata is no longer supported.
- Include directive-less subtask headings as valid `new` subtasks.

Keep the existing public store operations and CLI interfaces. Creation/archive
migration remains a separate sibling task. Do not migrate those APIs here or
make construction depend on an existing project.

## Interview

Approved decisions:

- `Store.tasks` contains one document-level task per existing loaded Markdown
  file. Parent-list entries belong only to their parent's `Task.Subtasks`.
  An existing child file has its own separate stored task; never merge its
  metadata with the corresponding parent-list entry.
- Directives must immediately follow an allowed heading, with only blank lines
  between them and at most one directive per heading. Unsupported placement is
  a fatal parsing error, not a semantic finding.
- Allowed positions are the document's primary `#` heading and direct `###`
  subtask headings under that primary task's `## Tasks` section. Ordinary
  headings without directives remain allowed.
- Every direct `###` subtask heading creates a record, even without a directive.
  A directive-less subtask is valid, has type `task` and status `new`, and has
  no source path or UUID. It is eligible for refinement by title. Validation
  does not report a missing UUID for it, and repairs do not invent a directive.
- Prefer the simpler eager scope-loading flow, accepting that a syntax error
  in an unrelated scoped file can block a task query.
- Scope loading is idempotent initialization. After the first successful call,
  repeated calls do not rescan or reread files. Failed initialization is not
  marked complete and can be retried.
- Cached tasks are snapshots. References reuse the same private file loader;
  repairs explicitly refresh affected files. External changes require a new
  store. Previously returned task values remain snapshots after repairs.

## Implementation plan

### Task ownership and interpreted metadata

- Replace `Store.documents` with the document-level task collection. Each
  stored task privately retains its parsed snapshot, including original bytes,
  section references, directive positions, and field spans needed for repairs.
  Do not retain a second document registry or introduce a wrapper that recreates
  the old validation record model.
- A document-level task's path identifies its existing defining file. An
  embedded parent-list entry's path remains its usable resolved `source`
  target, possibly not yet created, or empty when no source is available.
  Both representations retain the defining parent's parsed references where
  appropriate; a child's file task owns its separate snapshot.
- Extend tasks with the interpreted UUID and source/parent reference data
  needed by validation. Resolve references against the defining document, not
  an entry's source target. Keep parser types private and leave the precise
  field layout to the implementation.
- Interpret existing directives through the shared local field rules when
  constructing tasks. Invalid/repeated fields remain unusable rather than
  falling back to the first occurrence or a guessed value. Omitted status is
  `new`; present but unusable status remains unusable.
- Keep local metadata findings distinguishable from query-only structure
  findings. Validation must report metadata findings once, without imposing
  refinement's primary-heading/type requirements on ordinary Markdown files.
  Files without directives remain valid to scan.
- Construct `Subtasks` from all direct `###` headings under the primary task's
  `## Tasks`, not only headings with directives. For a directive-less subtask,
  synthesize only its title, type `task`, status `new`, and defining references;
  leave its source path and UUID absent. Do not apply directive-specific field
  requirements when there is no directive.
- Preserve existing interpretation for subtasks that do have directives. The
  directive-less defaults must not hide invalid or repeated metadata on an
  existing directive.
- Additional resolved-path or UUID indexes are allowed where they simplify
  lookup, deduplication, or checks. Duplicate UUIDs must still be reported,
  rather than overwritten silently in an index.

### Parser placement rules

- Enforce supported placement in `parse.go`, so both queries and validation
  receive fatal file/line errors before any repair work.
- Permit a directive only immediately after the document's primary `#`
  heading or a direct `###` subtask heading under its `## Tasks` section.
  Allow blank lines between the heading and opener. Permit at most one
  directive per heading.
- Reject directives before any heading, after prose or other nonblank content,
  under ordinary sections, under the `## Tasks` heading itself, at unsupported
  nesting levels, after later non-primary `#` headings, or as a second directive
  for an already-associated heading.
- Continue accepting ordinary headings without directives, including the
  headings that define directive-less subtasks. Do not require a directive on
  every Markdown section.
- Preserve the existing ATX-heading and column-zero directive subset, fenced
  and indented example exclusion, fatal field syntax handling, source spans,
  ordered field occurrences, and original bytes/line endings.
- Placement is structural syntax; metadata type/status/UUID validity and
  repeated or unknown fields remain local semantic findings. Do not turn
  metadata value errors into parsing failures.

### Shared loading and cache lifecycle

- Keep `NewStore(cwd, path)` lazy, side-effect-free, and usable before a project
  exists. Queries and validation call private `Store.load` when document data
  is needed. Creation must remain independent of loading unrelated tasks.
- On the first successful `load`, resolve the configured scope, enumerate all
  selected Markdown files, and construct their document-level tasks and
  embedded subtasks. Perform parsing and local conversion, but no reference,
  uniqueness, reciprocal-link, or status-agreement validation.
- Preserve scope selection: default active `.shemiq/tasks/` only, recursive
  explicit directory, or explicit Markdown file; retain case-insensitive
  `.md`/`.markdown` handling and resolved-path identity.
- Mark initialization complete only after the whole scope loads successfully.
  Never return successful partial query results. If loading fails, return the
  error and allow a later call to retry without duplicating task records.
- After successful initialization, repeated `load` calls return immediately.
  Do not detect external edits, newly created files, or deletions automatically;
  callers create a new store to observe external changes.
- Use a private single-file loader for initial scope loading, additional
  reference loading, and explicit repair refreshes. Reuse cached tasks for
  already-loaded files and aliases. Do not add public refresh or graph APIs.
- Public queries select from the initialized scope, not every cached task.
  Preserve `TaskByPath` containment, directory-to-`top-level.md` resolution,
  default top-level candidate ordering, and missing conventional file errors.
  Preserve explicit-directory/file candidate behavior and status filtering
  ownership in commands.
- The intentional query behavior change is that every scoped Markdown file is
  loaded first: unrelated syntax/read errors may now block a query. Queries
  still do not require comprehensive semantic validity.

### Task-based validation and repairs

- `Store.Validate(fix)` first initializes the scope, then follows usable local
  source/parent references through the shared file loader. References may load
  existing Markdown files outside the initial scope. Deduplicate aliases and
  cycles by resolved path; additional stored tasks do not broaden public query
  results.
- Validate the selected scope and its reachable references, not unrelated
  cached files. Load the complete reachable set before planning or writing any
  repairs. Unexpected filesystem failures and syntax errors remain operation
  errors and stop the operation before repair work.
- Iterate document-level tasks and their embedded subtasks to collect local
  metadata findings and perform UUID uniqueness, reference suitability,
  reciprocal-link, and linked-status checks. Retire `directiveRecord`,
  `collectRecords`, document-centric validation traversal, and superseded
  loading helpers rather than adapting tasks back into those representations.
- Preserve missing-source allowance, missing-parent findings, invalid directory
  and non-Markdown target findings, and reciprocal-link ambiguity checks.
  Pair only usable task types/references and compare only usable statuses.
  Preserve existing located semantic diagnostics where practical.
- Preserve field-specific repair safety. Insert UUIDs only on existing
  directives where both `source` and `uuid` are absent in the original fields.
  Empty, invalid, and repeated occurrences still count as present. Generate
  UUIDs with `NewUUID`; never create metadata for directive-less subtasks.
- Preserve complete, one-run promotion across eligible reciprocal links under
  `new < refined < done`. Unusable statuses and broken/ambiguous links cannot
  carry promotions. Do not guess invalid values or repair links.
- Keep targeted original-byte edits, batching per file and combining inserts
  sharing an offset. Retain permissions, preferred line endings, and unrelated
  Markdown. Parsed references are used for locations, original field presence,
  and writes, not as a second semantic validation model.
- After successful writes, explicitly reload each affected file and replace
  its stored task and embedded subtasks. Update any indexes and use refreshed
  tasks for remaining findings and subsequent queries. Previously returned
  values remain snapshots; no in-place mutation is required.
- Return successful repairs with `Fixed: true`, followed by final-state
  unresolved findings. A second fix call on the same store must make no
  changes. No cross-file transactional rollback is required.

### Minimal verification and sibling boundaries

- Reuse existing command/store tests. Add headings and supported task placement
  to headingless semantic/repair fixtures; keep their issue assertions except
  for necessary diagnostic line changes. This agreed syntax change requires
  more fixture edits than a strictly behavior-preserving refactor.
- Add focused parsing coverage for unsupported positions, heading/directive
  adjacency, and multiple directives. Preserve existing syntax-error and
  fence/indentation coverage without expanding into a Markdown test framework.
- Cover a directive-less subtask in store queries and refinement selection,
  with `new` status and no validation finding or repair-generated directive.
- Cover eager whole-scope query failure from an unrelated syntax error,
  repeated-load cache reuse, failure retry without duplicates, and refreshed
  queries after repairs. Retain scope containment after reference loading.
- Retain reciprocal-link/status repair, invalid-field safety, UUID uniqueness,
  missing-source compatibility, byte preservation, and repair-idempotence
  coverage. Update private cache assertions from documents to stored tasks.
- Leave creation/archive migration untouched. Keep their current behavior
  functional and note only necessary handoff changes in the parent document.
- Run `go test ./...`, `go vet ./...`, and `git diff --check` when implementing.

## Implementation notes

### Summary of changes

**parse.go**: Added `validateDirectivePlacement` as a post-parse check
called at the end of `parseMarkdownDocument`. It builds a set of allowed
directives (primary heading's adjacent directive + direct `###` subtask
directives under `## Tasks`) and rejects any directive not in this set
with a fatal `unsupported directive position` error.

**task.go**: Extended `Task` with private fields `directive`, `uuid`,
`sourceRef`, `parentRef` computed during `sectionTask`. Directive-less
subtasks (`parentList=true`, no directive) now get `Type=task`,
`Status="new"` with no issues — no "missing uuid" or "missing directive"
findings.

**store.go**: Replaced `Store.documents map[string]*parsedDocument` with
`Store.tasks map[string]*Task` and added `loaded bool` for idempotent
eager scope loading via `Store.load()`. `loadDocument` became `loadTask`
returning `*Task`. `documentTask` extracted from `loadTask` constructs
the primary task with all `###` subtasks under `## Tasks` (not only those
with directives). `TaskByPath` and `TopLevelTasks` call `load()` first.

**validate.go**: Retired `directiveRecord`, `collectRecords`, and
`loadParsedDocument`. `reachableTasks` replaces `reachableDocuments`
using `taskReferenceTargets` on stored tasks. `validateAllTasks` computes
local findings from `directiveFields` on task directives (not `t.Issues`)
to avoid reporting structural issues. `repair` and `planStatusPromotions`
now use `*Task` instead of `directiveRecord`. Reference following checks
`os.Stat` before `documentPath` to handle non-existent targets gracefully.

### Review follow-up

UUID uniqueness checking is extracted into `validateUUIDUniqueness`.
`TopLevelTasks` now filters cached scoped tasks by usable `top-level` type,
without filesystem rescans or repeated `TaskByPath` calls. This revises the
original conventional-file candidate rule: default scope includes top-level
files regardless of filename and excludes other types. Empty task directories
no longer cause missing conventional-file errors when listing; explicit
`TaskByPath` directory requests still require `top-level.md`. Candidates stay
in path order, exclude out-of-scope references, and remain snapshots despite
external additions or deletions. Store and refinement tests reflect this.

### Test fixture changes

All headingless-directive fixtures were updated with supported headings.
Placement-specific error tests were added for headingless directives,
ordinary sections, second `#` headings, `## Tasks` heading itself,
two directives on one heading, and non-adjacent directives. The source-
edits test was restructured to place the non-directive content after the
closer rather than between heading and directive. Subtask count in
`TestStoreEntryAndChildAreDistinct` changed from 2 to 3 to include the
directive-less subtask, with assertions for its type/status/path.

### Handoff notes for creation/archive task

`Store.tasks` now replaces `Store.documents`. The `loadTask` method
returns `*Task` and caches by resolved path. Creation code in `create.go`
remains unchanged and still independent of the store's loading path.
Archive code is also unchanged.
