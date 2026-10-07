# Consolidate creation and archive boundaries
:::shemiq
type: task
parent: ./top-level.md
status: done
uuid: 2b3c4a2e-22d4-44eb-88b3-81011e57961d
:::

## Context

Finish the creation/archive migration from the parent refactor. Creation
helpers currently live in `internal/task/create.go`, but `cmd/new.go` still
calls the package-level `CreateTopLevelTask`. Archive orchestration remains in
`internal/task/archive.go`, called through `ArchiveTask`.

The completed store/validation follow-up is the current loading contract:
`NewStore` is side-effect-free, the first query loads the complete read scope,
and cached tasks remain snapshots. The store caches document-level tasks in
`Store.tasks`; `loadTask` is the shared private single-file loader. Creation
must not initialize that scope or parse unrelated tasks.

This task introduces store-owned creation and command-owned filesystem archive
operations. Preserve the existing CLI interfaces and document template. Do not
add workflow features, transactional creation, or broader archive policy.

## Interview

Approved decisions:

- `Store.CreateTopLevel(title, description)` always discovers its destination
  from the store's invocation directory. An explicit scope affects queries and
  validation only; it neither redirects nor prevents creation.
- After successful filesystem creation, an already-loaded store caches the new
  task when its resolved file path is within the configured scope. Creation does
  not rescan existing files or widen that scope.
- Preserve unrestricted description input and do not parse generated documents
  before writing. If post-write cache insertion fails, report the error and
  leave the successfully written file in place.
- Replace the manual slug-building loop with regexp-based cleanup, preserving
  the existing ASCII-only slug behavior.
- Archive remains filesystem-only orchestration in its command. Shared project
  discovery stays a small filesystem helper, not a store-loading operation.
- Reuse existing end-to-end tests and add only focused coverage of the new
  store lifecycle and creation contract.

## Implementation plan

### Store-owned creation

- Implement `(*Store).CreateTopLevel(title, description string) (string, error)`
  in `internal/task/create.go`. Remove the superseded package-level
  `CreateTopLevelTask` entry point once its caller is migrated; do not retain a
  compatibility wrapper or parallel creation API.
- Discover the nearest `.shemiq` above `s.cwd`, independently of `s.selection`.
  If no project exists, creation may establish `.shemiq` under that invocation
  directory. Keep `NewStore`, queries, and validation non-creating.
- Preserve title/description checks, UUID generation, directory collision
  rejection, the existing template and permissions, and cleanup after a failed
  file write. Successful creation returns the absolute `top-level.md` path.
- Replace `taskSlug`'s byte-by-byte loop with a compiled regexp. Collapse runs
  of characters outside ASCII letters/digits into one hyphen, trim leading and
  trailing hyphens, then lowercase the remaining ASCII text. Cleanup before
  lowercasing avoids turning non-ASCII characters into accepted ASCII letters.
  Preserve results such as `Héllo  123` becoming `h-llo-123`, and reject titles
  whose cleaned slug is empty.
- Keep prompting, argument handling, and output in `cmd/new.go`. Construct
  `NewStore(cwd, "")` and call `CreateTopLevel(title, description)` instead of
  performing project discovery in the command.

### Cache lifecycle and failure boundary

- Creation must not call `Store.load` or resolve/load the configured read scope
  as a prerequisite. A malformed unrelated document must not block creation.
- If the store has not loaded successfully, write the document and return its
  path without parsing it. A later query performs normal scope initialization
  and sees the file if selected by that scope.
- If the store is already loaded, resolve the new file's path and check it
  against the existing scope. For an in-scope file, reuse `loadTask` to parse,
  convert, and cache that file only. Subsequent queries include the new task;
  existing cached tasks and previously returned task values remain snapshots.
- For an out-of-scope file, return the created path without adding it to query
  results or parsing it for cache insertion. This includes creation into the
  invocation project while the store reads another project or an archive.
- Descriptions remain verbatim template input. Do not add syntax validation
  before writing or make cache insertion perform semantic validation.
- An in-scope cache insertion can therefore fail after the file is created.
  Return an error identifying that written file, keep it in place, and leave
  existing snapshots intact. Do not roll back successful filesystem creation,
  rescan unrelated tasks, or add a transaction/recovery API. This post-write
  error is distinct from the existing cleanup on a failed file write.

### Command-owned archive

- Move archive orchestration and private selection/directory-checking helpers
  into `cmd/archive.go`. Remove `internal/task/archive.go` and its public
  `ArchiveTask` entry point rather than retaining an adapter.
- Retain the shared `FindProjectDirectory` filesystem helper for store
  operations and archive discovery. It neither loads tasks nor creates a
  project. Archive separately checks that the discovered project exists; avoid
  duplicating the upward-discovery algorithm.
- Archive must not construct a store, parse documents, validate metadata, or
  inspect status. Preserve whole-directory moves and unchanged contents,
  including arbitrary files and directories without a `top-level.md`.
- Preserve relative-path resolution against the invocation directory and the
  requirement that the selected task is an immediate child of that project's
  active `tasks/` directory. Accept the directory itself or its existing
  `top-level.md` file.
- Preserve existing `Lstat` checks for task entries and the tasks/archive
  directories, UTC date prefixes, destination collision rejection, rename
  behavior, and absolute destination output. Do not rewrite references or add
  new containment/symlink policy in this refactor.

### Minimal verification and sibling boundaries

- Reuse `cmd/new_test.go` and `cmd/archive_test.go` as the main regression suite.
  Preserve discovery, prompting, template/UUID checks, input rejection,
  collision/no-overwrite behavior, archive content preservation, missing-project
  rejection, and existing path/symlink checks.
- Add focused store coverage for invocation-based creation despite an explicit
  read scope, in-scope cache insertion after loading, out-of-scope exclusion,
  and the post-write cache parsing error with the created file left in place.
  Reuse fixtures/table cases where practical rather than duplicating CLI tests.
- Strengthen existing fixtures where useful to show malformed unrelated tasks
  do not block creation and malformed selected contents do not block archive.
  Check representative slug cases, including repeated separators, edge
  separators, uppercase letters, and non-ASCII text; avoid a broad test matrix.
- Leave the completed parsing, task-model, and validation tasks intact. The only
  cache lifecycle extension is adding a successfully parsed, newly created
  in-scope task after store-owned creation; external changes still require a
  new store.
- Run `go test ./...`, `go vet ./...`, and `git diff --check` during
  implementation.

## Implementation notes

Store-owned creation: `(*Store).CreateTopLevel(title, description)` implemented
in `internal/task/create.go`. It discovers the project from `s.cwd` independently
of the configured read scope. The package-level `CreateTopLevelTask` was removed;
`cmd/new.go` now constructs `NewStore(cwd, "")` and calls `CreateTopLevel`.

Cache lifecycle: after successful filesystem creation, if the store is already
loaded and the file resolves within scope, `loadTask` caches the new task.
If the store has not loaded, no cache insertion is attempted. Post-write cache
failure returns an error identifying the created file while leaving it in place.

Regexp-based slug: replaced the byte-by-byte loop with `regexp.MustCompile`
`[^a-zA-Z0-9]+` collapsing non-alphanumeric runs into hyphens, trimming edges,
then lowercasing. Preserves existing behavior including `Héllo  123` → `h-llo-123`.

Command-owned archive: moved `ArchiveTask`, `selectArchiveTask`, and
`requireDirectory` from `internal/task/archive.go` into `cmd/archive.go` as
private functions. `internal/task/archive.go` deleted. `FindProjectDirectory`
remains shared in `internal/task/create.go` for both store and archive use.

Tests: existing `cmd/new_test.go` and `cmd/archive_test.go` pass unchanged.
Added `TestStoreCreateTopLevel` in `internal/task/store_test.go` covering
invocation-based creation despite explicit read scope, unloaded-store no-cache,
and in-scope cache insertion after loading. `go test ./...`, `go vet ./...`,
and `git diff --check` pass.
