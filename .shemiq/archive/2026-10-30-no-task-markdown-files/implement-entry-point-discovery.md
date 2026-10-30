# Implement shared entry-point discovery and reference traversal
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 00ec348b-a9ec-4766-a39c-5bf3ead0bff2
:::

## Context

- Green phase of the test-first split started in
  `reproduce-entry-point-discovery.md`.
- Red tests are already in place and currently failing as intended:
  - `cmd/discovery_test.go` `TestEntryPointDiscoveryIgnoresUnrelatedMarkdown`
    (CLI regression: validate/`--fix`/tree over a scope with unrelated Markdown),
  - `internal/task/store_test.go` `TestStoreEntryPointDiscovery`
    (entry-point-only directory discovery, `TaskByPath` opt-in, transitive
    references, explicit arbitrary file selection).

## Implementation plan

- Add entry-point tracking to the shared store (`internal/task/store.go`):
  - directory scopes recursively discover only files named `top-level.md`;
  - files explicitly supplied to `NewStore` are entry points regardless of name;
  - `load()` completes the transitive `source:`/`parent:` closure so all
    consumers share one task-membership rule set;
  - `TaskByPath` opts a not-yet-cached in-scope file into task handling
    (recorded separately so `TopLevelTasks` stays entry-point based);
  - `TopLevelTasks` returns entry-point tasks with top-level type only.
- Move `reachableTasks`/`taskReferenceTargets` from `validate.go` into the
  store; seeds are entry points plus explicitly opted-in documents.
- `cacheCreated` registers a newly created `top-level.md` as an entry point
  so loaded stores keep observing their own creations.
- Update fixtures/expectations assuming broad Markdown discovery:
  - `internal/task/store_test.go` explicit-scope fixture layout,
  - `internal/task/validate_test.go` and `tree_test.go` primary documents
    renamed to `top-level.md` (references adjusted),
  - `cmd/validate_test.go` scope/duplicates case, `cmd/tree_test.go`
    validation-error case, `cmd/refine_test.go` duplicate candidate fixture.

## Interview

(the interview answers from the top-level apply; no additional questions)

## Implementation notes

- `internal/task/store.go` now owns discovery and traversal:
  - `Store` tracks a single `discovered` set (resolved paths) of
    deliberately selected documents, regardless of their source:
    directory-discovered `top-level.md` entry points, explicitly
    selected scope files, and `TaskByPath` opt-ins.
  - Directory `load()` discovers only files named `top-level.md`
    (`entryPointName`); explicit file scopes load their file as an
    entry point regardless of name.
  - `load()` completes the transitive `source:`/`parent:` closure by
    calling `reachableTasks()`, so validation, tree, and discovery
    share one task-membership rule set.
  - `reachableTasks()`/`taskReferenceTargets()` moved from
    `validate.go`; seeds are the discovered set.
  - `TopLevelTasks()` filters the discovered set by top-level type,
    plus the `top-level.md` name in directory scopes (the explicitly
    selected scope file is exempt), so `TaskByPath` opt-ins with
    arbitrary names never widen candidates; `TaskByPath` loads
    not-yet-cached in-scope files (strict parsing preserved) instead
    of failing with "file not found in store".
  - `cacheCreated` (create.go) adds a newly created `top-level.md`
    to the discovered set so loaded stores keep seeing their own
    creations.
- Preserved behaviors: snapshot caching and failed-refresh
  semantics, query scope boundaries (symlink escape, archive
  exclusion), missing-`source:` allowance vs missing-`parent:`
  findings, referenced-document syntax errors as operation errors,
  and `--fix` repair/refresh flow.
- Test/fixture updates for entry-point rules:
  - `store_test.go` explicit-scope fixture uses `a/top-level.md`,
    `nested/top-level.md`; `a.MARKDOWN` stays as explicit-file
    selection coverage.
  - `validate_test.go` hierarchy table primaries renamed to
    `top-level.md`; the cycle case uses `a|b|c/top-level.md` task
    directories to keep the DFS root order (and the documented
    no-spurious-C→B-cycle intent) stable.
  - `tree_test.go` full-validation fixture renamed `top.md` →
    `selected/top-level.md`.
  - `cmd/validate_test.go`: duplicate-uuid fixture at
    `nested/top-level.md`; `plain.md` now asserted absent from
    diagnostics (ignored Markdown).
  - `cmd/tree_test.go` validation-error fixture renamed to
    `top-level.md`.
  - `cmd/refine_test.go` duplicate-title candidate moved to
    `tasks/other/top-level.md` (was `other.MARKDOWN` in the same
    directory).
- Both red tests from the reproduction task now pass unchanged;
  full Go suite, `go vet`, `make check` (types), and the pi
  extension suite (67 tests incl. e2e) are green.
