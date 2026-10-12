# Development

Requires GNU Make and the Go version specified in `go.mod` (or a Go toolchain that can fetch it). Run these commands from the repository root:

- `make fmt` formats Go files after editing.
- `make check` runs `go test ./...`, `go vet ./...`, and `make check-types`; use `make test` or `make vet` individually.
- `make build` writes `bin/shemiq`; `make clean` removes the binary.
- `make run ARGS='new "Describe the task"'` runs the CLI without building a binary and prompts for a title. The CLI searches upward from its working directory for `.shemiq/`; run `bin/shemiq` from a nested directory to exercise that behavior.

## Generated TypeScript tree contract

- `extensions/shemiq/types.ts` provides generated tree types for the future Pi extension.
- Regenerate with `make generate-types`; commit the updated output rather than editing it by hand.

Run `make help` for a target list.
