# Generate and check TypeScript tree types
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 91434fba-071c-4477-9765-c7654b48f2e7
:::

## Context

- The store, naming, and command siblings are complete: `Store.Tree()` and `shemiq tree [dir] --json` provide the validated, merged hierarchy.
- `internal/task/tree.go` currently contains both the integration-facing `Tree*` declarations and tree implementation, including `ValidationError`.
- Generate the TypeScript contract from Go declarations and JSON tags; do not maintain a second handwritten contract.
- The only planned TypeScript consumer is the future Pi extension. Keep its contract at `extensions/shemiq/types.ts`, rather than introducing a shared top-level types directory for hypothetical consumers.
- The repository root is used as a Pi package. A future `extensions/shemiq/index.ts` can import from `./types`; the generated file alone does not make this nested directory an extension entry point.
- `Makefile` currently provides Go test/vet checks, and `DEV.md` documents the developer workflow.
- This task does not implement the Pi extension, create its entry point or package manifest, change JSON/CLI behavior, or reopen completed siblings.

## Interview

- Prefer `extensions/shemiq/types.ts` over a general-purpose contract directory because there is no other consumer yet.
- Add `make check-types` and include it in `make check`; keep standalone Go test/vet targets unchanged.
- Prefer a non-mutating temporary-output comparison over regenerating the working-tree artifact during checking.
- A temporary generation experiment with Tygo `v0.2.21` confirmed correct optional fields, required arrays, and the status union, but exposed two compatibility issues:
  - Tygo selects whole files, so selecting `tree.go` also exports `ValidationError`, which references the excluded `Issue` type.
  - `TreeTaskType` remains `string` because its constants use the `TreeType` prefix instead of `TreeTaskType`.
- Considered small Go-side adjustments, filtering/rewriting generated TypeScript, and a custom declaration-selection wrapper; choose the small Go-side adjustments.
- Move contract declarations to `internal/task/tree_types.go` within package `task`, and rename the two task-type constants to match their type prefix.
- Pin Tygo `v0.2.21`; accept its generated constants and `typeof` unions without custom rewriting.
- Approve compact generation checks and end-to-end workflow verification, without Node/npm tooling or a TypeScript compiler/test framework.
- Document the source of truth, consumer location, commands, and first-run downloads in `DEV.md`.

## Implementation plan

### Contract and generator compatibility

- Move `TreeDocument`, `TreeNode`, `TreeTaskType`, `TreeStatus`, and their constants into `internal/task/tree_types.go`; retain package `task` and existing public type names.
- Keep tree construction, `ValidationError`, and its helpers in `internal/task/tree.go`; do not introduce a separate DTO package.
- Rename `TreeTypeTopLevel` to `TreeTaskTypeTopLevel` and `TreeTypeTask` to `TreeTaskTypeTask`, updating Go references without compatibility aliases.
- Preserve existing Go field types, JSON tags, values, and wire behavior:
  - `TreeDocument` contains a required `tasks: TreeNode[]`.
  - `TreeNode` contains required `type`, `title`, `status`, `projectRoot`, and `subtasks: TreeNode[]`.
  - `path` and `uuid` are optional strings.
  - `TreeTaskType` represents `top-level | task`; `TreeStatus` represents `new | refined | done`.
  - Arrays remain required and non-null in the contract; preserve existing empty-array encoding.
- Add root-level `tygo.yaml`, explicitly including only `tree_types.go` and selecting union generation.
- Register Tygo `v0.2.21` as a tool dependency in `go.mod` and invoke it through `go tool tygo`; no globally installed generator is required.
- Commit generated output at `extensions/shemiq/types.ts` using the existing `Tree*` names.
- Accept Tygo's exported constants and `typeof` union declarations; do not hand-edit or post-process the output into a different contract.
- Generated types are compile-time declarations, not runtime JSON validators.

### Generation and non-mutating drift check

- Add `make generate-types` to generate the working-tree artifact from the shared configuration.
- Add `make check-types` to regenerate into a temporary destination using the same configuration and repository package context, then compare bytes with the working-tree artifact.
- Checking must not overwrite the artifact or depend on a clean Git checkout.
- Missing or stale output fails with a concise instruction to run `make generate-types`; generator failures also fail the target.
- Clean up temporary files, including on failures.
- Configure a recognizable unsupported-type fallback and reject that fallback or `any`, rather than silently publishing a weakened contract.
- Include `check-types` in `make check` alongside existing Go tests and vet; keep `make test` and `make vet` unchanged.
- Add the new targets to `.PHONY` and `make help`.
- The first invocation may download the pinned generator and its dependencies; do not introduce Node/npm tooling or TypeScript compiler setup.

### Verification and documentation

- Keep generation-contract checks compact: verify both named literal unions, required arrays, optional `path`/`uuid`, and exclusion of internal declarations.
- Exercise the actual workflow end to end:
  - Generate the artifact and confirm `make check-types` succeeds.
  - Temporarily make the artifact stale or missing and confirm checking fails without rewriting it.
  - Restore the generated artifact and run the full `make check`.
- Reuse existing Go tests/vet for the declaration move and constant renames; do not duplicate store, parser, validation, or command behavior tests.
- Do not add a TypeScript test framework or Pi extension tests.
- Update `DEV.md` with the Go source-of-truth location, generated output/import location, generation/check commands, and pinned-tool first-run download requirement.
- Keep completed sibling task records intact. The parent notes this generation-compatibility adjustment; their completed behavior and JSON contract remain unchanged.

## Implementation notes

- Moved integration declarations to `internal/task/tree_types.go`; renamed task-type constants to `TreeTaskTypeTopLevel` / `TreeTaskTypeTask`, without aliases or JSON/CLI changes.
- Added root `tygo.yaml`, selecting only `tree_types.go`, literal-union generation, and an explicit unsupported-type fallback marker.
- Generated `extensions/shemiq/types.ts` with Tygo `v0.2.21`; retained generated constants and `typeof` unions without rewriting. No extension entry point or package manifest added.
- Follow-up: moved the generator version from `Makefile` to a Go tool dependency in `go.mod`/`go.sum`; Make invokes `go tool tygo`. Update with `go get -tool github.com/gzuidhof/tygo@<version>`.
- Added shared `make generate-types` / `make check-types` workflow: temporary config overrides only the output destination; both targets reject fallback markers and `any` before publishing/comparing and clean up on failures.
- Included drift checking in `make check`, `.PHONY`, and help; standalone test/vet targets unchanged. Documented source, consumer, commands, and first-run downloads in `DEV.md`.
- Added one compact Go test for generated unions, required fields/arrays, optional path/UUID, and internal-declaration exclusion; no TypeScript tooling or framework added.
- Verified generation/check success, stale/missing artifact failures without rewriting, generator failure, fallback/`any` rejection in both targets, and temporary-file cleanup. Restored all probes; full `make check` passes.
- Completed sibling records remain intact; declaration separation and constant renames are the only Go-side compatibility adjustments.
