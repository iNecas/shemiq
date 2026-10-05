.DEFAULT_GOAL := help

GO ?= go
BIN := bin/shemiq

.PHONY: help fmt test vet check build run clean

help:
	@printf 'Targets: fmt, test, vet, check, build, run (ARGS="..."), clean\n'

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

run:
	$(GO) run . $(ARGS)

clean:
	rm -f $(BIN)
