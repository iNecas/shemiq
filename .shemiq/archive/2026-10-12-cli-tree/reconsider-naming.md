# Reconsider naming
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: c1ea39b4-ed17-4555-be51-db7637c612f4
:::

## Context

- The parent now describes `shemiq tree [dir]` with human-readable titles/statuses by default and `--json` for integrations.
- The shared validated, merged-tree operation is already implemented as `Store.Dump()` with `Dump*` contract declarations in `internal/task/dump.go` and tests in `internal/task/dump_test.go`.
- Align that existing implementation with the approved tree naming before the command and TypeScript-generation sibling tasks begin.
- This is a naming-only task: preserve directory scope, validation, reference traversal, authoritative child metadata, hierarchy, ordering, read-only behavior, and the JSON wire contract.
- The completed store-export task remains `done`; its original plan and implementation notes record the work under the original names.
- CLI implementation, human rendering, and TypeScript generation belong to the sibling tasks, not this task.

## Interview

- Considered `dump`, `export`, `list --json`, `inspect`, `snapshot`, and `tree`.
- An initial preference for `export` was superseded by `tree`, with a human-readable default and optional JSON output.
- Use `Tree` consistently for the Go API, shared contract types/constants, implementation/test files, and generated TypeScript names.
- Do not add dump/export command aliases, deprecated Go wrappers, or type aliases.
- Keep `ValidationError` unchanged and do not change JSON keys, field types, or behavior.
- Keep naming alignment separate from command implementation and generation; those siblings can proceed independently once the renamed contract is available.

## Implementation plan

### API and contract naming

- Rename `Store.Dump() (DumpDocument, error)` to `Store.Tree() (TreeDocument, error)`.
- Rename the dedicated contract types:
  - `DumpDocument` → `TreeDocument`.
  - `DumpNode` → `TreeNode`.
  - `DumpTaskType` → `TreeTaskType`.
  - `DumpStatus` → `TreeStatus`.
- Rename associated constants consistently:
  - `DumpTypeTopLevel` / `DumpTypeTask` → `TreeTypeTopLevel` / `TreeTypeTask`.
  - `DumpStatusNew` / `DumpStatusRefined` / `DumpStatusDone` → `TreeStatusNew` / `TreeStatusRefined` / `TreeStatusDone`.
- Preserve the independent named string types and explicit values used for TypeScript literal unions.
- Keep `ValidationError`, its `Error()` method, and `validationIssuesToError` unchanged in name and behavior.

### Files and implementation references

- Rename `internal/task/dump.go` → `internal/task/tree.go` and `internal/task/dump_test.go` → `internal/task/tree_test.go`.
- Keep the implementation and contract declarations in package `task`; do not introduce another DTO package.
- Align dump-specific private helpers, test names, comments, and diagnostics with tree terminology, including `buildDumpDocument`, `toDumpType`, and `toDumpStatus`.
- Update callers and test references to the renamed API and types.
- Do not add compatibility wrappers or aliases, output modes, format flags, or new behavior in this naming task.

### Contract preservation and verification

- Preserve `{ "tasks": [...] }` and all existing node keys, field types, optional-field behavior, and empty-array encoding.
- Preserve full read-only validation, typed findings, directory-only scope, the source/parent closure, merging, and deterministic ordering.
- Reuse the existing store and validation tests for verification; no new naming-specific test infrastructure or duplicated behavior coverage is needed.
- Use the existing Go test/vet checks and check the diff for unintended behavior changes.

### Sibling handoff

- The command sibling consumes `Store.Tree()` and the `Tree*` contract to render human-readable output or JSON, using `task.ValidationError` for located stderr diagnostics.
- The generation sibling selects the contract declarations in `internal/task/tree.go`, generates the same `Tree*` names, and excludes `ValidationError` and the internal parser model.
- Keep the completed store-export document as implementation history; its naming-handoff note identifies this task as superseding the original `Dump*` identifiers.

## Implementation notes

- Renamed `internal/task/dump.go` and `dump_test.go` to `tree.go` and `tree_test.go`.
- Replaced `Store.Dump()` and all dedicated DTOs, types, constants, conversion/build helpers, test names, comments, and diagnostics with `Tree` naming; no compatibility aliases or wrappers.
- Preserved JSON keys/tags, named string values, optional fields, empty arrays, scope, validation, reference closure, merging, ordering, and read-only behavior. `ValidationError` and its helper behavior are unchanged.
- Reused existing tests; no new test infrastructure or duplicated coverage.
- Sibling handoff: CLI consumes `Store.Tree()`/`Tree*` and `ValidationError`; TypeScript generation selects the contract declarations in `internal/task/tree.go`, excluding `ValidationError`. Both siblings remain `new`.
- Verification passed: `go test ./...`, `go vet ./...`, task metadata validation, and `git diff --check`; diff review confirmed naming-only code changes.
