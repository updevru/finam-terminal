.PHONY: test test-integration test-all test-race coverage lint

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
