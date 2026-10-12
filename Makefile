.DEFAULT_GOAL := help

GO ?= go
BIN := bin/shemiq
TYGO := $(GO) tool tygo
TREE_TYPES := extensions/shemiq/types.ts

.PHONY: help fmt test vet check generate-types check-types build install run clean

help:
	@printf 'Targets: fmt, test, vet, check, generate-types, check-types, build, install, run (ARGS="..."), clean\n'

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test vet check-types

# Tygo has no output override: share the configuration, changing only the
# destination in a temporary copy. Validate before publishing or comparing.
generate-types check-types:
	@set -eu; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	trap 'exit 1' HUP INT TERM; \
	sed 's|^\([[:space:]]*output_path:\).*|\1 "'"$$tmp"'/types.ts"|' tygo.yaml > "$$tmp/tygo.yaml"; \
	$(TYGO) generate --config "$$tmp/tygo.yaml"; \
	if [ ! -s "$$tmp/types.ts" ]; then \
		printf 'Type generation produced no output.\n' >&2; exit 1; \
	fi; \
	if grep -Eq '(^|[^[:alnum:]_])(any|__SHEMIQ_UNSUPPORTED_TYPE__)([^[:alnum:]_]|$$)' "$$tmp/types.ts"; then \
		printf 'Type generation contains unsupported types; fix the Go contract or tygo.yaml.\n' >&2; exit 1; \
	fi; \
	if [ "$@" = generate-types ]; then \
		mkdir -p "$(dir $(TREE_TYPES))"; \
		cp "$$tmp/types.ts" "$(TREE_TYPES)"; \
	elif ! cmp -s "$$tmp/types.ts" "$(TREE_TYPES)"; then \
		printf 'Generated tree types are missing or stale; run make generate-types.\n' >&2; exit 1; \
	fi

build:
	mkdir -p bin
	$(GO) build -o $(BIN) .

install:
	$(GO) install .

run:
	$(GO) run . $(ARGS)

clean:
	rm -f $(BIN)
