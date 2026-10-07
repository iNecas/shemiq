# Consolidate validation and repairs
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 0141f750-0fa6-40b4-9950-e2b5b1fda2bb
:::

## Context

Implement store-based validation and repairs described by the parent task.
The parser and scoped store are already implemented; validation still uses
legacy parsing and loading in `internal/task/metadata.go` and `validate.go`.
Replace that parallel pipeline rather than retaining a permanent adapter.

The private `loadDocument(path, refresh)` returns a `Task` whose `document`
contains original bytes, the parsed section tree, ordered field occurrences,
and repair spans. `directiveFields` returns unique usable fields and located
local findings on demand. Validation can use these boundaries without requiring
a primary task, headings, or a task type on every directive.

Keep the existing `Issue` shape (`Path`, `Line`, `Message`, `Fixed`) and the
parent's five-file organization. Creation and archive migration remain a
separate sibling task; do not migrate their public APIs here.

## Interview

Approved decisions:

- Use field-specific safety rather than blocking all work on a directive with
  any local finding. Unrelated errors remain reportable but do not prevent
  safe repairs. Repeated or invalid fields are unusable; never select their
  first occurrence as a fallback.
- UUID insertion eligibility depends on original field presence: both `source`
  and `uuid` must be absent, not merely unusable.
- Complete all safe status promotions in one run. Across eligible reciprocal
  links, propagate the highest usable status and write each affected status
  once. Do not reject chains or cycles or require repeated repair runs.
- Existing non-Markdown reference targets produce located invalid-reference
  findings, not operation errors. Keep the store's Markdown-only loader.
- Preserve the CLI interface, targeted edits, final-state reporting, and
  metadata-only validation. Default project validation now scans active tasks
  only, as already agreed in the parent plan.

## Implementation plan

### API, scope, and loading

Implement the agreed public boundary:

```go
(*Store).Validate(fix bool) ([]Issue, error)
```

- Migrate `cmd/validate.go` to `NewStore(cwd, path)` and `store.Validate(fix)`.
  Preserve `validate [path]`, `--fix`, stderr diagnostics, `fixed:` labels,
  and command failure for unresolved findings. Do not add public diagnostic,
  reference, graph, or refresh APIs.
- Resolve the configured scope lazily. Empty selection means Markdown files
  recursively under the nearest project's active `.shemiq/tasks/`. Archives
  are excluded unless reached through references. Explicit directories scan
  recursively; explicit files select that Markdown document. Retain
  case-insensitive `.md` and `.markdown` support.
- Enumerate documents independently of `TopLevelTasks` or `TaskByPath` queries.
  Validation does not require conventional filenames or task structure and
  never creates a project.
- Use the private store loader and the returned task's defining document.
  Traverse every directive, including root-level, headingless, and status-only
  metadata. Follow usable local `source` and `parent` fields regardless of
  whether the directive has a task type.
- References resolve against their containing document and may load files
  outside the initial scope. Deduplicate by resolved filesystem identity so
  aliases and cycles do not cause repeated loading. Validate the selected
  scope and its reachable references, not every document already in the cache.
  Loading references must not expand subsequent task-query scope.
- Load the complete reachable set before writing repairs. Unexpected filesystem
  failures and parser syntax errors, including in referenced documents, remain
  operation errors rather than semantic findings. A syntax/loading failure
  must not result in partial repair work.

### Local and cross-document validation

- Call `directiveFields` once per directive per semantic validation pass and
  collect its findings. Do not recursively aggregate `Task.Issues`: that would
  duplicate local findings and incorrectly impose query-only heading/type
  requirements. No interpreted-record cache or second semantic pipeline is
  needed.
- Use usable fields for cross-document checks. Check usable UUIDs for uniqueness
  across the reachable documents. Invalid/repeated UUIDs already have local
  findings and must not be silently selected for uniqueness checks.
- Check reference suitability independently of task structure. Missing source
  files are allowed while awaiting refinement; missing parent files are
  reported. Directory targets and existing non-Markdown targets are located
  invalid-reference findings and are not loaded. Unexpected inspection/read
  failures remain operation errors.
- Check reciprocal links only between directives with usable `type: task` and
  usable reference fields. Pairing requires exactly one matching source and
  exactly one matching parent in the two documents. Broken or ambiguous links
  are reported and cannot participate in status comparison or promotion.
  Headings and placement under `## Tasks` are not prerequisites.
- Compare statuses only when both are usable. An omitted status means `new`;
  invalid/repeated statuses do not have a fallback. Preserve existing located
  reciprocal-link and status-mismatch diagnostics where practical, without
  introducing redundant findings for unusable fields.

### Repair planning and targeted writes

- Insert a UUID only when neither `source` nor `uuid` occurs in the original
  directive. Empty, invalid, and repeated occurrences still count as present.
  Unrelated local findings do not block insertion. Generate canonical UUIDv4
  values through `NewUUID`; never replace invalid UUIDs or invent identities
  for source-bearing directives.
- Form eligible status links from unambiguous reciprocal task pairs with usable
  statuses at both endpoints. Within each connected group of eligible links,
  promote every lower status to the maximum under `new < refined < done`.
  Invalid/repeated statuses and broken/ambiguous links cannot carry promotions.
  Keep this algorithm private and narrow; no general graph framework is needed.
- Plan at most one status edit per directive. Replace only an existing field's
  value span, or insert an omitted field immediately before the closer. Never
  guess invalid statuses, repair broken links, or rewrite unrelated text.
- Batch non-overlapping edits per document against its original-byte snapshot.
  Combine insertions sharing an offset, including UUID and status insertions at
  the same closer. Preserve surrounding bytes, preferred line endings, and
  existing file permissions. No transactional rollback across files is required.
- Refresh affected parsed documents through the store loader after successful
  writes, then rerun semantic validation. Return successful repair findings
  with `Fixed: true`, followed by remaining findings from the resulting state.
  Do not report unresolved findings from stale pre-edit offsets or metadata.
- Fresh task queries use refreshed snapshots. Previously returned task values
  remain snapshots; no in-place mutation or public refresh mechanism is needed.
  A second `--fix` run must make no changes, including for linked chains/cycles.

### Retire legacy paths and preserve sibling boundaries

- Remove the package-level `Validate(cwd, path, fix)` after migrating its caller.
  Retire legacy document/directive types, `parseDocument`, selection/loading
  paths, and superseded validation helpers rather than adapting the new store
  back into the old representation.
- Move the syntax-only `metadataKeyPattern` and `lineEnding` helpers from
  `metadata.go` into `parse.go`, then remove the legacy metadata parser file.
  Keep shared Markdown/path helpers near their owner, reusing the store's path
  normalization and alias handling rather than retaining duplicate resolution.
- Leave `CreateTopLevel`, existing creation helpers, and archive orchestration
  functional for the creation/archive sibling task. This task consolidates
  validation only; no placeholders or additional store methods are needed.

### Minimal verification

Use existing command-level tests as the primary regression suite. Add focused
cases only where they exercise the changed boundaries or approved repair rules.

- Update default validation-scope fixtures to active tasks and verify archived
  metadata is excluded by default but included through explicit scope or a
  reachable reference.
- Separate fatal-syntax scenarios from semantic-invalidity scenarios. Assert
  syntax failure stops validation without repairs, including when reached
  through a reference; retain metadata-only validation and fenced-example
  exclusion coverage without duplicating the parser's full test matrix.
- Cover unusable/repeated references and existing non-Markdown targets with
  located findings, no guessed reference traversal, and continued validation
  of other documents. Retain missing-source compatibility, missing-parent
  findings, UUID uniqueness, and reciprocal-link cases.
- Extend a focused end-to-end repair scenario for linked-group convergence,
  one edit per status, and second-run idempotence. Retain existing CRLF/byte
  preservation, omitted-status insertion, invalid-status non-repair, and
  remaining-findings tests.
- Add a focused store-level check that validation refreshes repaired documents
  for subsequent queries without expanding their configured scope. Reuse
  existing store alias/cache tests rather than introducing a new framework.
- Run `go test ./...`, `go vet ./...`, and `git diff --check`.

## Implementation notes

Validation is now `(*Store).Validate(fix bool) ([]Issue, error)` in `validate.go`,
built entirely on the private parser and store loader. `cmd/validate.go` calls
`NewStore(cwd, path)` then `store.Validate(fix)`, preserving the CLI interface,
`fixed:` labels, stderr diagnostics, and command failure on unresolved findings.

Key points:

- `scopedSeeds` resolves the configured scope lazily (default active
  `.shemiq/tasks/`, explicit directory, or single Markdown file) and enumerates
  documents independently of `TopLevelTasks`/`TaskByPath`.
- `reachableDocuments` loads seeds and follows usable `source`/`parent`
  references via `loadParsedDocument`, deduplicating by resolved filesystem
  identity (through `documentPath`) so aliases and cycles load once. It does not
  enforce query scope, so references outside the selection are validated.
  Non-existent, directory, and non-Markdown targets are left for the semantic
  pass. Syntax/loading failures (including in references) surface as operation
  errors before any repair runs.
- Semantic checks operate on `directiveRecord`s (parsed directive + usable
  `directiveFields` + local findings), computed once per pass. All directives
  are traversed via `allDirectives`, including root-level/headingless and
  status-only metadata; no query-structure (heading/type) requirement is
  imposed. Cross-document UUID uniqueness uses only usable UUIDs. `referenceIssues`
  reports missing parents, directory targets, and existing non-Markdown targets;
  missing sources stay allowed.
- Reciprocal links pair only usable `type: task` directives with exactly one
  matching source/parent on each side. `statusRank` distinguishes omitted
  (`new`) from present-but-unusable (no rank). `planStatusPromotions` uses a
  small union-find over eligible pairs, raising every member of a connected
  group to the group maximum, so all promotions finish in one run and a second
  run is idempotent. In practice reciprocal pairs are independent groups; the
  union-find keeps the algorithm correct without a general graph framework.
- Repairs plan at most one UUID insert (when both `source` and `uuid` are absent
  in the original) and one status edit per directive, batched per document
  against the original snapshot via `applyEdits`. Insertions sharing the closer
  offset combine (UUID before status) using a stable sort. Affected documents
  are refreshed through `loadDocument(path, true)` before the final semantic
  pass, so remaining findings and later task queries use fresh offsets/state.

Legacy retirement: removed `metadata.go` (moving `metadataKeyPattern` and
`lineEnding` into `parse.go`), the package-level `Validate(cwd, path, fix)`, and
the legacy `document`/`directive`/`metadataField` types with their
`parseDocument`, selection, loading, and validation helpers. `isMarkdown` now
lives in `validate.go`. Creation (`create.go`) and archive remain untouched for
the sibling task.

Behavioral change surfaced by tests: a malformed metadata field is now a fatal
syntax error (from the parser) rather than a non-fatal finding, so the mixed
semantic/syntax fixture was split. Added focused tests:
`TestValidateFatalSyntaxStopsValidation` (direct and via reference),
`TestValidateNonMarkdownAndReachableArchive`, `TestValidateLinkedGroupConverges`,
and store-level `TestStoreValidateRefreshesWithoutExpandingScope`. Default-scope
fixtures moved to `.shemiq/tasks/`. `go test ./...`, `go vet ./...`, and
`git diff --check` all pass.
