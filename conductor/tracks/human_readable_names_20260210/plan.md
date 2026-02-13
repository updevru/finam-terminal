# Implementation Plan - Human-Readable Instrument Names

This plan details the steps to transition the TUI from displaying ticker symbols to human-readable instrument names across all main views.

## Phase 1: Data Models & Infrastructure [checkpoint: 8e146c6]
Implement the necessary data structures and the centralized name cache to support O(1) lookups.

- [x] Task: Add `Name` field to core data models. `822edb2`
    - [x] Add `Name string` to `Position` struct in `models/models.go`.
    - [x] Add `Name string` to `Trade` struct in `models/models.go`.
    - [x] Add `Name string` to `Order` struct in `models/models.go`.
- [x] Task: Implement `InstrumentCache` in `api.Client`. `ef7da9f`
    - [x] Add `instrumentNameCache map[string]string` to `Client` struct in `api/client.go`.
    - [x] Initialize the map in `NewClient`.
    - [x] Implement thread-safe `GetInstrumentName(key string) string` method.
    - [x] Implement thread-safe `UpdateInstrumentCache(ticker, fullSymbol, name string)` method.
- [x] Task: Populate cache during initial load. `7020093`
    - [x] Update `loadAssetCache` in `api/client.go` to populate `instrumentNameCache` with both ticker and full symbol as keys.
- [x] Task: Write unit tests for the cache infrastructure. `7020093`
    - [x] Create/Update tests in `api/client_test.go` to verify cache population and lookup logic.
- [x] Task: Conductor - User Manual Verification 'Data Models & Infrastructure' (Protocol in workflow.md) `8e146c6`

## Phase 2: API Integration & Data Enrichment
Ensure that all data retrieval methods populate the new `Name` fields using the centralized cache.

- [x] Task: Update `GetAccountDetails` to enrich positions with names. `756bad9`
    - [x] In `api/client.go`, populate `Position.Name` using the cache during the response processing.
- [ ] Task: Update `GetTradeHistory` to enrich trades with names.
    - [ ] In `api/client.go`, populate `Trade.Name` using the cache.
- [ ] Task: Update `GetActiveOrders` to enrich orders with names.
    - [ ] In `api/client.go`, populate `Order.Name` using the cache.
- [ ] Task: Update Search logic to ensure cache consistency.
    - [ ] Ensure that search results (which already contain names) keep the cache up-to-date if new instruments are found.
- [ ] Task: Write unit tests for data enrichment.
    - [ ] Verify that `Position`, `Trade`, and `Order` objects returned by the client have their `Name` field populated correctly.
- [ ] Task: Conductor - User Manual Verification 'API Integration & Data Enrichment' (Protocol in workflow.md)

## Phase 3: UI Implementation
Update the terminal interface to display the instrument names and rename relevant columns.

- [ ] Task: Update Positions Table.
    - [ ] In `ui/render.go`, rename "Symbol" header to "Instrument".
    - [ ] Update cell rendering to use `p.Name` with fallback to `p.Ticker`.
- [ ] Task: Update Trade History Table.
    - [ ] In `ui/render.go`, rename "Symbol" header to "Instrument".
    - [ ] Update cell rendering to use `t.Name` with fallback to `t.Symbol`.
- [ ] Task: Update Active Orders Table.
    - [ ] In `ui/render.go`, rename "Symbol" header to "Instrument".
    - [ ] Update cell rendering to use `o.Name` with fallback to `o.Symbol`.
- [ ] Task: Update Confirmation Modals.
    - [ ] Update `showClosePositionModal` in `ui/close_modal.go` to display the instrument name prominently.
    - [ ] Update `OrderEntry` modal in `ui/modal.go` to display the instrument name next to or instead of the ticker.
- [ ] Task: Verify UI changes.
    - [ ] Run the application and manually verify that names are displayed correctly across all tabs and modals.
- [ ] Task: Conductor - User Manual Verification 'UI Implementation' (Protocol in workflow.md)
