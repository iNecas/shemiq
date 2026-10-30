# Reproduce entry-point-based task discovery
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: b616e82c-48c6-452a-8be0-c6992b2d1f20
:::

## Context

- Red phase of the agreed test-first split; the fix itself lands in the
  follow-up task `implement-entry-point-discovery.md`.
- The current store eagerly loads every Markdown file in a directory scope,
  so unrelated notes (plain, valid-directive, or malformed) break validation,
  repair, and tree construction today.

## Implementation plan

- Add a representative CLI regression (`cmd` package) combining valid linked
  `top-level.md` + `child.md` with direct and nested unrelated Markdown
  (plain notes, a valid directive missing uuid, malformed directive content):
  - `validate` and `validate --fix` succeed without findings or repairs;
  - unrelated files stay byte-for-byte unchanged after `--fix`;
  - `tree` renders only the linked hierarchy.
- Add focused store coverage (`internal/task`) for the agreed entry-point rules:
  - directory discovery returns only `top-level.md` entry-point tasks;
  - documents reachable through `source:`/`parent:` references (transitively)
    are queryable via `TaskByPath`;
  - arbitrary explicit file selection and `TaskByPath` opt-in work without
    widening `TopLevelTasks` results;
  - opt-in documents keep strict parsing (issues surface to the caller).
- The new tests intentionally encode the target behavior and are expected to
  fail until the follow-up task implements shared discovery/traversal.

## Implementation notes

- Added `cmd/discovery_test.go` `TestEntryPointDiscoveryIgnoresUnrelatedMarkdown`:
  linked `top-level.md` + `child.md` plus direct/nested unrelated Markdown
  (plain notes, valid directive without uuid, malformed directive).
- Added `internal/task/store_test.go` `TestStoreEntryPointDiscovery`:
  directory discovery of only `top-level.md`, transitive reachability of
  `child.md`/`grandchild.md`, `TaskByPath` opt-in without widening discovery,
  and arbitrary explicit file selection as entry point.
- Confirmed both tests are red for the intended reasons against the current
  implementation:
  - store: unrelated `other/plan.md` is returned by `TopLevelTasks`;
  - CLI: unrelated malformed `notes/broken.md` fails the whole scope load
    (`load scope ...: malformed metadata field`).
- Full suite otherwise green; no existing tests or fixtures were touched.
- Created the follow-up stub `implement-entry-point-discovery.md` pointing at
  the red tests; the fix (shared discovery/traversal) belongs there.
