# Finam Terminal Project

## Project Overview

**finam-terminal** is a Go-based Terminal User Interface (TUI) application designed to interact with the Finam Trade API. It demonstrates how to authenticate, retrieve account information, and fetch market data (quotes, positions) using gRPC.

### Key Technologies
*   **Language:** Go (v1.26)
*   **API Protocol:** gRPC
*   **TUI Library:** `github.com/rivo/tview`
*   **Configuration:** `github.com/joho/godotenv`
*   **SDK:** `github.com/FinamWeb/finam-trade-api/go`
*   **Testing:** `google.golang.org/grpc/test/bufconn` (in-process gRPC for integration tests)

## Architecture

The project follows a clean modular structure:

*   **`main.go`**: The entry point. Handles configuration loading, API client initialization, and starting the UI loop.
*   **`api/`**: Contains the `Client` struct and methods for interacting with the Finam gRPC services. Encapsulates the complexity of the raw API calls.
    *   `client.go`: Core client — `NewClient` creates a TLS connection, `newClientFromConn` initializes service clients (including `corporateActionsClient` for the CorporateActionsService), authenticates, starts the JWT renewal stream, and loads the asset cache. `newClientFromConn` is also used by integration tests to create clients via `bufconn` without TLS. Both `Auth` and `SubscribeJwtRenewal` requests carry `SourceAppId` set to the `sourceAppID` constant (`"finam-terminal"`). After the initial `authenticate()` call (needed for the first JWT and account list via `TokenDetails`), `subscribeJwtRenewal` keeps the token fresh by consuming the `SubscribeJwtRenewal` server stream instead of a timer — it reconnects with exponential backoff (1s, capped at 30s) if the stream drops, and stops silently when the client's context is cancelled via `Close()`. Two API contracts shape the auth path: the connection is built with `grpc.WithDisableServiceConfig()` because Finam publishes no `_grpc_config.<host>` TXT record and the resolver would otherwise stall the first `Auth` call, and `AuthService.TokenDetails` must be called through `getUnauthenticatedContext()` — it rejects requests that also carry the `Authorization` header with `InvalidArgument` (`getContext()` stays the default for every other service). The session token is an opaque `tapi_ak_...` string, not a JWT, so its lifetime is not parsed from the token: `fetchTokenExpiry` reads `TokenDetails.expires_at` after the initial auth and on every renewal-stream token, and `TokenExpiry()` exposes it.
    *   `quote_stream.go`: Realtime quote layer — `quoteToModel` (shared by `GetQuotes` and the stream), `mergeQuote` (snapshot replaces state, increment overwrites only non-nil fields), and the `SubscribeQuote` stream manager (`StartQuoteStream`, `SetQuoteSymbols`, `runQuoteStream`, `normalizeSymbols`, `getStreamContext`).
*   **`api/testserver/`**: In-process mock gRPC server for integration testing (see [Testing](#testing) section).
*   **`ui/`**: Manages the Terminal User Interface.
    *   `app.go`: Main `App` struct, state management, tabbed view (Positions/History/Orders), and lifecycle (Run/Stop). `warmLotSizeAsync` resolves the trade lot off the event loop for all three order-modal open paths (ticker/search, positions, modify), and `applyOrderModalLot`/`applyModifyModalLot` apply it only while the modal still shows the same instrument.
    *   `render.go` / `components.go`: Responsible for drawing UI elements (tables, lists, headers).
    *   `data.go`: Data fetching logic for trades history and active orders. `loadDataAsync` skips quote polling for the account the live stream owns (`shouldSkipQuotePolling`) and writes results via `applyAccountData`, which keeps the cached quote map instead of replacing it with an empty one; `refreshProfileQuoteAndBars` skips only the quote leg while the stream is live (bars keep polling). `loadProfileAsync` fetches `GetAssetInfo` first, then fans out the remaining fetches in parallel; the instrument type gates which corporate-action calendars are loaded (equities: dividends + splits; bonds: bond events; each non-fatal).
    *   `stream.go`: Realtime quote consumption — coalescing inbox (`onStreamQuote`/`flushQuoteInbox`, one queued `QueueUpdateDraw` at a time, quotes dropped after `Stop()`), `onStreamState` (`streamLive` + `[INFO]` log), `shouldSkipQuotePolling` (the live stream owns the active account only), and `computeStreamSymbols`/`recomputeStreamSymbols` (positions of the active account ∪ open profile symbol).
    *   `search.go`: Dedicated search window for finding securities.
    *   `profile.go`: Full-screen instrument profile overlay with asset details, trading parameters, and chart. Renders instrument-type-specific fields (futures: expiration + contract size; options: + strike; bonds: face value + currency) and open interest in the Quote section for derivatives. Renders compact corporate-action calendar sections (equities: Dividends/Splits; bonds: Coupons/Amortization/Offers), each capped at 3 past + 3 future with a `…` overflow hint via the generic `capCalendar` helper. Instrument type is classified by `isEquityDetails`/`isBondDetails`.
    *   `chart.go`: Unicode candlestick chart renderer with smart time labels.
    *   `input.go`: Keyboard input handlers for all views (navigation, shortcuts, order actions).
    *   `modal.go`: Order placement modal with dynamic fields for Market/Limit/Stop/TP/SL+TP order types.
    *   `utils.go`: UI utility functions (number formatting, account ID masking).
    *   `update_prompt.go`: pre-TUI dialog (`NewUpdatePromptApp(current, latest).Run() bool`) shown after the splash and before `RunStartupSteps`. Every non-explicit path (Esc, default focus, a draw failure) resolves to "continue".
    *   `update_flow.go`: `RunUpdateFlow(rel)` — console progress bar in the `RunStartupSteps` style, readable Russian errors including `ManualUpdateCommand()` on `ErrNotWritable`. Returns the executable path to restart.
    *   `update_indicator.go`: `SetUpdateAvailable`/`NotifyUpdateAvailable` (the goroutine-safe variant that marshals onto the event loop and drops notifications after `Stop()`), `LatestVersion`, the `U`-key modal lifecycle, and `ConfirmUpdate`/`UpdateRequested` — the flag `main.go` acts on after `app.Run()` returns, since the process cannot replace itself while tview owns the terminal.
*   **`config/`**: Handles loading environment variables from `.env` or system environment.
*   **`models/`**: Shared data structures used across the application to represent accounts, quotes, positions, trades, and orders. Key fields include `LotSize` and `Name` for instrument metadata. `AssetParams.TradeLotSize` (int64, 2.18.1) carries the broker's trade lot; 0 means the API has no value. `AccountInfo.LoadError` is set when an account fails to load from the broker. `AccountInfo.DailyPnL` holds the daily P&L value. `Order` includes extended fields for stop/limit prices, conditions, validity, and SL/TP quantities, plus `TriggeredOrderID` (the exchange order a stop spawned, 2.17.0). `Trade` carries `AccruedInterest` and `Currency` (bond НКД + price currency, 2.16.0). Corporate-action calendar types `Dividend`, `Split`, and `BondEvent` (flat pre-formatted strings; `BondEvent.Kind` ∈ {Coupon, Amortization, Offer} selects the populated detail group) are surfaced via `InstrumentProfile.Dividends`/`Splits`/`BondEvents`.
*   **`version/`**: Build-time version metadata. Exposes `Version`, `Commit`, and `BuildDate` as **package-level vars** (not consts — the linker can only override vars via `-ldflags -X`). `String()` returns the display string used by the UI header: a release tag verbatim (`v1.2.3`), or a dev build with VCS info (`dev (a1b2c3d)` or `dev (a1b2c3d, dirty)`), falling back through `runtime/debug.ReadBuildInfo()` when no commit is injected. `Info()` returns the raw tuple for diagnostics.
*   **`updater/`**: Self-update machinery, standard library only (no new `go.mod` dependency). Gated entirely on `updater.IsRelease(version.Version)` — a `dev` build performs no request, writes no file and shows nothing.
    *   `semver.go`: `IsRelease`, `Compare`, `IsNewer` — a hand-rolled semver parser (optional `v` prefix, pre-release below its release, build metadata ignored). `IsNewer` returns false unless **both** sides are release versions, so a dev build is never nagged and a locally-newer build is never asked to downgrade.
    *   `state.go`: `State` + `LoadState`/`SaveState` over `~/.finam-cli/update.json` (same directory as the token `.env`, resolved via `config.UserConfigDir()`). Written atomically (temp + rename); a missing, empty or corrupt file degrades to the zero state with a `[WARN]` so a bad cache can never block startup.
    *   `github.go`: `Release`/`Asset` + `FetchLatestRelease` against `/repos/updevru/finam-terminal/releases/latest`. Unauthenticated (60 req/h/IP vs one check per day), 10s timeout, `apiBaseURL` is a package var so tests point it at `httptest`.
    *   `checker.go`: `ShouldCheck(state, now)` (24h window, zero time = due) and `Run(ctx, current, onNewVersion)` — the background loop. A failed check deliberately does **not** advance `LastCheck`.
    *   `asset.go`: `AssetName(goos, goarch)` mirroring the `release.yml` build matrix; platforms outside it (`linux/arm64`, `windows/arm64`) return an error naming the platform.
    *   `download.go`: streaming download through `io.MultiWriter(file, sha256)`, verified against `checksums.txt` (parser handles both the `␠␠` and `␠*` sha256sum separators) with a fallback to the asset size for releases predating it. 5 minute timeout; the partial file is removed on every failure path.
    *   `apply.go`: `SelfUpdate` (resolve exe → `ensureWritable` → download to `.finam-terminal-update-<pid>.tmp` in the exe's own directory → verify → `chmod 0755` on Unix → replace), `replaceExecutable` (`exe→exe.old`, `tmp→exe`, rollback on failure; the backup is removed on Unix and left for `CleanupStaleBackup` on Windows), and `ManualUpdateCommand()`. **Invariant: on any failure the existing binary is byte-for-byte unchanged.**
    *   `restart.go` + `restart_unix.go`/`restart_windows.go`: `Restart(exePath)` — `syscall.Exec` on Unix (same PID and terminal), child process + exit on Windows. Build tags follow `platform/console_*.go`; the exec call sits behind the `execRestart` var so tests assert argv/env without replacing the process.

## Getting Started

### Prerequisites
*   Go 1.26 or higher
*   Finam Trade API Token (obtain from Finam Developer Portal)

### Installation

1.  Clone the repository.
2.  Install dependencies:
    ```bash
    go mod tidy
    ```

### Configuration

The application requires an API token, obtained from [api.finam.ru/tokens/](https://api.finam.ru/tokens/). New tokens use the short `tapi_sk_...` format; old long tokens are accepted too (Legacy) — `authenticate()` passes whatever the user entered as `Secret` unchanged, so no client-side format handling is needed.

1.  Copy the example configuration:
    ```bash
    cp .env.example .env
    ```
2.  Edit `.env` and add your token:
    ```env
    FINAM_API_TOKEN=your_actual_token_here
    ```

### Building and Running

**Run directly:**
```bash
go run main.go
```

**Run with specific account (by index):**
```bash
go run main.go -account 0
```

**Build executable:**
```bash
go build -o finam-trade.exe main.go
./finam-trade.exe
```

**Build with version metadata (recommended for local distribution):**
```bash
make build
```
The `build` target injects `git describe --tags --always --dirty` as `Version`, `git rev-parse HEAD` as `Commit`, and the current UTC time as `BuildDate` via `-ldflags -X` against the `version` package. The resulting binary shows the resolved version in the TUI header.

If you skip `make` and use a plain `go build .` (note the `.`, not `main.go` — `main.go` does not embed `vcs.*` settings), the binary still falls back to `runtime/debug.ReadBuildInfo()` and renders `dev (<short-sha>)` (or with `, dirty` when the working tree has changes).

### Releasing a New Version

To cut a release, just push a `vX.Y.Z` git tag — `.github/workflows/release.yml` is triggered on `push: tags: 'v*'` and will:

1. Build the binary for each `(GOOS, GOARCH)` matrix entry with `-ldflags "-X finam-terminal/version.Version=${{ github.ref_name }} -X finam-terminal/version.Commit=${{ github.sha }} -X finam-terminal/version.BuildDate=<UTC>"` so each artifact reports the tag in the UI header.
2. Upload the artifacts and create a GitHub Release with auto-generated notes.
3. Build and push the Docker image, tagged via `docker/metadata-action`.

**Steps:**
```bash
git tag v1.2.3
git push origin v1.2.3
```

That's it — no manual constant bumps anywhere in source.

## Development Conventions

*   **Style:** Standard Go formatting (`gofmt`).
*   **Logging:** Use standard `log` package with prefixes like `[INFO]` and `[ERROR]`.
*   **UI Updates:** The TUI is event-driven. Ensure UI updates happen on the main thread or using `app.QueueUpdateDraw` (implied by `tview` usage).
*   **Configuration:** Always use `config.Load()` to access settings; do not hardcode credentials.

## Testing

The project has two layers of automated tests: **unit tests** and **integration tests**.

### Running Tests

```bash
# Unit tests only (default, no build tags required)
go test ./...

# Integration tests (against mock gRPC server via bufconn)
go test -tags=integration ./api/...

# All tests together
go test ./... && go test -tags=integration ./api/...

# With race detector (requires CGO_ENABLED=1)
CGO_ENABLED=1 go test -race ./...
CGO_ENABLED=1 go test -tags=integration -race ./api/...
```

A `Makefile` is available with shortcuts: `make test`, `make test-integration`, `make test-all`, `make test-race`, `make coverage`, `make lint`.

### Unit Tests

Unit tests use manual mock structs that implement gRPC service client interfaces (defined in `api/client_test.go`). They test individual methods in isolation without network I/O.

### Integration Tests

Integration tests use build tag `//go:build integration` and are located in `api/client_*_integration_test.go`. They exercise the real `api.Client` lifecycle (connect, authenticate, cache, call methods, close) against an in-process mock gRPC server.

**Mock gRPC Server** (`api/testserver/`):
*   `server.go` — `TestServer` struct: creates a `grpc.Server` + `bufconn.Listener`, registers all 6 mock services, exposes `Start()`, `Stop()`, `Dial()`.
*   `auth_server.go` — `MockAuthServer`: validates tokens, generates JWTs with configurable expiry, tracks call count via `AuthCallCount` and notifies via `AuthCalled` channel. Supports `AuthOverride` for per-call error injection. `TokenDetails` mirrors the real API: it rejects calls carrying an `Authorization` header with `InvalidArgument` and reports `created_at`/`expires_at` (window controlled by `TokenExpiry`, default 1h), so the startup-auth regression is caught end to end.
*   `accounts_server.go` — `MockAccountsServer`: returns configurable positions and trade history per account ID.
*   `marketdata_server.go` — `MockMarketDataServer`: returns quotes and bars. Supports `QuoteOverride` for custom behavior. `SubscribeQuote` is driven by `QuoteStreamQueue` (`QuoteStreamItem{Quotes, StreamErr, Err}`, cap 100) like the JWT renewal mock, and records `QuoteStreamCallCount`, `QuoteStreamCalled`, `LastQuoteStreamSymbols`; `SubscribeQuoteOverride` replaces the default loop.
*   `assets_server.go` — `MockAssetsServer`: returns bulk assets, per-symbol details, trading parameters, and schedule. Supports error injection via `GetAssetError`, `GetAssetParamsError`, `ScheduleError`.
*   `orders_server.go` — `MockOrdersServer`: records `PlaceOrder`, `PlaceSLTPOrder`, `CancelOrder` requests for assertion. Returns configurable active orders.
*   `corporateactions_server.go` — `MockCorporateActionsServer`: implements all 6 CorporateActionsService methods, serving separate past/future fixtures per calendar kind with per-kind error injection (`DividendsError`/`SplitsError`/`BondEventsError`).
*   `testdata.go` — Fixture functions: `MakeJWT()`, `DefaultAssets()`, `DefaultAccountPositions()`, `DefaultQuote()`, `DefaultBars()`, `DefaultOrders()`, `DefaultTrades()`, `DefaultAssetInfo()`, `DefaultAssetParams()`, `DefaultSchedule()`, `DefaultDividends()`, `DefaultSplits()`, `DefaultBondEvents()` (the last three return `(past, future)` and cover all BondEvent oneof branches + nil pointer wrappers), `DefaultStreamQuote(symbol, snapshot)` (full state with `IsDataSnapshot`, or a `Last`+`Timestamp` increment). `DefaultAssetParams()` reports `TradeLotSize: 5` deliberately against `DefaultAssetInfo`'s `LotSize: "10"`, so lot priority is proven end to end.

**Test helper**: `setupTestServer(t)` in `api/client_integration_test.go` creates a `TestServer` + `Client` pair and registers cleanup.

**Integration test files**:
*   `client_integration_test.go` — Client lifecycle, accounts, market data, search, orders (20 tests).
*   `client_cache_integration_test.go` — Asset cache population, lot size on-demand fetch, name lookup (5 tests).
*   `client_token_refresh_integration_test.go` — Auto-refresh before expiry, retry on failure, stop on close (3 tests).
*   `client_errors_integration_test.go` — Unauthenticated, NotFound, ServerUnavailable, DeadlineExceeded, empty response (5 tests).
*   `client_corporate_actions_integration_test.go` — Dividend/split/bond-event calendars: past+future merge, ascending date sort, `IsFuture` flags, oneof mapping (coupon/amortization/offer), nil-safe wrappers (3 tests).
*   `client_quote_stream_integration_test.go` — `SubscribeQuote` manager: snapshot delivery + up only after the first `Recv`, incremental merge, resubscribe on symbol change (no down event), reconnect after a drop, empty set never subscribes, `Close()` stops the manager, in-band `StreamError` keeps the stream (7 tests).

### CI Pipeline

The CI workflow (`.github/workflows/ci.yml`) has 4 jobs:
1.  **unit-test** — runs `go test -race -coverprofile` on all packages.
2.  **integration-test** — runs `go test -tags=integration -race -coverprofile` on `./api/...`.
3.  **coverage** — merges profiles from both jobs and reports via `go tool cover -func`.
4.  **lint** — runs `golangci-lint`.

## Directory Structure

*   `api/`: gRPC client wrapper.
    *   `testserver/`: Mock gRPC server for integration tests (bufconn-based, all 6 Finam services).
*   `updater/`: Update check + self-update (semver, state cache, GitHub client, scheduler, download, apply, restart).
*   `config/`: Configuration loader.
*   `models/`: Data types.
*   `ui/`: TUI implementation (views, controllers).
*   `.env`: Local configuration (git-ignored).

## API Implementation Details

### Retrieving Security Prices

1.  **Market Data (Quotes)**
    *   **Service:** `MarketDataServiceClient`
    *   **Method:** `LastQuote`
    *   **File:** `api/client.go` (`GetQuotes`)
    *   **Key Field:** `Last` (Last trade price)
    *   **Usage:** Ticker lookup, general price checks.

2.  **Security Search**
    *   **Service:** `InstrumentsServiceClient`
    *   **Method:** `GetSecurities`
    *   **File:** `api/client.go` (`SearchSecurities`)
    *   **Usage:** Finding assets by ticker or name.

3.  **Portfolio Positions**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetAccount`
    *   **File:** `api/client.go` (`GetAccountDetails`)
    *   **Key Field:** `CurrentPrice` (Broker's valuation price)
    *   **Usage:** Calculating equity, PnL, and position value. Positions are enriched with `LotSize` and human-readable `Name` from the instrument cache.

4.  **Trade History**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetTradeHistory`
    *   **File:** `api/client.go` (`GetTradeHistory`)
    *   **Usage:** Fetching completed trades for display in the History tab.

5.  **Active Orders**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetOrders`
    *   **File:** `api/client.go` (`GetActiveOrders`)
    *   **Usage:** Fetching pending/active orders for display in the Orders tab.

6.  **Asset Info**
    *   **Service:** `AssetsServiceClient`
    *   **Method:** `GetAsset`
    *   **File:** `api/client.go` (`GetAssetInfo`)
    *   **Usage:** Retrieving detailed instrument information (name, ISIN, type, board, currency, lot size, decimals, expiration).

7.  **Asset Trading Parameters**
    *   **Service:** `AssetsServiceClient`
    *   **Method:** `GetAssetParams`
    *   **File:** `api/client.go` (`GetAssetParams`)
    *   **Usage:** Fetching trading parameters (tradability, long/short availability, risk rates, margins) and `trade_lot_size` (2.18.1, mapped to `models.AssetParams.TradeLotSize`). The call also warms the trade lot cache, so opening an instrument profile primes order sizing for free.

8.  **Candlestick Bars**
    *   **Service:** `MarketDataServiceClient`
    *   **Method:** `Bars`
    *   **File:** `api/client.go` (`GetBars`)
    *   **Usage:** Fetching OHLCV candlestick data for chart rendering. Supports multiple timeframes (M5, H1, D, W).

9.  **Trading Schedule**
    *   **Service:** `AssetsServiceClient`
    *   **Method:** `Schedule`
    *   **File:** `api/client.go` (`GetSchedule`)
    *   **Usage:** Retrieving trading session times for an instrument.

10.  **Instrument Name Cache**
    *   **File:** `api/client.go` (`InstrumentCache`, `GetInstrumentName`, `UpdateInstrumentCache`)
    *   **Usage:** Centralized O(1) cache mapping ticker symbols to human-readable names. Populated during asset loading and search operations.

11.  **Place Order (Market, Limit, Stop, Take-Profit)**
    *   **Service:** `OrdersServiceClient`
    *   **Method:** `PlaceOrder`
    *   **File:** `api/client.go` (`PlaceOrder`)
    *   **Usage:** Places market, limit, stop-loss, and take-profit orders. Accepts optional `*OrderParams` to specify order type and prices. Quantity is in lots (auto-multiplied by lot size). Stop condition is auto-selected based on direction and order type.

12.  **Place SL/TP Linked Order**
    *   **Service:** `OrdersServiceClient`
    *   **Method:** `PlaceSLTPOrder`
    *   **File:** `api/client.go` (`PlaceSLTPOrder`)
    *   **Usage:** Places a linked stop-loss + take-profit order pair where one cancels the other. Supports placing with only SL, only TP, or both. Quantities are in lots. Defaults to GTC (Good Till Cancel) validity.

14.  **Cancel Order**
    *   **Service:** `OrdersServiceClient`
    *   **Method:** `CancelOrder`
    *   **File:** `api/client.go` (`CancelOrder`)
    *   **Usage:** Cancels an active order by account ID and order ID. Returns error if order is already executed or not found.

15.  **gRPC Error Logging**
    *   **File:** `api/client.go` (`logGRPCError`)
    *   **Usage:** Unified helper used by all gRPC calls to log errors in a structured format: `[ERROR] Service.Method failed | Param: value | gRPC code: <code> | Message: <msg> | Endpoint: <addr>`. Never logs secrets (tokens).

16.  **Dividend Calendar** (2.16.0)
    *   **Service:** `CorporateActionsServiceClient`
    *   **Methods:** `GetPastDividends` + `GetFutureDividends`
    *   **File:** `api/client.go` (`GetDividends`)
    *   **Usage:** Returns the merged past (last 12 months, DESC) + future (ASC) dividend calendar for a symbol, limit 20 each, sorted ascending by date with `IsFuture` flags. Surfaced only in the equity profile.

17.  **Split Calendar** (2.16.0)
    *   **Service:** `CorporateActionsServiceClient`
    *   **Methods:** `GetPastSplits` + `GetFutureSplits`
    *   **File:** `api/client.go` (`GetSplits`)
    *   **Usage:** Same past+future windows as dividends; maps ratio (old→new), new lot, and conversion type. Surfaced only in the equity profile.

18.  **Bond Event Calendar** (2.17.0)
    *   **Service:** `CorporateActionsServiceClient`
    *   **Methods:** `GetPastBondsEvents` + `GetFutureBondsEvents`
    *   **File:** `api/client.go` (`GetBondEvents`)
    *   **Usage:** Same past+future windows; flattens the `oneof` event details (`CouponDetails`/`AmortizationDetails`/`OfferDetails`) into a `models.BondEvent` with `Kind` ∈ {Coupon, Amortization, Offer}. Surfaced only in the bond profile. `Future*` requests take no date interval; all SDK pointer/wrapper fields are formatted nil-safe (`formatDate`, `formatDecimalOpt`, `formatInt32Value`).

19.  **Trade Lot Size / Lot Resolution** (2.18.1)
    *   **Service:** `AssetsServiceClient`
    *   **Method:** `GetAssetParams` (field `trade_lot_size`)
    *   **File:** `api/client.go` (`fetchTradeLotSize`, `storeTradeLotSize`, `lotSizeLocked`, `GetLotSize`, `EnsureLotSize`)
    *   **Usage:** The trade lot is the lot the broker sizes orders by and takes priority over `GetAsset.lot_size`. `tradeLotCache` is a second cache tier keyed by ticker and full symbol; a stored `0` is a negative cache entry ("checked, the API has no value") that prevents a `GetAssetParams` call on every refresh tick, while a failed call is deliberately not cached so the next miss retries. `lotSizeLocked` is the single resolution point (trade[ticker] → trade[mic] → asset[ticker] → asset[mic]) and exists because `sync.RWMutex` is not reentrant — `GetAccountDetails` resolves lots while already holding the read lock. `EnsureLotSize` is the blocking warm-up the UI calls off the event loop before rendering the order modal.

20.  **Realtime Quotes (SubscribeQuote)** (2.19.0)
    *   **Service:** `MarketDataServiceClient`
    *   **Method:** `SubscribeQuote` (server stream)
    *   **File:** `api/quote_stream.go` (`StartQuoteStream`, `SetQuoteSymbols`, `runQuoteStream`, `mergeQuote`), `ui/stream.go` (consumption)
    *   **Usage:** Replaces the N+1 `LastQuote` poll for the active account. `SetQuoteSymbols` declares the desired symbol set (normalized: `@`-filtered, deduped, sorted) and never blocks the UI thread; changing it cancels the current subscription and resubscribes without reporting an outage, while a real drop reconnects with 1s→30s backoff. Liveness is claimed only after the first received message (gRPC opens streams lazily), and `resp.Error` is logged as a warning without ending the stream. `getStreamContext` carries the current token with no unary timeout. `Quote.is_data_snapshot` decides the merge: a snapshot replaces the remembered state, an increment overwrites only its non-nil fields. While the stream is live the UI stops polling quotes for the active account and never replaces the cached quote map; inactive accounts and chart bars keep polling.

22.  **Update Check (GitHub Releases)**
    *   **Service:** GitHub REST API (not gRPC)
    *   **Endpoint:** `GET https://api.github.com/repos/updevru/finam-terminal/releases/latest`
    *   **File:** `updater/github.go` (`FetchLatestRelease`), `updater/checker.go` (`Run`, `ShouldCheck`)
    *   **Usage:** Background check once per 24h for release builds only; the result is cached in `~/.finam-cli/update.json`. Unauthenticated, 10s timeout, `[WARN]`-logged on any failure and retried on the normal schedule. `main.go` reads the cache at startup (no network) to decide whether to show the update dialog.

21.  **Session Token Details / Expiry**
    *   **Service:** `AuthServiceClient`
    *   **Method:** `TokenDetails`
    *   **File:** `api/client.go` (`fetchTokenExpiry`, `TokenExpiry`, `GetAccounts`)
    *   **Usage:** Returns the account list and the session token expiry. Must be called **without** the `Authorization` metadata header (use `getUnauthenticatedContext()`) — the token travels in the request body and sending both is rejected with `InvalidArgument: Token is invalid or malformed`.

# Conductor Context

If a user mentions a "plan" or asks about the plan, and they have used the conductor extension in the current session, they are likely referring to the `conductor/tracks.md` file or one of the track plans (`conductor/tracks/<track_id>/plan.md`).

## Universal File Resolution Protocol

**PROTOCOL: How to locate files.**
To find a file (e.g., "**Product Definition**") within a specific context (Project Root or a specific Track):

1.  **Identify Index:** Determine the relevant index file:
    -   **Project Context:** `conductor/index.md`
    -   **Track Context:**
        a. Resolve and read the **Tracks Registry** (via Project Context).
        b. Find the entry for the specific `<track_id>`.
        c. Follow the link provided in the registry to locate the track's folder. The index file is `<track_folder>/index.md`.
        d. **Fallback:** If the track is not yet registered (e.g., during creation) or the link is broken:
            1. Resolve the **Tracks Directory** (via Project Context).
            2. The index file is `<Tracks Directory>/<track_id>/index.md`.

2.  **Check Index:** Read the index file and look for a link with a matching or semantically similar label.

3.  **Resolve Path:** If a link is found, resolve its path **relative to the directory containing the `index.md` file**.
    -   *Example:* If `conductor/index.md` links to `./workflow.md`, the full path is `conductor/workflow.md`.

4.  **Fallback:** If the index file is missing or the link is absent, use the **Default Path** keys below.

5.  **Verify:** You MUST verify the resolved file actually exists on the disk.

**Standard Default Paths (Project):**
- **Product Definition**: `conductor/product.md`
- **Tech Stack**: `conductor/tech-stack.md`
- **Workflow**: `conductor/workflow.md`
- **Product Guidelines**: `conductor/product-guidelines.md`
- **Tracks Registry**: `conductor/tracks.md`
- **Tracks Directory**: `conductor/tracks/`

**Standard Default Paths (Track):**
- **Specification**: `conductor/tracks/<track_id>/spec.md`
- **Implementation Plan**: `conductor/tracks/<track_id>/plan.md`
- **Metadata**: `conductor/tracks/<track_id>/metadata.json`
