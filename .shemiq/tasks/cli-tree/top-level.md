# CLI task tree with JSON for integrations
:::shemiq
type: top-level
uuid: 319b7f05-6a7f-4ad8-acf8-cbdd446dbf58
status: refined
:::

## Description

- Provide a human-readable task hierarchy through `shemiq tree [dir]`, with `--json` selecting a machine-readable representation for external integrations.
- Support a future interactive UI inside the Pi coding agent for choosing tasks and actions.
- Reduce duplicated implementation and improve code reuse by keeping the Pi side focused on UI integration; Markdown parsing, discovery, and storage remain the CLI's responsibility.
- Provide a shared Go/TypeScript data contract for the JSON representation.
- This task delivers task metadata and a merged hierarchy, not the Pi UI, Markdown content, or available-action lists.

## Context

- `internal/task/store.go` provides lazy, directory-scoped loading of Markdown task documents and caches parsed snapshots; existing queries do not follow metadata references or merge parent entries with child documents.
- `internal/task/task.go` contains task metadata, embedded parent-list entries, and located structural/local issues; UUIDs and resolved references are currently private.
- The completed store-export and naming subtasks provide `Store.Tree()` and `Tree*` DTOs in `internal/task/tree.go`; the store-export document retains the original naming as implementation history.
- `internal/task/validate.go` provides read-only validation and optional repairs, follows source/parent references, and checks UUID uniqueness, reciprocal links, and linked statuses; the shared tree operation also rejects structural task issues.
- Missing subtask source files are allowed under existing validation rules, supporting subtasks that have not yet been refined into separate documents.
- Parent-list titles can differ from child-document titles without failing existing validation; the tree uses child-document metadata when the file exists.
- `cmd/root.go` registers Cobra commands and separates result output from diagnostics; the tree command follows that pattern.
- `cmd/refine.go` and `cmd/implement.go` retain workflow eligibility and routing; the tree does not expose available actions.
- The project currently has no JSON CLI command or TypeScript generation tooling; command-level tests can reuse the existing command harness, and `Makefile` provides Go test/vet checks.

## Interview

- Original refinement approved a JSON-only `shemiq dump [dir]`, metadata and hierarchy without Markdown or available actions, and a three-subtask split. The naming follow-up below supersedes the command/API names and adds human-readable output.
- Treat the directory as a starting store scope, rather than exporting an individual document; the argument must be a directory, not a Markdown file.
- Require full read-only validation; invalid files prevent successful output, including cross-document inconsistencies and structural problems.
- Use a merged tree, not independent records for parent entries and child files.
- Use existing child-document metadata as authoritative; parent lists determine hierarchy and sibling order, with entry-backed nodes for missing files.
- Store-export refinement supersedes ancestor exclusion: reuse the existing validation closure, following both source and parent references, including ancestors and their referenced sibling branches outside the starting directory.
- Use a typed validation error, reject multiple owning parents, and enforce source/parent placement in main validation.
- Use dedicated Go DTOs as the source of truth and pinned Tygo generation for committed TypeScript types.
- Keep the minimal `{ "tasks": [...] }` payload, optional path/UUID, absolute paths, deterministic ordering, and empty arrays.
- Keep all-or-nothing output, stderr diagnostics, cycle rejection, empty-directory success, and minimal command-level tests.
- Naming follow-up considered `dump`, `export`, `list --json`, `inspect`, `snapshot`, and `tree`; the final choice is `tree`, superseding the preliminary preference for `export`.
- Plain `shemiq tree [dir]` prints a human-readable tree of titles and statuses; `--json` selects the existing machine-readable contract.
- Use `Tree` consistently across the store API, contract types/constants, implementation/test files, and generated TypeScript. Do not add dump/export aliases or compatibility wrappers.
- Both output modes share the same validated model and scope rules. Empty human output reports `No tasks.`; empty JSON output contains an empty `tasks` array.
- Approve the revised four-task split, keeping the completed store-export task and adding a separate naming alignment task before command implementation and type generation.
- Rename the feature directory to `.shemiq/tasks/cli-tree/`; preserve existing UUIDs and completed implementation history.
- Command refinement: use unprefixed human roots, Unicode child connectors, `Title [status]` labels, and two-space JSON indentation with trailing newlines.
- Supersede the original no-color rule: automatically color only status words on terminal result streams (`new` cyan, `refined` yellow, `done` green); disable color for pipes/files, `NO_COLOR`, or `TERM=dumb`. Add no color flag; JSON remains uncolored.
- Place human rendering and reusable color utilities in `internal/console`; retain JSON serialization, diagnostics, and final result writing in `cmd`. Consume the existing DTO without a separate presentation model.
- Keep runtime diagnostics concise without usage text; retain normal Cobra help for argument/flag mistakes and leave other commands unchanged.

## Design

### Architecture and components

- Keep the Cobra command thin: create the store, request the read-only tree, select output mode, prepare the complete result, and write it.
- Put human tree rendering in `internal/console/tree.go` and terminal/environment checks plus reusable ANSI helpers in `internal/console/color.go`; keep JSON serialization and diagnostics in `cmd/tree.go`.
- Use `console.ColorEnabled(output io.Writer) bool` and `console.RenderTree(document task.TreeDocument, color bool) string`; consume the existing DTO directly and leave the interactive console API unchanged.
- Rename the implemented store operation to `Store.Tree() (TreeDocument, error)` in `internal/task/tree.go`; reuse shared validation from `validate.go` before merged-tree construction. The command must not inspect parser internals or assemble relationships itself.
- Keep `ValidationError`, its `Error()` method, and `validationIssuesToError` with the tree operation; semantic failures carry located issues.
- Rename the dedicated DTOs to `TreeDocument` and `TreeNode`, with `TreeTaskType`, `TreeStatus`, and corresponding constants, separate from the internal task/parser structs; no separate DTO package is needed.
- Rename `internal/task/dump_test.go` to `internal/task/tree_test.go` and align dump-specific helpers, comments, and diagnostic wording.
- Do not add Pi UI implementation, action eligibility exports, Markdown exports, runtime TypeScript validation, or a new integration framework.
- Do not add dump/export command aliases, deprecated Go wrappers, or type aliases.

### Command and human-readable output

- Provide `shemiq tree [dir]` and `shemiq tree [dir] --json`; neither mode prompts or launches Pi.
- Human-readable output is the default: print `Title [status]` with unprefixed roots, Unicode child connectors, no synthetic root or blank separators, and a trailing newline.
- Color only status words on terminal result streams: `new` cyan, `refined` yellow, `done` green. Keep brackets, titles, and connectors plain; reset immediately after each status, with no bold or background colors.
- Use the existing `golang.org/x/term` dependency for detection; disable color for non-terminal/unrecognized writers, pipes/files, when `NO_COLOR` is set, or when `TERM=dumb`. Add no color flag.
- Do not add paths, UUIDs, filtering, or interactive controls to human-readable output.
- Use the same validated tree and ordering for both modes; only rendering differs.
- An existing valid directory with no task documents succeeds and reports `No tasks.` in human mode.
- JSON is the only machine-readable format; do not introduce format flags or additional formats.

### Scope and data flow

- The path argument must be a directory.
- Without an argument, discover the nearest project's active `.shemiq/tasks` directory using existing store behavior.
- Recursively load an explicit starting directory using existing Markdown/store rules.
- Perform full validation without repairs, including structural task issues, then construct the merged tree.
- Reuse the existing `reachableTasks()` closure for both validation and tree construction; follow source and parent references without a separate discovery traversal.
- Include every reached document, including ancestors and their referenced sibling branches outside the starting directory.
- Existing child documents supply authoritative metadata; parent lists determine hierarchy and sibling order.
- Missing source targets remain entry-backed nodes, without requiring a child file or UUID.
- Do not duplicate children as roots; reached documents without an owning parent remain roots.
- Sort roots by path and retain parent-list order for children.

### JSON and cross-language contract

- Return one JSON object with a `tasks` array, using two-space indentation and a trailing newline; never include ANSI color escapes.
- Each node contains `type`, `title`, `status`, `projectRoot`, and `subtasks`.
- Include `path` as a resolved absolute document/target path when available; omit it for inline entries without a target.
- Include `uuid` when available; omit it for uncreated source-backed subtasks without a UUID.
- Encode empty task/subtask collections as `[]`, not `null`.
- Do not include Markdown, available actions, validation issues, or a schema-version field.
- Renaming Go/TypeScript identifiers does not change JSON keys, field types, or wire behavior.
- Use Go DTOs and their JSON tags as the source of truth for TypeScript generation, with independent named string types/constants yielding literal unions for task types and statuses.
- Configure Tygo to select the contract declarations in `internal/task/tree.go`, not the internal task/parser model; run a pinned version through `go run` and expose generation through `make generate-types`.
- Commit the generated TypeScript artifact using the same `Tree*` names so consumers do not need to run generation.
- Add a regeneration drift check and generation documentation; no Node/npm tooling or globally installed generator is required.
- Keep DTO types compatible with Tygo rather than accepting unsupported types silently becoming `any`; generation provides compile-time types, not runtime validators.

### Error handling

- Finish loading, validation, tree construction, and rendering/JSON encoding before writing the result.
- Preparation failures return an error/nonzero exit, report diagnostics on stderr, and leave stdout empty. Propagate output-writer failures; already written bytes cannot be rolled back.
- Print validation findings once as `path:line: message`, without a duplicate summary. Runtime failures omit usage text; argument/flag mistakes retain normal Cobra help. Apply this policy only to the tree command.
- Reject unreadable files, syntax errors, structural task problems, invalid metadata, duplicate UUIDs, broken reciprocal links, and linked-status mismatches.
- Apply validity checks to the complete reached closure, which is also the represented document set.
- Main validation permits source references only on parent-list entries and parent references only on child-document primary tasks; reciprocal links connect those locations.
- Reject child-reference cycles and multiple owning parents throughout the closure with located diagnostics rather than truncating the tree, duplicating children, or choosing an owner.
- Preserve the existing allowance for missing source files.
- An existing valid directory with no task documents succeeds with `{ "tasks": [] }` in JSON mode; missing directories and file arguments fail in both modes.
- Never create task files, insert UUIDs, repair statuses, or launch Pi while reading the tree.

### Testing

- Reuse one representative command-level fixture for both output modes, covering authoritative merged child metadata, an uncreated child, an external descendant, ordering, and absence of duplicate roots.
- Check exact plain human layout and final newline plus JSON contract/indentation over the same model; successful commands leave stderr empty.
- Keep individual validity-rule coverage in validation tests; compact command failure tests cover validation-before-output flow, file-argument rejection, missing directories, empty stdout, and concise runtime diagnostics in both modes.
- Check empty-directory behavior in both modes.
- Add focused console tests for status colors and immediate ANSI resets through explicit color selection; check disabling for buffered writers, `NO_COLOR`, and `TERM=dumb`, without pseudo-terminal infrastructure.
- Document tree usage, scope/reference expansion, output modes, automatic colors, and read-only failure behavior in `README.md` and `skills/shemiq/SKILL.md`.
- Reuse existing store tests for the naming-only change; do not duplicate parser or validation-rule tests.
- Check generated TypeScript for drift and use existing Go test/vet checks.
- Avoid broad parser retesting, Pi extension tests, and additional test infrastructure.

## Current status

- Store-export, naming, and command subtasks are complete; TypeScript generation remains `new`.
- `shemiq tree [dir]` implemented in `cmd/tree.go`, registered in `cmd/root.go`.
- Human rendering in `internal/console/tree.go`; color utilities in `internal/console/color.go`.
- JSON serialization uses `json.MarshalIndent` in the command; both modes share the same `Store.Tree()` model.
- Automatic terminal color detection with `NO_COLOR` and `TERM=dumb` support; status words colored cyan/yellow/green.
- Validation errors printed as `path:line: message` on stderr; runtime failures suppress Cobra usage text.
- Tests cover both output modes, empty directories, validation errors, file rejection, and missing directories.
- Console tests cover status colors, ANSI resets, and color disabling for non-terminals.
- Usage documented in `README.md` (step 5) and `skills/shemiq/SKILL.md`.
- All tests pass: `go test ./...` and `go vet ./...`.

## Tasks

### Add validated merged-tree store export
:::shemiq
type: task
source: ./add-validated-store-export.md
status: done
:::

Define the Go contract DTOs and implement read-only, fully validated merged-tree construction through the store, reusing the source/parent validation closure, including reached documents outside the starting directory.

- Completed under the original `Dump*` naming; the naming task supersedes those identifiers without changing behavior or reopening this task.

### Reconsider naming
:::shemiq
type: task
source: ./reconsider-naming.md
status: done
:::

Align the implemented store API, DTOs/types/constants, implementation/test files, helpers, comments, and diagnostics with the approved `tree` naming. Preserve the JSON contract and existing behavior; do not add compatibility aliases.

### Expose the tree command
:::shemiq
type: task
source: ./expose-tree-command.md
status: done
:::

Add `shemiq tree [dir]` with a human-readable title/status tree by default and `--json` selecting the shared JSON contract, all-or-nothing output, minimal command-level tests, and CLI usage documentation.

- Store handoff ready: consume `Store.Tree()` and use `task.ValidationError` for located stderr diagnostics; both modes share the source/parent closure and merged model.
- This replaces the planned JSON-only dump command and includes human rendering; it does not add a separate rendering subtask.

### Generate and check TypeScript tree types
:::shemiq
type: task
source: ./generate-typescript-tree-types.md
status: new
:::

Configure pinned Tygo generation, commit the generated TypeScript contract, and add the generation target, drift check, and developer documentation.

- Store handoff ready: select `Tree*` DTO/type/constant declarations in `internal/task/tree.go`, preserving literal unions; exclude `ValidationError` and the internal task/parser model.
- The generated type names follow the tree naming; the JSON shape and generation approach remain unchanged.
