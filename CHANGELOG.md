# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Index Tab**: a fourth tab showing the composition of the MOEX Index (IMOEX) — ticker, Russian name, price, session change (absolute and percent), weight and volume, sorted by weight. `Enter` opens the instrument profile and `A` the standard order modal, both through the existing paths with the correct trade lot. The tab is account-independent (`index_tab`).
- **`AssetsService.GetConstituents`**: `Client.GetIndexConstituents` collects the index composition across the cursor pagination (guarded at 10 pages), caches it per index symbol for 24h, and keeps serving the previous composition when a refetch fails or comes back empty (stale-on-error). An empty response is treated as a failed load and never cached (`index_tab`).
- **`models.Quote.Change`**: the broker's own session change (`quote.change` = last − close) is now mapped into the quote model. It arrives with every `LastQuote` response and every `SubscribeQuote` message, so the Index tab renders Chg and Chg% without a single extra API call (`index_tab`).
- **Bounded quote fallback for the Index tab**: when the realtime stream is down, the composition is fetched in one `LastQuote` batch on tab entry and on `R`; automatic batches are spaced by at least 60s and only run while the tab is open. A `ResourceExhausted` answer turns automatic refresh off for the session, reports it in the status bar and leaves `R` working (`index_tab`).
- **Positions-stream guard**: if the subscription does not come up within 60s of the index composition joining it (or drops three times), the index symbols are excluded for the rest of the session so portfolio quotes recover, and the tab falls back to batches (`index_tab`).
- **User manual**: new page [«Вкладка Индекс»](docs/user_manual/index-tab.md) (`index_tab`).
- **Mock server**: `MockAssetsServer.GetConstituents` with paginated fixtures and per-call error, empty and endless-cursor injection (`index_tab`).

### Changed
- **Tab navigation**: the tab set is now derived from a single `tabLabels` list instead of two hardcoded modulo-3 expressions, so ←/→ cycle all four tabs (`index_tab`).
- **`GetQuotes`**: a rate-limited (`ResourceExhausted`) response now ends the batch and is returned to the caller instead of being logged per symbol and skipped — one refused call must not become dozens (`index_tab`).
- **Closing an instrument profile** returns focus to the tab it was opened from, with the selection intact, instead of always to Positions (`index_tab`).

## [v0.15.0] - 2026-08-25

### Added
- **Auto Update**: the terminal now checks GitHub Releases once a day in the background, shows the available version next to the running one in the header as `⚡ vX.Y.Z`, and offers to download, verify and install it at the next launch before restarting itself. The indicator is informational only — no popups over the trading interface — and the same dialog is available on demand via the `U` hotkey (`auto_update`).
- **`updater` package**: standard-library only — semver comparison, the `~/.finam-cli/update.json` state cache, the GitHub Releases client, the daily scheduler, verified asset download and the atomic binary replacement with rollback (`auto_update`).
- **Release checksums**: `checksums.txt` is now generated and published with every release, and the self-update verifies the downloaded binary against it (falling back to the asset size for older releases) (`auto_update`).
- **`config.UserConfigDir()`**: the `~/.finam-cli` path is now resolved in one place, shared by the token `.env` and the update cache (`auto_update`).

### Changed
- **Header**: renders through the new `headerLabel` helper with dynamic colours; the text is unchanged when no update is available, and gains a `⚡ <version>` segment when one is (`auto_update`).

## [v0.14.0] - 2026-08-25

### Added
- **Realtime Quotes**: quotes for the active account (and an open instrument profile) now arrive over the `SubscribeQuote` gRPC stream instead of a 5-second poll per position; the stream manager reconnects with backoff, resubscribes when the symbol set changes, and merges incremental updates using the 2.19.0 `is_data_snapshot` flag (`trade_lot_and_realtime_quotes`).
- **Quote Polling Fallback**: if the stream drops, the next 5-second tick resumes polling automatically; inactive accounts and chart bars keep polling as before (`trade_lot_and_realtime_quotes`).
- **Trade Lot Size**: `GetAssetParams.trade_lot_size` (2.18.1) is now the primary lot size for order sizing, shown as `Trade Lot` in the profile Trading section and in the order modal label `Lots (size - N)` (`trade_lot_and_realtime_quotes`).

### Changed
- **Finam Trade API SDK** updated to commit `ac0abdd` (2026-08-13), covering releases 2.18.0–2.19.0 (`trade_lot_and_realtime_quotes`).
- **Lot Resolution**: two-tier cache — the trade lot from `GetAssetParams` wins over the asset lot from `GetAsset`, with a negative cache entry when the API reports no trade lot; positions, the order modal, the close modal, `PlaceOrder` and `PlaceSLTPOrder` all read the same resolved value (`trade_lot_and_realtime_quotes`).

## [v0.13.0] - 2026-08-24

### Added
- **Corporate Action Calendars**: Instrument profile now shows dividend and split calendars for equities and coupon/amortization/offer calendars for bonds, via the new `CorporateActionsService` (`GetDividends`/`GetSplits`/`GetBondEvents`); each section is capped at 3 past + 3 future with a `…` overflow hint (`corporate_actions_and_trade_enrichment`).
- **Trade НКД**: History tab shows a combined `НКД` column (accrued interest + currency, e.g. `12.34 RUB`) for bond trades, blank for others (`corporate_actions_and_trade_enrichment`).
- **Order Link Marker**: Orders tab marks a stop order and the exchange order it triggered with a `↳` cross-reference (`corporate_actions_and_trade_enrichment`).

### Changed
- **Finam Trade API SDK** updated to commit `ee013ef` (2026-07-07), past releases 2.15.0–2.17.0 (`sdk_update_new_api_keys`).
- **API Key Format**: onboarding, `.env.example`, and `README.md` now point to `https://api.finam.ru/tokens/` and describe the new short `tapi_sk_...` key format; old long tokens remain supported as Legacy (`sdk_update_new_api_keys`).
- **Token Refresh**: replaced timer-based re-authentication with the `SubscribeJwtRenewal` gRPC stream for automatic JWT renewal, including reconnect with backoff (`sdk_update_new_api_keys`).

### Fixed
- **Startup Authentication**: `AuthService.TokenDetails` is now called without the `Authorization` metadata header — the API rejects calls that carry both the header and the body token, which broke account loading at startup with `InvalidArgument: Token is invalid or malformed`.
- **Slow First Connect**: `grpc.WithDisableServiceConfig()` skips the `_grpc_config.api.finam.ru` TXT lookup that Finam does not publish; its ~11s NXDOMAIN alone consumed the 10s deadline of the first `Auth` call (measured: 12.2s connect vs 0.3s RPC).
- **Token Expiry Source**: session token expiry now comes from `TokenDetails.expires_at` (readable via the new `TokenExpiry()` getter) instead of a JWT parser that always failed on the opaque `tapi_ak_...` token and fell back to an invented 50m lifetime (real: 15m).

## [v0.12.0] - 2026-04-07

### Added
- **Integration Test Suite**: In-process mock gRPC server (`api/testserver/`) covering all 5 Finam services, plus integration tests for client lifecycle, asset cache, token refresh, and gRPC error paths (`integration_testing`).
- **CI Coverage & Race**: CI split into unit-test, integration-test, coverage, and lint jobs with `-race` and merged coverage reporting (`integration_testing`).
- **Extended Instrument Info**: Profile screen renders futures (expiration, contract size), options (+ strike), and bonds (face value, currency); open interest shown in Quote section for derivatives (`extend_instrument_info`).
- **Real Version Display**: New `version/` package; TUI header shows the actual build version via `-ldflags -X`, with `runtime/debug.ReadBuildInfo()` fallback rendering `dev (<sha>[, dirty])` for local builds (`app_version_display`).
- **Makefile**: `make build` injects version metadata; `make test`, `test-integration`, `test-all`, `test-race`, `coverage`, `lint` shortcuts (`integration_testing`, `app_version_display`).

### Changed
- **Go 1.26**: Toolchain bumped to Go 1.26; codebase modernized via `go fix` (`rangeint`, `minmax`, `stringscut`, `any`); dependencies refreshed (`go126_upgrade`).
- **Finam Trade API SDK** updated to 2.14.0 with new derivative/bond fields and open interest (`extend_instrument_info`).
- **Release Workflow**: `.github/workflows/release.yml` injects `Version`/`Commit`/`BuildDate` per matrix artifact via ldflags (`app_version_display`).

## [v0.11.0] - 2026-03-13

### Added
- **Advanced Order Types**: Order modal now supports Limit, Stop-Loss, Take-Profit, and linked SL/TP pair orders with dynamic price fields and auto-selected stop conditions (`stop_loss_take_profit`).
- **Order Management**: Cancel active orders (X/Del) and modify orders (E) directly from the Orders tab with confirmation dialogs (`order_management`).
- **Enhanced Orders Table**: Richer columns showing stop conditions, limit/stop prices, validity, and executed/remaining quantities per order type (`order_management`).
- **Account List Redesign**: Two-row format per account — ID on first row, Equity + daily P&L (color-coded) on second row (`account_list_redesign`).
- **Number Formatting**: Thousand-separator formatting with spaces (Russian locale) for all monetary values (`account_list_redesign`).
- **PlaceSLTPOrder API**: New method for placing linked stop-loss + take-profit order pairs where one cancels the other (`stop_loss_take_profit`).
- **CancelOrder API**: New method for cancelling active orders via gRPC (`order_management`).

### Changed
- **Finam Trade API SDK** updated with `PlaceSLTPOrder` support (`stop_loss_take_profit`).
- **Account List** removed Type column in favor of the two-row Equity/PnL layout (`account_list_redesign`).

### Fixed
- **Order Status Mapping**: All order statuses including SL/TP-specific ones are now correctly mapped (`order_management`).
- **Price Auto-Fill**: Order modal pre-fills price fields with current market price from search results (`stop_loss_take_profit`).

## [v0.10.1] - 2026-03-05

### Added
- **Detailed gRPC Error Logging**: All 16 gRPC API calls now log errors in a unified format including service, method, request parameters, gRPC status code, error message, and endpoint — makes broker support diagnosis significantly easier (`detailed_grpc_logging`).

### Fixed
- **Broker Error Indication**: Accounts that fail to load from the broker now display an error indicator in the UI instead of silently showing stale data (`detailed_grpc_logging`).
- **Portfolio Preservation**: Portfolio data is preserved on transient API errors and Equity/PnL values update in real-time.

## [v0.10.0] - 2026-02-24

### Added
- **Instrument Profile**: Full-screen instrument profile overlay opened via Enter on positions or P on search results, displaying asset details, trading parameters, quotes, trading schedule, and a Unicode candlestick chart with switchable timeframes (M5/H1/D/W) (`instrument_profile`).
- **Candlestick Chart**: Unicode-based price chart with smart time labels on X-axis and support for multiple timeframes (`instrument_profile`).

### Fixed
- **Local Timezone**: All dates in History and Orders tables now display in the user's local timezone instead of UTC (`local_timezone_dates`).
- **Code Formatting**: Fixed `gofmt` formatting across all Go source files.

## [v0.9.0] - 2026-02-13

### Added
- **Portfolio Tabs**: Tabbed interface within the Positions window with History (trade history) and Orders (pending orders) views, switchable via arrow keys and Tab (`portfolio_tabs`).
- **Lot-Based Trading**: Quantities displayed in lots across Positions, History, and Orders tables; lot-based input in Buy/Close modals with real-time cost calculation and lot size display (`lot_based_trading`).
- **Human-Readable Names**: Descriptive instrument names (e.g., "Sberbank" instead of "SBER") displayed across all tables and modal titles, with automatic caching and fallback to ticker symbols (`human_readable_names`).

## [v0.8.1] - 2026-02-04

### Added
- **Security Search**: Dedicated full-width search window for finding assets and initiating orders (`security_search`).

## [v0.8.0] - 2026-01-26

### Added
- **Community Health**: Added `CONTRIBUTING.md` with detailed development guidelines.
- **Community Health**: Added `LICENSE` file (Apache 2.0).
- **Documentation**: Added `CHANGELOG.md` to track project history.
- **Documentation**: Added status badges (CI, Go Report, License, Version) to `README.md`.
- **Documentation**: Added "Development with Gemini and Conductor" section to `README.md`.

## [v0.7.0] - 2026-01-26

### Added
- **Portfolio View**: Comprehensive view of current portfolio holdings (`portfolio_view`).
- **Order Placement**: Ability to place market and limit orders (`order_placement`).
- **Position Closing**: Dedicated modal and logic for closing existing positions (`close_position`).
- **Startup Wizard**: Interactive initial setup and UI for API token configuration (`startup_setup`, `startup_ui`).
- **Token Management**: Proactive token refresh to maintain session validity (`proactive_token_refresh`).
- **UI Layout**: Full-width positions table for better visibility (`full_width_positions_table`).
- **CI/CD**: GitHub Actions pipeline for automated testing and builds (`github_actions_pipeline`).

### Changed
- **UI Responsiveness**: Improved interface adaptation to terminal resizing (`ui_responsiveness`).
- **UX**: Enhanced quantity input handling in order forms (`improve_qty_input`).
- **Filtering**: Automatically filter out positions with zero quantity (`filter_zero_positions`).
