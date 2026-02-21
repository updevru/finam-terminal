# Plan: Instrument Profile Screen

## Overview
Add a full-screen instrument profile overlay with candlestick chart to the TUI terminal app. The profile opens as a full-screen page via `tview.Pages` (not a tab), triggered by Enter on positions or P in search. It displays asset details, trading parameters, quotes, schedule, and a Unicode candlestick chart. Data auto-refreshes while the overlay is open. ESC closes the overlay and returns to the portfolio view.

---

## Phase 1: Data Models & API Layer [checkpoint: 7e91ba7]

- [x] Task: Add profile-related structs to `models/models.go` c37aa94
  - Add `Bar`, `AssetDetails`, `AssetParams`, `TradingSession`, `InstrumentProfile` structs
  - Acceptance: Structs defined, `go build ./...` passes

- [x] Task: Add `GetBars()` method to `api/client.go` 04d4eaf
  - Call `marketDataClient.Bars()` with `BarsRequest{Symbol, Timeframe, Interval}`
  - Parse `decimal.Decimal` fields to float64 via `formatDecimal()`
  - Timeframe param uses SDK `TimeFrame` enum (M5, H1, D, W)
  - Import: `marketdata` package, `interval.Interval`, `timestamppb`
  - Acceptance: Method compiles, returns `[]models.Bar`

- [x] Task: Add `GetAssetInfo()` method to `api/client.go` 0275e0b
  - Call `assetsClient.GetAsset()` with `GetAssetRequest{Symbol, AccountId}`
  - Map response fields: Board, Id, Ticker, Mic, Isin, Type, Name, Decimals, MinStep, LotSize, ExpirationDate, QuoteCurrency
  - Acceptance: Method compiles, returns `*models.AssetDetails`

- [x] Task: Add `GetAssetParams()` method to `api/client.go` b693c2a
  - Call `assetsClient.GetAssetParams()` with `GetAssetParamsRequest{Symbol, AccountId}`
  - Map: IsTradable, Longable, Shortable (enum to human-readable), risk rates, margins
  - Acceptance: Method compiles, returns `*models.AssetParams`

- [x] Task: Add `GetSchedule()` method to `api/client.go` d79a9cc
  - Call `assetsClient.Schedule()` with `ScheduleRequest{Symbol}`
  - Map sessions: type string, start/end times
  - Acceptance: Method compiles, returns `[]models.TradingSession`

---

## Phase 2: Chart Renderer [checkpoint: b1ae822]

- [x] Task: Create `ui/chart.go` with `RenderCandlestickChart()` function 155486c
  - Pure function: `RenderCandlestickChart(bars []models.Bar, width, height int) string`
  - Unicode chars: `█` (body), `│` (wick), `▄▀` (half-blocks for precision)
  - Color tags: `[green]` for bullish (close >= open), `[red]` for bearish
  - Y-axis: price labels (8-char left gutter)
  - X-axis: date/time labels at bottom
  - Auto-scale to fit dimensions; visible candles = `(width - 8) / 2`
  - Empty state: "No data" centered message
  - Acceptance: Function returns valid tview-tagged string, handles empty bars

---

## Phase 3: Profile Panel Component

- [x] Task: Create `ui/profile.go` with `ProfilePanel` struct c920178
  - Fields: `Layout *tview.Flex`, `InfoPanel *tview.TextView` (42 cols fixed), `ChartView *tview.TextView` (flex)
  - `app *tview.Application`, `profile *models.InstrumentProfile`, `timeframe int`
  - `Footer *tview.TextView` for keyboard hint bar
  - Layout: vertical Flex containing horizontal Flex (InfoPanel + ChartView) + Footer
  - Acceptance: Component instantiates without errors

- [x] Task: Implement `NewProfilePanel()`, `Update()`, `UpdateChart()`, `renderInfoPanel()`, `renderChart()` methods c920178
  - `renderInfoPanel()`: formatted sections — Details, Quote, Trading, Schedule — with tview color tags
  - `renderChart()`: calls `RenderCandlestickChart()` using `ChartView.GetInnerRect()` for dimensions
  - `Update()`: full refresh of both panels
  - `UpdateChart()`: chart-only refresh for timeframe switch
  - Timeframe options: M5 (7d), H1 (30d), D (365d), W (5y)
  - Footer: `[1] M5  [2] H1  [3] D  [4] W  | [A] Order  [R] Refresh  [ESC] Back`
  - Acceptance: Panel renders with mock data, all sections display correctly

---

## Phase 4: Integration — Overlay & App State

- [ ] Task: Add profile overlay management to `ui/app.go`
  - Add 4 new methods to `APIClient` interface: `GetBars`, `GetAssetInfo`, `GetAssetParams`, `GetSchedule`
  - Add `ProfilePanel *ProfilePanel` field to `App` struct
  - Add state fields: `profileSymbol string`, `profileTimeframe int` (default: 2 for Daily), `profileOpen bool`
  - Create ProfilePanel in `NewApp()`, add as page "profile" to `app.pages` (existing `tview.Pages`)
  - Add methods:
    - `OpenProfile()` — get symbol from selected positions row, show overlay, trigger load
    - `OpenProfileForSymbol(symbol string)` — open profile for arbitrary symbol (from search)
    - `CloseProfile()` — hide overlay via `pages.SwitchToPage("main")`, stop profile refresh
    - `switchProfileTimeframe(idx int)` — reload only bars for new timeframe
  - Acceptance: Profile overlay shows/hides correctly, `pages` manages main/profile views

- [ ] Task: Add `loadProfileAsync()` to `ui/data.go`
  - Launch 5 parallel goroutines (WaitGroup): GetAssetInfo, GetAssetParams, GetQuotes, GetSchedule, GetBars
  - Each failure logged but doesn't block others (partial data OK)
  - On completion: `QueueUpdateDraw` → update ProfilePanel, set status
  - Acceptance: Profile loads with parallel API calls, partial failures handled

- [ ] Task: Add profile auto-refresh to `ui/data.go`
  - When `profileOpen == true`, include profile data refresh in `backgroundRefresh()` cycle
  - Refresh quote + bars on each cycle; asset info and schedule less frequently (or only on manual R)
  - Acceptance: Profile data stays current while overlay is open

---

## Phase 5: Input Handling & Search Integration

- [ ] Task: Add profile keyboard handlers in `ui/input.go`
  - Set `InputCapture` on `ProfilePanel.Layout` or `ChartView`:
    - `ESC` → `app.CloseProfile()` (return to portfolio)
    - `1/2/3/4` → switch timeframe via `app.switchProfileTimeframe()`
    - `a/A` → `app.OpenOrderModalWithTicker(profileSymbol)` (standard modal with Buy/Sell choice)
    - `r/R` → manual refresh profile data
    - `s/S` → open search modal
    - `q/Q` → quit
  - Add `Enter` key handler on PositionsTable → `app.OpenProfile()`
  - Acceptance: All keyboard shortcuts work on profile overlay

- [ ] Task: Update search modal in `ui/search.go`
  - Add `onViewProfile func(symbol string)` callback to `SearchModal`
  - Update `NewSearchModal()` signature — add `onViewProfile` parameter
  - Add `P` key binding in results table → calls `onViewProfile` with selected symbol
  - Update footer text: add `[P] Profile`
  - Update caller in `app.go` `NewApp()` to pass callback
  - Acceptance: P key in search opens profile for selected instrument

---

## Phase 6: Tests & Polish

- [ ] Task: Update mock client in `ui/mock_client_test.go`
  - Add 4 mock methods: `GetBars`, `GetAssetInfo`, `GetAssetParams`, `GetSchedule`
  - Acceptance: `go test ./...` passes

- [ ] Task: Update existing tests for new signatures
  - Update tests calling `NewSearchModal()` to include `onViewProfile` param
  - Verify no TabbedView changes break existing tests (should be minimal since TabbedView is unchanged)
  - Acceptance: All existing tests pass

- [ ] Task: Update status bar in `ui/render.go`
  - When profile overlay is open, show profile-specific hints:
    `[1-4] Timeframe  [A] Order  [R] Refresh  [ESC] Back`
  - Acceptance: Status bar shows correct hints contextually

---

## Files to Create
- `ui/chart.go` — candlestick chart renderer (~150 lines)
- `ui/profile.go` — ProfilePanel component (~250 lines)

## Files to Modify
- `models/models.go` — add 5 new structs
- `api/client.go` — add 4 new API methods
- `ui/app.go` — extend APIClient interface, add ProfilePanel, overlay methods, page management
- `ui/data.go` — add `loadProfileAsync`, profile auto-refresh in `backgroundRefresh`
- `ui/input.go` — add Enter on positions, profile-specific keys (ESC, 1-4, A, R, S, Q)
- `ui/search.go` — add `onViewProfile` callback, P key, update footer
- `ui/render.go` — update status bar for profile state
- `ui/mock_client_test.go` — add 4 mock methods

## Key Architecture Decisions
1. **Full-screen overlay** (not a tab) — profile shown via `tview.Pages.SwitchToPage("profile")`, closed with ESC
2. **Auto-refresh** — profile data refreshes every 5-10s while overlay is open (quotes + bars); asset info/schedule only on manual R
3. **Order modal** — A key opens standard OrderModal with ticker pre-filled and Buy/Sell direction choice
4. **TabbedView unchanged** — no modifications to existing tab system (Positions/History/Orders stays 3 tabs)

## Summary
- **6 phases**, **14 tasks**
- **2 new files**, **8 files modified**
- Full-screen overlay architecture, not a tab
- Auto-refresh while open; all data loads in parallel; partial failures gracefully handled
- No breaking changes to existing tab navigation
