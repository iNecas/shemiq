# no-task markdown files
:::shemiq
type: top-level
uuid: 15725362-d1be-487b-a808-849103c1ac14
status: refined
:::

## Description

- Task directories can contain supporting Markdown directly or in nested directories; these files are not necessarily tasks and should not be validated.
- Currently, directory loading treats every Markdown file as a task, causing validation to check unrelated files.
- Preserve the original test-first split: reproduce the issue in tests, then implement the fix.

## Context

- `internal/task/store.go` recursively loads every `.md` / `.markdown` file and converts it into a task, including ordinary notes.
- `internal/task/validate.go` follows references and reports task metadata issues; unrelated files currently receive missing-heading/directive findings.
- `internal/task/tree.go` consumes the same store and validates before building the tree, so unrelated Markdown can also break tree construction.
- Refinement/implementation discovery uses the shared store and must follow the same task-membership rules.
- Existing store, validation, tree, and CLI tests include fixtures or expectations that assume every Markdown file is discovered; update affected tests.

## Interview

- Use naming and explicit selection, not directive presence, to identify initial task documents.
- Recursively discover only `top-level.md` files from directory scopes; ignore unreferenced Markdown even if it contains Shemiq directives.
- Files explicitly supplied to `NewStore`, regardless of filename, are entry points.
- `TaskByPath` explicitly opts a file into task handling within existing scope restrictions.
- Follow all usable `source:` and `parent:` references transitively from entry points.
- Apply these rules consistently across validation, tree, and task discovery.
- Use shared store discovery and reference traversal rather than filtering directory loading while leaving traversal exclusive to validation/tree.
- Keep selected and referenced documents subject to strict existing parsing and validation.
- Keep testing minimal: one representative CLI regression scenario, focused store coverage, and updates to affected existing tests.

## Design

### Architecture and components

- Keep entry-point discovery and reference traversal in the shared task store.
- Update loading and explicit queries in `internal/task/store.go`; make the traversal currently in `internal/task/validate.go` shared store behavior.
- Validation, tree construction, and workflow discovery consume the same task-membership rules.
- Preserve snapshot caching, existing query scope boundaries, reference semantics, and repair behavior; add no configuration or registry.

### Data flow

- Select scope → discover `top-level.md` files or explicitly select a file → load the transitive reference closure → perform the requested operation.
- Referenced files can have arbitrary Markdown filenames.
- Explicit `TaskByPath` queries can extend the selected task set without opting neighboring Markdown into discovery.
- Leave unrelated Markdown outside the task set, including files directly in task directories and in nested directories.

### Error handling

- Parse and validate selected `top-level.md`, explicitly requested files, and existing referenced files as tasks; do not silently skip them when metadata is missing.
- Preserve parsing errors for malformed selected documents and validation findings for metadata or link problems.
- Continue allowing missing `source:` targets for not-yet-created subtask documents; missing `parent:` targets remain errors.
- Never parse unrelated Markdown as tasks or repair those files; `--fix` must leave their bytes unchanged.

### Testing

- Add one representative CLI regression scenario combining valid linked tasks with direct and nested unrelated Markdown, including malformed Shemiq content.
- Exercise validation, `--fix`, and tree output; verify ignored files remain byte-for-byte unchanged.
- Add focused store coverage for arbitrary explicitly selected filenames, `TaskByPath` opt-in, and transitive references while preserving scope and snapshot behavior.
- Update existing fixtures and expectations that depend on broad Markdown discovery.
- Avoid an exhaustive edge-case matrix or a new extension/browser test suite.

## Current status

- Project exploration, interview, high-level design, and task split are complete and approved.
- Reproduction task is done: CLI regression and focused store tests added and
  confirmed red against the current implementation.
- Implementation of shared entry-point discovery and reference traversal is
  done: the store discovers only `top-level.md` entry points from directory
  scopes, treats explicitly selected files as entry points, opts `TaskByPath`
  queries in without widening discovery, and loads the transitive
  `source:`/`parent:` closure for all consumers; affected tests and fixtures
  are aligned and the full suite (Go, vet, types, pi extension incl. e2e) is
  green.

## Tasks

### Reproduce entry-point-based task discovery
:::shemiq
type: task
source: ./reproduce-entry-point-discovery.md
status: done
:::

Add the representative CLI regression and focused store tests for ignored Markdown, explicit file requests, and transitive references.

### Implement shared entry-point discovery and reference traversal
:::shemiq
type: task
source: ./implement-entry-point-discovery.md
status: done
:::

Update shared store loading and its consumers to follow the agreed discovery rules, preserving existing validation, repair, scope, and caching behavior while aligning affected tests.
