.PHONY: build test test-integration test-all test-race coverage lint

# Version metadata injected into the binary at build time. VERSION uses
# `git describe` so the value reflects the most recent tag plus a -dirty
# suffix when the working tree has uncommitted changes; COMMIT pins the full
# SHA, BUILD_DATE is the build moment in RFC3339 UTC. All three feed
# `-ldflags -X` against the corresponding vars in the version package.
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X finam-terminal/version.Version=$(VERSION) \
              -X finam-terminal/version.Commit=$(COMMIT) \
              -X finam-terminal/version.BuildDate=$(BUILD_DATE)

## Build the binary with version metadata injected via ldflags
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o finam-terminal main.go

## Run unit tests only (no integration tag)
test:
	go test -v ./...

## Run integration tests only
test-integration:
	go test -tags=integration -v ./api/...

## Run all tests (unit + integration)
test-all:
	go test -v ./...
	go test -tags=integration -v ./api/...

## Run all tests with race detector (requires CGO_ENABLED=1)
test-race:
	CGO_ENABLED=1 go test -race -v ./...
	CGO_ENABLED=1 go test -tags=integration -race -v ./api/...

## Generate combined coverage report
coverage:
	go test -coverprofile=unit-coverage.out ./...
	go test -tags=integration -coverprofile=integration-coverage.out ./api/...
	go install github.com/wadey/gocovmerge@latest
	gocovmerge unit-coverage.out integration-coverage.out > merged-coverage.out
	go tool cover -func=merged-coverage.out
	@rm -f unit-coverage.out integration-coverage.out

## Run linter
lint:
	golangci-lint run ./...
