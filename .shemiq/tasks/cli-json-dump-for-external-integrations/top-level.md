# CLI JSON dump for external integrations
:::shemiq
type: top-level
uuid: 319b7f05-6a7f-4ad8-acf8-cbdd446dbf58
status: refined
:::

## Description

- Support a future interactive UI inside the Pi coding agent for choosing tasks and actions.
- Reduce duplicated implementation and improve code reuse by keeping the Pi side focused on UI integration; Markdown parsing, discovery, and storage remain the CLI's responsibility.
- Provide a CLI JSON representation consumable by JavaScript integrations, with a shared Go/TypeScript data contract.
- This task delivers task metadata and a merged hierarchy, not the Pi UI, Markdown content, or available-action lists.

## Context

- `internal/task/store.go` provides lazy, directory-scoped loading of Markdown task documents and caches parsed snapshots; existing queries do not follow metadata references or merge parent entries with child documents.
- `internal/task/task.go` contains task metadata, embedded parent-list entries, and located structural/local issues; UUIDs and resolved references are currently private.
- `internal/task/validate.go` provides read-only validation and optional repairs, follows source/parent references, and checks UUID uniqueness, reciprocal links, and linked statuses; dump must also reject structural task issues.
- Missing subtask source files are allowed under existing validation rules, supporting subtasks that have not yet been refined into separate documents.
- Parent-list titles can differ from child-document titles without failing existing validation; dump uses child-document metadata when the file exists.
- `cmd/root.go` registers Cobra commands and separates result output from diagnostics; the dump command follows that pattern.
- `cmd/refine.go` and `cmd/implement.go` retain workflow eligibility and routing; this export does not expose available actions.
- The project currently has no JSON export contract or TypeScript generation tooling; command-level tests can reuse the existing command harness, and `Makefile` provides Go test/vet checks.

## Interview

- Export metadata and hierarchy only, without Markdown content or available actions.
- Use a dedicated, noninteractive `shemiq dump [dir]` command; the path argument must be a directory, not a Markdown file.
- Treat the directory as a starting store scope, rather than exporting an individual document.
- Require full read-only validation; invalid files prevent a successful dump, including cross-document inconsistencies and structural problems.
- Export a merged tree, not independent records for parent entries and child files.
- Use existing child-document metadata as authoritative; parent lists determine hierarchy and sibling order, with entry-backed nodes for missing files.
- Store-export refinement supersedes ancestor exclusion: reuse the existing validation closure for export, following both source and parent references, including ancestors and their referenced sibling branches outside the starting directory.
- Use a typed validation error, reject multiple owning parents, and enforce source/parent placement in main validation.
- Use dedicated Go export DTOs as the source of truth and pinned Tygo generation for committed TypeScript types.
- Approve the minimal `{ "tasks": [...] }` payload, optional path/UUID, absolute paths, deterministic ordering, and empty arrays.
- Approve all-or-nothing output, stderr diagnostics, cycle rejection, empty-directory success, and minimal command-level tests.
- Approve the three-subtask split below.

## Design

### Architecture and components

- Keep the Cobra command thin: create the store, request a read-only dump, and serialize the result.
- Put export behind `Store.Dump() (DumpDocument, error)` in `internal/task/dump.go`; reuse shared validation from `validate.go` before merged-tree construction. The command must not inspect parser internals or assemble relationships itself.
- Keep dump-specific `ValidationError`, its `Error()` method, and `validationIssuesToError` in `dump.go`; semantic failures carry located issues.
- Define dedicated export structs such as `DumpDocument` and `DumpNode` in the same file, separate from the internal task/parser structs; no separate DTO package is needed.
- Do not add Pi UI implementation, action eligibility exports, Markdown exports, runtime TypeScript validation, or a new integration framework.

### Scope and data flow

- `shemiq dump [dir]` never prompts and accepts directories only.
- Without an argument, discover the nearest project's active `.shemiq/tasks` directory using existing store behavior.
- Recursively load an explicit starting directory using existing Markdown/store rules.
- Perform full validation without repairs, including structural task issues, then construct the merged tree.
- Reuse the existing `reachableTasks()` closure for both validation and export; follow source and parent references without a separate discovery traversal.
- Export every reached document, including ancestors and their referenced sibling branches outside the starting directory.
- Existing child documents supply authoritative metadata; parent lists determine hierarchy and sibling order.
- Missing source targets remain entry-backed nodes, without requiring a child file or UUID.
- Do not duplicate exported children as roots; reached documents without an owning parent remain roots.
- Sort roots by path and retain parent-list order for children.

### JSON and cross-language contract

- Return one JSON object with a `tasks` array.
- Each node contains `type`, `title`, `status`, `projectRoot`, and `subtasks`.
- Include `path` as a resolved absolute document/target path when available; omit it for inline entries without a target.
- Include `uuid` when available; omit it for uncreated source-backed subtasks without a UUID.
- Encode empty task/subtask collections as `[]`, not `null`.
- Do not include Markdown, available actions, validation issues, or a schema-version field.
- Use Go DTOs and their JSON tags as the source of truth for TypeScript generation, with independent named string types/constants yielding literal unions for task types and statuses.
- Configure Tygo to select the export declarations in `internal/task/dump.go`, not the internal task/parser model; run a pinned version through `go run` and expose generation through `make generate-types`.
- Commit the generated TypeScript artifact so consumers do not need to run generation.
- Add a regeneration drift check and generation documentation; no Node/npm tooling or globally installed generator is required.
- Keep DTO types compatible with Tygo rather than accepting unsupported types silently becoming `any`; generation provides compile-time types, not runtime validators.

### Error handling

- Finish loading, validation, tree construction, and JSON encoding before writing the result.
- Validity failures return an error/nonzero exit, report diagnostics on stderr, and produce no JSON.
- Reject unreadable files, syntax errors, structural task problems, invalid metadata, duplicate UUIDs, broken reciprocal links, and linked-status mismatches.
- Apply validity checks to the complete reached closure, which is also the exported document set.
- Main validation permits source references only on parent-list entries and parent references only on child-document primary tasks; reciprocal links connect those locations.
- Reject child-reference cycles and multiple owning parents throughout the closure with located diagnostics rather than truncating the tree, duplicating children, or choosing an owner.
- Preserve the existing allowance for missing source files.
- An existing valid directory with no task documents succeeds with `{ "tasks": [] }`; missing directories and file arguments fail.
- Never create task files, insert UUIDs, repair statuses, or launch Pi while dumping.

### Testing

- Use one representative command-level scenario covering authoritative merged child metadata, an uncreated child, an external descendant, ordering, and absence of duplicate roots.
- Keep individual validity-rule coverage in validation tests; dump/command failure tests check validation-before-output flow, file-argument rejection, and absence of partial output.
- Check the empty-directory result.
- Check generated TypeScript for drift and use existing Go test/vet checks.
- Avoid broad parser retesting, Pi extension tests, and additional test infrastructure.

## Current status

- Refinement interview, high-level design, and subtask split approved.
- Store-export subtask implemented: `Store.Dump()` API, export DTOs, merged-tree construction, and full read-only validation in `validate.go`.
- Review follow-up: validation and dump now share `validateAllTasks`, including cached structural/local findings; removed duplicate field validation and `validateFullTasks`.
- Dump-specific `ValidationError`/`validationIssuesToError` live in `dump.go`; validation assertions live in validation tests, not task conversion/query tests.
- Fixed duplicate findings and cycle traversal; reused existing validation tests, with dump failure coverage limited to shared validation-before-export flow.
- Command and TypeScript-generation subtasks remain `new`.

## Tasks

### Add validated merged-tree store export
:::shemiq
type: task
source: ./add-validated-store-export.md
status: done
:::

Define the Go dump DTOs and implement read-only, fully validated merged-tree export through the store, reusing the source/parent validation closure, including reached documents outside the starting directory.

### Expose the JSON dump command
:::shemiq
type: task
source: ./expose-json-dump-command.md
status: new
:::

Add `shemiq dump [dir]` with all-or-nothing JSON output, minimal command-level tests, and CLI usage documentation.

- Store-export handoff: consume `Store.Dump()` and use `task.ValidationError` for located stderr diagnostics; validation and export share the source/parent closure.

### Generate and check TypeScript dump types
:::shemiq
type: task
source: ./generate-typescript-dump-types.md
status: new
:::

Configure pinned Tygo generation, commit the generated TypeScript contract, and add the generation target, drift check, and developer documentation.

- Store-export handoff: select dump DTO/type/constant declarations in `internal/task/dump.go`, preserving literal unions; exclude `ValidationError` and the internal task/parser model.
