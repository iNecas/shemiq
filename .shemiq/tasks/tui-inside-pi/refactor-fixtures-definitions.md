# Refactor fixtures definitions
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: e02a453b-6fca-4117-aa32-c4abd540e390
:::

## Context

- `extensions/shemiq/controller.test.ts` and `workflows.test.ts` duplicate `makeNode()` and `makeRow()` helpers.
- Express fixture hierarchies directly as nested objects instead of maintaining node relationships, depths, and ancestor arrays separately.
- Provide shared fixture construction for existing tests and future browser coverage; keep scenarios local to the tests that use them.

## Interview

- Choose nested tree definitions with shared defaults over complete `TreeNode` literals or shared versions of the existing separate builders.
- Name the recursive node builder `makeTreeNode()`.
- Export and reuse production `flattenTree()` rather than duplicate row conversion in test helpers.
- Leave `path` and `uuid` absent unless explicitly supplied.
- Infer root and descendant types from their position, allowing explicit overrides.
- Approve the default policy, migration scope, and minimal verification below.

## Implementation plan

### Shared fixture API

- Add `extensions/shemiq/test-fixtures.ts` exporting `makeTreeNode(definition): TreeNode`.
- Accept a typed recursive definition: require `title`, make other node fields optional, and accept simplified nested definitions in `subtasks`.
- Apply these defaults:
  - Root `type`: `"top-level"`; descendant `type`: `"task"`.
  - `status`: `"new"` for each node, independently of its parent's status.
  - Root `projectRoot`: `"/project"`; descendants inherit their parent's project root unless overridden.
  - `subtasks`: an empty array when omitted.
  - `path` and `uuid`: absent when omitted; never derive them from titles or generate placeholder identities.
- Allow explicit field values to override defaults and pass through unchanged.
- Build fresh nodes without mutating the supplied definition.
- Share construction logic, not a catalogue of named scenarios.

Example:

```ts
const tree = makeTreeNode({
  title: "Feature",
  status: "refined",
  path: "/project/.shemiq/tasks/feature/top-level.md",
  subtasks: [{ title: "Build widget" }],
});
```

### Production row conversion and test migration

- Export the existing pure `flattenTree()` from `extensions/shemiq/controller.ts` without changing its behavior.
- Construct fixture trees with `makeTreeNode()` and obtain browser rows through `flattenTree([tree])`, deriving ancestors, depth, and identity through production logic.
- Destructure rows for small scenarios or locate an explicitly supplied identity where relevant; do not add a fixture query language or lookup registry.
- Migrate `extensions/shemiq/controller.test.ts` and `workflows.test.ts`; remove their duplicated `makeNode()` and `makeRow()` helpers.
- Represent parent–child relationships through nested `subtasks` instead of manually supplied ancestor arrays.
- Preserve existing assertions and scenario coverage, including inline subtasks, explicit UUID-based selection, external paths, and quoted titles.
- Adapt paths to the shared project-root default where necessary without weakening expectations.

### Minimal verification and boundaries

- Add one focused helper test covering recursive defaults, explicit overrides, and absent document identity.
- Run it alongside the existing controller and workflow tests; do not introduce a separate fixture-testing framework.
- Do not change runtime eligibility, workflow routing, row-identity rules, or hotkeys.
- Leave broader browser coverage to the browser-test task and test-runner setup, Makefile integration, and consolidated documentation to the verification/documentation task.
- Future tests can reuse the shared fixture helper and exported `flattenTree()`; keep those follow-up tasks in place.

## Implementation notes

- Added `extensions/shemiq/test-fixtures.ts` with typed recursive `TreeNodeDefinition` and `makeTreeNode()`; position-based types, independent statuses, inherited project roots, and explicit overrides follow the plan.
- Builds fresh nodes and arrays without mutating definitions; omitted `path` and `uuid` stay absent.
- Exported production `flattenTree()` unchanged and migrated both test suites to nested definitions; removed duplicated node/row builders and manual ancestor/depth setup.
- Preserved inline subtasks, UUID-based selection, external paths, quoted titles, and all existing assertions.
- Added one focused helper test for recursive defaults, overrides, absent identity, and fresh construction.
- Verification: cached Vitest CLI ran `vitest run extensions/shemiq/test-fixtures.test.ts extensions/shemiq/controller.test.ts extensions/shemiq/workflows.test.ts` — 3 files, 17 tests passed. Strict helper type-check and task metadata validation also passed.
- Browser coverage, runner/Makefile setup, and consolidated documentation remain separate follow-ups; no runtime policy, routing, identity, or hotkey changes.
