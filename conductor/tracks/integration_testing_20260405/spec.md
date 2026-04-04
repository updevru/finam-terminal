# Spec: Comprehensive Integration & Unit Testing

## Problem

The project has 31 test files with decent unit test coverage via mock service clients, but lacks true integration tests that validate the full `api.Client` lifecycle (connect, authenticate, cache, call methods, close). Current "integration" tests in `ui/` are actually component-level tests using mocks. When code changes, there's no way to verify the real gRPC client flow works end-to-end without manually connecting to the Finam API.

Additionally, several API methods lack unit test coverage (`GetBars`, `GetAssetInfo`, `GetAssetParams`, `GetSchedule`, `loadAssetCache`, `getFullSymbol`), and the CI pipeline doesn't track coverage or separate unit from integration test runs.

## Solution

Build a comprehensive automated testing infrastructure:

1. **Mock gRPC Server** (`api/testserver/`) — in-process gRPC server via `bufconn` implementing all 5 Finam proto services (Auth, Accounts, MarketData, Assets, Orders) with realistic test data. No network, no real tokens, fully deterministic.

2. **Integration Tests** — test the real `api.Client` struct against the mock server, validating the full lifecycle including authentication, token refresh, caching, all API methods, and error handling.

3. **Expanded Unit Tests** — fill coverage gaps in `api/client_test.go` for untested methods.

4. **Enhanced CI Pipeline** — split unit/integration jobs, add race detection, coverage reporting, and coverage merging.

## Requirements

### Functional
- Mock gRPC server implements all 5 Finam service interfaces with configurable responses
- Integration tests cover all 18 public `api.Client` methods
- Integration tests validate authentication flow (JWT parsing, token refresh, invalid token)
- Integration tests validate cache behavior (MIC cache, lot size, instrument names, security cache)
- Integration tests validate error handling (Unauthenticated, NotFound, Unavailable, DeadlineExceeded)
- Unit tests added for all currently untested methods
- All tests are deterministic (no flaky timing, no network dependencies)

### Technical
- Integration tests use build tag `//go:build integration` (don't run with plain `go test ./...`)
- Mock server uses `google.golang.org/grpc/test/bufconn` for in-process connections
- `api.Client` refactored to expose `newClientFromConn()` for testability without changing public API
- CI pipeline runs unit and integration tests separately with `-race` flag
- Combined test coverage >85% for `api/client.go`

## Acceptance Criteria

- [ ] `go test ./...` runs only unit tests (no integration tag) and all pass
- [ ] `go test -tags=integration ./api/...` runs integration tests against mock server and all pass
- [ ] `go test -tags=integration -race ./api/...` passes with no data races
- [ ] Coverage of `api/client.go` exceeds 85% (combined unit + integration)
- [ ] CI pipeline has separate unit-test, integration-test, coverage, and lint jobs
- [ ] Mock gRPC server is reusable and extensible for future tests
- [ ] No changes to public API (`NewClient` signature unchanged)

## Edge Cases

- Token refresh goroutine: must stop cleanly on `Close()`, handle auth errors with retry
- Cache miss: `getFullSymbol` falls back to API call, then caches
- Zero-quantity positions: must be filtered in `GetAccountDetails`
- gRPC errors: structured logging must work through the real gRPC stack
- Concurrent access: token mutex and asset mutex must be safe under `-race`
- Empty responses: mock returns nil/empty fields, client handles gracefully

## Dependencies

- `google.golang.org/grpc/test/bufconn` (already transitive via `google.golang.org/grpc`)
- No new external dependencies required
