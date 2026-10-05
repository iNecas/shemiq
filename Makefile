.DEFAULT_GOAL := help

GO ?= go
BIN := bin/shemiq

.PHONY: help fmt test vet check build install run clean

help:
	@printf 'Targets: fmt, test, vet, check, build, install, run (ARGS="..."), clean\n'

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test vet

build:
	mkdir -p bin
	$(GO) build -o $(BIN) .

install:
	$(GO) install .

run:
	$(GO) run . $(ARGS)

clean:
	rm -f $(BIN)
