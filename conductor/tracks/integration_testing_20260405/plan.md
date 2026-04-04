# Plan: Comprehensive Integration & Unit Testing

## Overview

Build a mock gRPC server via `bufconn` implementing all 5 Finam services, write integration tests that exercise the real `api.Client` lifecycle against it, fill unit test coverage gaps, and enhance the CI pipeline with separate jobs and coverage reporting.

---

## Phase 1: Client Refactoring for Testability

- [x] Task 1.1: Extract `newClientFromConn` from `NewClient` <!-- bd3ca3f -->
  - Extract the client initialization logic (service client creation, authenticate, start refresh, load cache) into an unexported `newClientFromConn(conn *grpc.ClientConn, apiToken string) (*Client, error)`
  - Refactor `NewClient` to create TLS connection then delegate to `newClientFromConn`
  - File: `api/client.go`
  - Acceptance: All existing tests pass (`go test ./...`), `NewClient` behavior unchanged

---

## Phase 2: Mock gRPC Server Infrastructure

- [x] Task 2.1: Create `TestServer` core with bufconn <!-- fed2079 -->
  - New file: `api/testserver/server.go`
  - `TestServer` struct with `*grpc.Server`, `*bufconn.Listener`
  - Methods: `NewTestServer()`, `Start()`, `Stop()`, `Dial(ctx) (*grpc.ClientConn, error)`
  - Expose mock service fields: `Auth`, `Accounts`, `MarketData`, `Assets`, `Orders`
  - Acceptance: Can create server, start, dial, stop without errors

- [x] Task 2.2: Implement `MockAuthServer` <!-- fed2079 -->
  - New file: `api/testserver/auth_server.go`
  - Implements `auth.AuthServiceServer` (embeds `UnimplementedAuthServiceServer`)
  - `Auth()`: validates secret against `ValidTokens` map, returns JWT with configurable expiry
  - `TokenDetails()`: returns configured `AccountIDs`
  - Tracks call count for refresh tests
  - Acceptance: Auth and TokenDetails return expected responses

- [x] Task 2.3: Implement `MockAccountsServer` <!-- fed2079 -->
  - New file: `api/testserver/accounts_server.go`
  - Implements `accounts.AccountsServiceServer`
  - `GetAccount()`: returns configurable account with positions
  - `Trades()`: returns configurable trade history
  - Acceptance: Both methods return configured test data

- [x] Task 2.4: Implement `MockMarketDataServer` <!-- fed2079 -->
  - New file: `api/testserver/marketdata_server.go`
  - Implements `marketdata.MarketDataServiceServer`
  - `LastQuote()`: returns configurable quotes by symbol
  - `Bars()`: returns configurable OHLCV data
  - Acceptance: Both methods return configured test data

- [x] Task 2.5: Implement `MockAssetsServer` <!-- fed2079 -->
  - New file: `api/testserver/assets_server.go`
  - Implements `assets.AssetsServiceServer`
  - `Assets()`: returns bulk asset list (populates cache)
  - `GetAsset()`: returns per-symbol asset details
  - `GetAssetParams()`: returns trading parameters
  - `Schedule()`: returns trading sessions
  - Acceptance: All 4 methods return configured test data

- [x] Task 2.6: Implement `MockOrdersServer` <!-- fed2079 -->
  - New file: `api/testserver/orders_server.go`
  - Implements `orders.OrdersServiceServer`
  - `PlaceOrder()`: records request, returns order ID
  - `PlaceSLTPOrder()`: records request, returns order ID
  - `CancelOrder()`: records cancellation
  - `GetOrders()`: returns configurable active orders
  - Acceptance: All methods work, requests recorded for assertion

- [x] Task 2.7: Create test data fixtures <!-- fed2079 -->
  - New file: `api/testserver/testdata.go`
  - `MakeJWT(expiry time.Time) string` — generates valid JWT for tests
  - `DefaultAssets()` — 5-10 instruments (SBER@TQBR, GAZP@TQBR, etc.)
  - `DefaultAccount(id)` — account with positions
  - `DefaultQuote(symbol)` — realistic bid/ask/last
  - `DefaultBars(symbol)` — 20 candlesticks
  - `DefaultOrders(accountID)` — mix of order types and statuses
  - `DefaultTrades(accountID)` — trade history entries
  - Acceptance: All fixture functions return valid proto-compatible data

---

## Phase 3: Integration Tests — Core API Methods

- [x] Task 3.1: Test setup helper and client lifecycle <!-- 1a73d38 -->
  - New file: `api/client_integration_test.go` (build tag: `//go:build integration`)
  - `setupTestServer(t) (*Client, *testserver.TestServer)` helper
  - `TestIntegration_ClientLifecycle` — connect, auth, cache, close
  - `TestIntegration_Auth_InvalidToken` — Unauthenticated error
  - `TestIntegration_Auth_JWTParsing` — expiry parsed from mock JWT
  - Acceptance: All 3 tests pass with `-tags=integration`

- [x] Task 3.2: Account and position tests <!-- 1a73d38 -->
  - `TestIntegration_GetAccounts` — multiple accounts, some with load errors
  - `TestIntegration_GetAccountDetails` — positions with zero-qty filtering, MIC resolution, name enrichment
  - Acceptance: Account data matches mock server configuration

- [x] Task 3.3: Market data tests <!-- 1a73d38 -->
  - `TestIntegration_GetQuotes` — multiple symbols, found/not-found mix
  - `TestIntegration_GetSnapshots` — keyed by ticker (not full symbol)
  - `TestIntegration_GetBars` — OHLCV parsing from proto
  - Acceptance: Quote and bar data matches mock responses

- [x] Task 3.4: Search and asset info tests <!-- 1a73d38 -->
  - `TestIntegration_SearchSecurities` — cache-based partial match by ticker and name
  - `TestIntegration_GetAssetInfo` — basic + future/option/bond oneof details
  - `TestIntegration_GetAssetParams` — longable/shortable/margin formatting
  - `TestIntegration_GetSchedule` — session intervals
  - Acceptance: All 4 tests pass, verify data transformations

- [x] Task 3.5: Trade history and order management tests <!-- 1a73d38 -->
  - `TestIntegration_GetTradeHistory` — side mapping, timestamp conversion
  - `TestIntegration_GetActiveOrders` — status mapping, SL/TP fields
  - `TestIntegration_PlaceOrder_Market` — lot size multiplication verified via recorded request
  - `TestIntegration_PlaceOrder_Limit` — limit price in request
  - `TestIntegration_PlaceOrder_Stop` — stop condition auto-selection
  - `TestIntegration_PlaceSLTPOrder` — linked SL+TP, lot multiplication
  - `TestIntegration_CancelOrder` — cancel by ID
  - `TestIntegration_ClosePosition` — direction inference from quantity sign
  - Acceptance: All 8 tests pass, recorded requests match expectations

---

## Phase 4: Integration Tests — Cache & Token Refresh

- [x] Task 4.1: Cache behavior tests <!-- f559393 -->
  - New file: `api/client_cache_integration_test.go` (build tag: `//go:build integration`)
  - `TestIntegration_AssetCache_PopulatedOnInit` — MIC, names, securities populated
  - `TestIntegration_AssetCache_LotSizeFetchOnDemand` — cache miss triggers API call
  - `TestIntegration_GetLotSize_CacheLookup` — ticker vs full-symbol
  - `TestIntegration_GetInstrumentName_CacheLookup` — by ticker and full symbol
  - `TestIntegration_UpdateInstrumentCache` — manual update
  - Acceptance: All 5 tests pass

- [x] Task 4.2: Token refresh tests <!-- f559393 -->
  - New file: `api/client_token_refresh_integration_test.go` (build tag: `//go:build integration`)
  - `TestIntegration_TokenRefresh_BeforeExpiry` — short-lived JWT, verify second Auth call via counter
  - `TestIntegration_TokenRefresh_RetryOnFailure` — mock error then success
  - `TestIntegration_TokenRefresh_StopsOnClose` — no more Auth calls after Close
  - Use channels/counters in mock auth server (no flaky `time.Sleep`)
  - Acceptance: All 3 tests pass, no race conditions with `-race`

---

## Phase 5: Integration Tests — Error Handling

- [ ] Task 5.1: Error condition tests
  - New file: `api/client_errors_integration_test.go` (build tag: `//go:build integration`)
  - `TestIntegration_Error_UnauthenticatedOnMethod` — auth ok, then method returns Unauthenticated
  - `TestIntegration_Error_NotFound` — GetAsset returns NotFound
  - `TestIntegration_Error_ServerUnavailable` — stop mock server mid-test
  - `TestIntegration_Error_DeadlineExceeded` — mock delays beyond timeout
  - `TestIntegration_Error_EmptyResponse` — nil/empty proto fields handled gracefully
  - Acceptance: All 5 tests pass, errors are meaningful (not panics)

---

## Phase 6: Fill Unit Test Gaps

- [ ] Task 6.1: Add unit tests for untested API methods
  - File: `api/client_test.go` (extend existing)
  - `TestGetBars` — bar parsing with parseDecimalFloat
  - `TestGetAssetInfo` — future/option/bond oneof handling
  - `TestGetAssetParams` — longable/shortable/margin formatting
  - `TestGetSchedule` — session interval parsing
  - `TestLoadAssetCache` — full cache population
  - `TestGetFullSymbol` — cache hit, cache miss with fallback
  - Follow existing mock patterns in the file
  - Acceptance: All new tests pass, coverage of `api/client.go` improves

- [ ] Task 6.2: Fix UI mock client completeness
  - File: `ui/mock_client_test.go`
  - Add function pointer fields for `GetBars`, `GetAssetInfo`, `GetAssetParams`, `GetSchedule`
  - Update methods to use function pointers (like all other methods)
  - Acceptance: Mock client fully matches `APIClient` interface, existing UI tests still pass

---

## Phase 7: CI Pipeline Enhancement

- [ ] Task 7.1: Update CI workflow
  - File: `.github/workflows/ci.yml`
  - Split `test` job into `unit-test` and `integration-test`
  - Update Go version from `1.24` to `1.26`
  - Add `-race` flag to both test jobs
  - Add `-coverprofile` to both jobs
  - Add `coverage` job that merges profiles and reports via `go tool cover -func`
  - Upload coverage artifacts
  - Acceptance: CI pipeline has 4 jobs (unit-test, integration-test, coverage, lint), all green

- [ ] Task 7.2: Add Makefile for local testing convenience
  - New file: `Makefile`
  - Targets: `test` (unit only), `test-integration`, `test-all`, `test-race`, `coverage`, `lint`
  - Acceptance: `make test` and `make test-integration` work locally
