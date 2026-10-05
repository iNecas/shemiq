# Development

Requires GNU Make and the Go version specified in `go.mod` (or a Go toolchain that can fetch it). Run these commands from the repository root:

- `make fmt` formats Go files after editing.
- `make check` runs `go test ./...` and `go vet ./...`; use `make test` or `make vet` individually.
- `make build` writes `bin/shemiq`; `make clean` removes the binary.
- `make run ARGS='task new "Describe the task"'` runs the CLI without building a binary and prompts for a title. The CLI searches upward from its working directory for `.shemiq/`; run `bin/shemiq` from a nested directory to exercise that behavior.

Run `make help` for a target list.
