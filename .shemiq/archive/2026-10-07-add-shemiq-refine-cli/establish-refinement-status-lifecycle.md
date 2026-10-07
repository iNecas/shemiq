# Establish the refinement status lifecycle
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: c0c3759a-f54c-4a1e-b24d-b36c4d9eaf5f
:::

## Context

`shemiq refine` will route by the top-level status and, for a refined top-level task, by the statuses of its subtask directives in `## Tasks`. Those directives use `source:` to identify subtask files; subtask files use `parent:` to point back. A `source:` file may not exist until the subtask is refined. The parent directive is authoritative for CLI selection. `shemiq new` omits `status:`, which must continue to mean `new`.

The validator currently accepts `new`, `progress`, and `done` in `internal/task/validate.go`; `--fix` currently only inserts missing UUIDs. The refinement prompts in `prompts/shemiq-refine-and-split.md` and `prompts/shemiq-refine-sub-task.md` do not yet record completion. `skills/shemiq/SKILL.md` describes the existing statuses and validation behavior. The later tasks will add CLI selection, Pi launch, and CLI usage documentation.

## Interview

- Accepted statuses become `new`, `refined`, and `done`, with an omitted status equivalent to `new`. Do not migrate archived documents; `progress` becomes invalid rather than an alias.
- On approved subtask refinement, put `status: refined` in **both** the parent-list directive and the subtask file. The prompts write the approved outcome but do not need to interpret the incoming status; the CLI selects eligible tasks beforehand.
- Extend `validate` to check status agreement across linked files. Validating either the parent or child follows existing `source:` and `parent:` references. Compare only reciprocal links; report broken or one-sided links separately rather than guessing a match.
- `validate --fix` resolves status disagreement by promoting the less advanced side according to `new < refined < done`; it does not repair links or use invalid statuses to pick a winner. Preserve omitted statuses when they already agree with explicit `new`.

## Implementation plan

- Update status validation and its focused CLI tests. Keep missing status valid; assert that `refined` is valid and `progress` is invalid. Avoid adding transition enforcement or a status API solely for future CLI routing.
- Enhance validation's document loading to follow existing local `source:` and `parent:` links from a file or directory selection, without duplicate findings or cycles. Missing `source:` targets remain allowed; missing `parent:` targets remain invalid. For an existing target, verify that the child points back to the parent and that the parent's specific task directive points to the child. Report one-sided or incorrect links as validation errors, without rewriting them. Only compare statuses of valid reciprocal task pairs.
- Treat absent statuses as `new` for comparison. Without `--fix`, report mismatches. With `--fix`, promote the less advanced side, adding `status:` if absent and necessary; leave invalid statuses unresolved. Preserve unrelated document content and the existing UUID repair behavior, then recheck the final state so diagnostics reflect remaining issues. A second fix should make no further changes. Note that `--fix` can now change a linked file outside the path explicitly passed to `validate`.
- Update the top-level refinement prompt to set its top-level directive to `refined` only after the user approves and the design and task split are written. Update the subtask refinement prompt to create or update its task file (preserving existing content if present), write reciprocal `parent:` / `source:` metadata, and mark both sides `refined` only after approved refinement. If the subtask file already exists, do not overwrite unrelated content; leave mismatch handling to validation. Neither prompt should mark completion just because Pi exits.
- Update `skills/shemiq/SKILL.md` for the new status vocabulary and linked-file `validate --fix` behavior. Leave `shemiq refine` usage documentation for the Pi-launch task; leave historical archived task documents unchanged.
- Keep tests lean: one parent/child fixture exercised from either validation entry point, promotion and idempotence of `--fix`, a not-yet-created `source:` target, and a one-sided link. Do not launch Pi in tests for this task.

## Implementation notes

Implemented the `new` / `refined` / `done` vocabulary (`status:` omitted means `new`); `progress` now fails validation. Validation follows existing `source:` and `parent:` references in either direction, checks reciprocal task directives and status agreement, and reports broken links without repairing them. Missing `source:` files remain allowed. `--fix` retains UUID insertion and now promotes the lower valid status across linked files, rechecks the result, and is idempotent; invalid statuses are never promoted. Added focused CLI coverage for both entry points, linked-file fixes, missing sources, one-sided links, and repeated fixes.

Updated both Pi refinement prompts to record approved refinement (including both sides of a subtask link) rather than treating session exit as success; updated the Shemiq skill's validation guidance. No Pi launch or CLI routing is included here; those remain in the later tasks.
