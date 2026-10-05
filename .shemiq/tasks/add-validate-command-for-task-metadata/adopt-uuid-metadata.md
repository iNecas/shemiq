# Generate UUIDs for new tasks and adopt the metadata rules
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: caa20e6d-b1cd-40c0-be4a-d1911538daed
:::

## Context

The parent defines the UUID metadata rules. The completed sibling task implemented `shemiq validate [path] [--fix]` and `internal/task.NewUUID()`. `task new` still renders a top-level directive without a UUID in `internal/task/task.go`; `cmd/new_test.go` compares its document against a fixed template. The Shemiq skill still has `status: todo` examples and does not explain UUIDs or validation. Existing `.shemiq/` documents have already been repaired manually and pass validation; do not migrate them again. In this checkout, the installed `shemiq` binary predates `validate`; use the current source (`go run . validate`) for repository checks until the binary is updated. Keep the sibling implementation task in place.

## Interview

- Use the shared UUID generator directly when the CLI creates a top-level task. For task documents or directives written by hand, agents must not generate or copy UUIDs themselves: leave the field absent, then run `shemiq validate --fix <file>` separately for every touched Markdown file needing repair.
- A directive with `source:` has no `uuid:`; a directive without `source:` needs its own UUID, including a status-only directive. Accepted statuses are `new`, `progress`, and `done`.
- Existing documents are already compliant. Do not perform a migration or change unrelated documents; scope repair commands to touched files. Existing prompts load the skill and `/shemiq-new` calls the CLI, so no prompt changes are needed.

## Implementation plan

- In `internal/task`, have `CreateTopLevelTask` call `NewUUID()` and render its value in the new top-level directive. Return a generation error instead of writing an ID-less document; preserve the rest of `task new` behavior, including both title entry paths, path output, description, slug, and collision handling. Avoid adding a second UUID generator or invoking validation as a post-creation workaround.
- Update `skills/shemiq/SKILL.md` with concise rules for `uuid` versus `source`, allowed statuses, and `shemiq validate [path] [--fix]`. Retain its CLI installation/creation instructions. Specify a manual-edit workflow: omit UUIDs in source-less directives, run `validate --fix <file>` on each touched file, and resolve any remaining validation issues manually. Make examples clearly illustrative pre-repair documents rather than reusable UUID values; replace `status: todo` examples with `status: new`. Do not imply `--fix` repairs anything other than missing UUIDs.
- Adapt the existing `cmd/new_test.go` document assertion to check the generated ID's canonical lowercase UUIDv4 shape without comparing a fixed UUID. Add a focused command-level check that a newly created task passes `validate`; avoid broad new generator or parser test matrices. Run `go test ./...` and check the finished project's validation without rewriting already compliant documents.

## Implementation notes

- `CreateTopLevelTask` obtains a UUID from the shared `NewUUID()` before creating task directories and passes it to the document renderer. Both CLI title paths now write UUID-bearing top-level directives; no post-creation repair is needed.
- Updated the existing document assertions to verify canonical lowercase UUIDv4 shape while retaining the original content and path checks. A new task also passes command-level `validate`.
- The skill now explains UUID/source rules, accepted statuses, validation and manual repair, with pre-repair examples and `new` instead of `todo`. No existing Shemiq documents were migrated.
- Verified with `go test ./...` and read-only `go run . validate` from the repository root; both succeeded.
