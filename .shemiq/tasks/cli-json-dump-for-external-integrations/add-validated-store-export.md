# Add validated merged-tree store export
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 3eeae0fa-d14b-46c5-a015-70011f3d1564
:::

## Context

- Provide the read-only store operation and Go JSON contract consumed by the dump command and TypeScript-generation sibling tasks.
- `internal/task/store.go` owns scope resolution, parsed snapshots, resolved paths, and cached documents.
- `internal/task/task.go` contains task metadata, structural/local issues, and private UUID/source/parent fields.
- `internal/task/validate.go` already provides `reachableTasks()`, which follows both source and parent references from scoped documents, deduplicates resolved document paths, and returns documents in path order.
- Reuse that existing closure unchanged for both validation and export; do not introduce a separate descendant-only export traversal.
- This supersedes the parent's original ancestor-exclusion rule: starting at a child can include ancestors and their referenced sibling branches, including documents outside the starting directory.
- No command implementation, TypeScript generation, Markdown content, available actions, or Pi integration belongs in this subtask.

## Interview

- Use `Store.Dump() (DumpDocument, error)` with a typed validation error carrying located issues.
- Keep dedicated export structs separate from parser structs, but colocate them with the store export in `internal/task/dump.go`; no separate DTO package is needed. Use names such as `DumpDocument` and `DumpNode` to distinguish the JSON contract from the internal `Task` model.
- Use independent named string types and explicit values for task types and statuses, suitable for generated TypeScript literal unions.
- Reject multiple owning parents instead of duplicating children or choosing an owner.
- Reject main-heading `source:` in ordinary validation, not just dump-specific validation.
- Also reject entry-level `parent:` in ordinary validation; reciprocal links connect a parent-list entry to the referenced document's primary task.
- Validate and export one closure following both source and parent references, reusing the current validation traversal.
- API/validation design and revised single-closure tree design approved.

## Implementation plan

### File layout

- Add `internal/task/dump.go` for `Store.Dump()`, export DTOs/types/constants, merged-tree construction, DTO conversion, and dump-specific `ValidationError`/`validationIssuesToError`.
- Keep `dump.go` in package `task` to reuse private store/validation helpers and parsed metadata without exposing them publicly. Leave loading and caching responsibilities in `internal/task/store.go`.
- Extend `internal/task/validate.go` with shared structural/reference/graph validation; preserve `Validate`'s return signature.
- Add `internal/task/dump_test.go` for store-export tests and extend `internal/task/validate_test.go` for the new main-validation rules.

### Store API and export contract

- Add `(*task.Store).Dump() (DumpDocument, error)`.
- Accept the existing default project scope or an explicit directory scope; reject file scopes without changing other store operations' support for files.
- Define dedicated export structs in `internal/task/dump.go`, independent of the internal task/parser structs; do not serialize `Task` directly:
  - `DumpDocument` contains `Tasks []DumpNode`, with JSON key `tasks`.
  - `DumpNode` contains `type`, `title`, `status`, `projectRoot`, `path`, `uuid`, and `subtasks` with those exact JSON keys.
  - Define independent named string types and constants for `top-level | task` and `new | refined | done`, compatible with generating TypeScript literal unions.
  - `path` and `uuid` are optional strings using `omitempty`.
  - Initialize task/subtask collections so empty JSON arrays are `[]`, never `null`.
  - Do not include Markdown, available actions, issues, or a schema-version field.
- Return no usable export document on failure.
- Loading and parsing failures remain ordinary errors. Semantic failures use `task.ValidationError` carrying `Issues []task.Issue`, accessible through `errors.As` for command diagnostics.
- Keep the existing `Store.Validate` return contract unchanged.

### Read-only validation

- Reuse scope loading, cached snapshots, `reachableTasks()`, and existing validation helpers rather than adding another reference-discovery implementation.
- Validate all reached documents with no repairs, including UUID uniqueness, reference suitability, reciprocal links, and linked statuses.
- Extend main validation with reference-placement rules:
  - `source:` is permitted only on parent-list entries under `## Tasks`, not on a document's main heading.
  - `parent:` is permitted only on the child document's main task, not on parent-list entries.
  - Reciprocal checks connect the parent's source entry to the child document's primary task.
  - Report these findings with field locations; do not repair them.
- Review-approved change: both validation and dump include all parsed `Task.Issues`, including missing primary headings/directives, missing task types, and empty titles; no separate dump-only structural validation.
- Combine overlapping findings without duplicate diagnostics and maintain deterministic diagnostic ordering.
- Check child-reference cycles and multiple owning parents throughout the reached closure. Report located errors rather than truncating, duplicating, or selecting an owner.
- Cycle detection must cover the entire graph, including components with no roots.
- Preserve the existing allowance for missing source targets.
- Never write files, insert UUIDs, promote statuses, create a project, or invoke Pi during dump.

### Merged-tree construction

- Use the same complete document closure for validation and export; both source and parent references expand it.
- Use resolved absolute document paths for identity and relationships, reusing existing symlink/reference handling. Do not infer links from UUIDs or titles.
- Parent-list source links determine ownership and hierarchy, regardless of which document was initially scoped.
- For each parent-list entry:
  - If its source document exists, replace entry metadata with the document's authoritative primary-task metadata and recursively merged children.
  - If its source document does not exist, retain entry metadata and its resolved absolute target path; omit UUID.
  - If it is inline, retain its metadata and omit path.
- Export UUIDs when available, including standalone documents and UUID-bearing inline entries.
- Existing documents use their own `projectRoot`; missing-source and inline nodes use their defining document's ownership. Generic documents outside `.shemiq` retain an empty `projectRoot`.
- Every reached document without an owning parent becomes a root, regardless of task type. Do not emit merged children again as separate roots.
- Sort roots by resolved path; preserve parent-list order for siblings.
- An existing valid empty directory returns a document whose `tasks` array is empty. Missing directories and file selections fail.

### Minimal verification and sibling handoff

- Add one representative store-level success scenario covering authoritative child metadata, missing-source and inline children, external references, ordering, and root deduplication.
- Include a child-directory selection that recovers the ancestor and its referenced sibling branches through the existing closure.
- Check empty-directory success and JSON array/optional-field behavior.
- Keep individual structural/reference/graph failure cases in validation tests, reusing existing command coverage. Dump tests check shared validation-before-export flow, typed findings, no usable document, and file-scope rejection.
- Verify representative input files remain unchanged.
- Reuse existing fixtures and tests; avoid broad parser retesting or additional test infrastructure.
- The command sibling consumes `Store.Dump` and handles typed findings on stderr before producing JSON; command-level end-to-end coverage remains there.
- The generation sibling selects the export declarations in `internal/task/dump.go`, including literal-union types, rather than generating the internal task/parser model. Pinned Tygo generation, committed output, and drift checks remain there.

## Implementation notes

- Added `internal/task/dump.go` with `Store.Dump()`, export DTOs (`DumpDocument`, `DumpNode`), independent `DumpTaskType`/`DumpStatus` named string types with explicit constants.
- Shared validation lives in `internal/task/validate.go`; dump-specific `ValidationError`, its `Error()` method, and `validationIssuesToError` live in `dump.go`.
- Reference placement, reciprocal links, cycles, and multiple-owner checks are shared with ordinary validation; reciprocal links match source entries only to document primary tasks, regardless of task type.
- Removed `validateFullTasks`: both APIs use `validateAllTasks`, collecting cached `Task.Issues` instead of repeating `directiveFields`; ordinary validation now also reports structural findings.
- Review fixes: removed duplicate placement checks, deduplicated/sorted diagnostics, and corrected DFS to finish every branch with findings located at reference fields.
- Merged-tree construction: authoritative child metadata from existing source documents, entry-backed nodes for missing sources, inline entries without path. Roots sorted by path, siblings in parent-list order.
- Empty collections serialize as `[]`, never `null`. `path` and `uuid` use `omitempty`.
- Dump tests cover merged trees, JSON roundtrip equality, empty-directory/file-scope behavior, ancestor recovery, and one read-only validation-before-export scenario over an external reference.
- Structural/hierarchy validation assertions live in `validate_test.go`; `task_test.go` stays focused on conversion/query behavior. Existing command tests cover headingless documents, missing types, and misplaced backlinks.
- Reused scope/source-exception tests for the stricter validation; dump-flow coverage checks equality with `Validate(false)` findings, no usable export, and unchanged inputs.
- Verification passed: `go test ./...`, `go vet ./...`, task metadata validation, and `git diff --check`.
