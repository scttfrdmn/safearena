.PHONY: build test test-race vet lint fmt check clean install-tools

GOEXPERIMENT := arenas

## build: compile all packages
build:
	GOEXPERIMENT=$(GOEXPERIMENT) go build ./...
	cd cmd/arenacheck && GOEXPERIMENT=$(GOEXPERIMENT) go build .

## test: run all tests
test:
	GOEXPERIMENT=$(GOEXPERIMENT) go test .

## test-race: run tests with race detector
test-race:
	GOEXPERIMENT=$(GOEXPERIMENT) go test -race .

## test-all: run tests for all modules
test-all: test
	cd cmd/arenacheck && GOEXPERIMENT=$(GOEXPERIMENT) go test ./...

## vet: run go vet
vet:
	GOEXPERIMENT=$(GOEXPERIMENT) go vet ./...

## fmt: check formatting (use fmt-fix to apply)
fmt:
	@test -z "$$(gofmt -s -l .)" || (echo "gofmt -s issues:"; gofmt -s -l .; exit 1)

## fmt-fix: apply gofmt -s formatting
fmt-fix:
	gofmt -s -w .

## lint: run staticcheck
lint:
	staticcheck ./...

## check: run all quality checks (fmt, vet, lint, test)
check: fmt vet lint test

## install-tools: install development tools
install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest

## clean: remove build artifacts
clean:
	go clean ./...

help:
	@grep -E '^##' Makefile | sed 's/## //'
