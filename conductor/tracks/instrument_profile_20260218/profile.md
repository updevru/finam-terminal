# Instrument Profile Tab — Implementation Plan

## Context

The user wants to add a new "Profile" tab to the TUI application to view detailed instrument information and a candlestick price chart. The Finam Trade API provides all necessary data: `Bars()` for OHLCV candles, `GetAsset()`/`GetAssetParams()` for instrument details and trading parameters, `Schedule()` for trading sessions. Currently none of these are used in the app.

**Approach**: Unicode candlestick chart rendered in `tview.TextView`, new 4th tab in TabbedView, all available data displayed.

---

## Phase 1: Models (`models/models.go`)

Add new structs:

```go
type Bar struct {
    Timestamp time.Time
    Open, High, Low, Close, Volume float64
}

type AssetDetails struct {
    Board, ID, Ticker, MIC, ISIN, Type, Name string
    Decimals       int32
    MinStep        int64
    LotSize        float64
    ExpirationDate string
    QuoteCurrency  string
}

type AssetParams struct {
    IsTradable      bool
    Longable        string  // human-readable status
    Shortable       string
    LongRiskRate    string
    ShortRiskRate   string
    LongInitMargin  string
    ShortInitMargin string
}

type TradingSession struct {
    Type      string
    StartTime time.Time
    EndTime   time.Time
}

type InstrumentProfile struct {
    Symbol   string
    Details  *AssetDetails
    Params   *AssetParams
    Quote    *Quote
    Schedule []TradingSession
    Bars     []Bar
}
```

---

## Phase 2: API Methods (`api/client.go`)

Add 4 new methods following existing patterns (`getContext()`, `formatDecimal()`, error wrapping):

1. **`GetBars(symbol string, timeframe int, from, to time.Time) ([]models.Bar, error)`**
   - Calls `marketDataClient.Bars()` with `BarsRequest{Symbol, Timeframe, Interval}`
   - Parses Decimal fields to float64 via `formatDecimal()`

2. **`GetAssetInfo(symbol, accountID string) (*models.AssetDetails, error)`**
   - Calls `assetsClient.GetAsset()` — returns board, ISIN, type, decimals, min_step, lot, currency, expiration
   - Named `GetAssetInfo` to avoid collision with existing `GetAccountDetails`

3. **`GetAssetParams(symbol, accountID string) (*models.AssetParams, error)`**
   - Calls `assetsClient.GetAssetParams()` — returns tradability, long/short status, risk rates, margins

4. **`GetSchedule(symbol string) ([]models.TradingSession, error)`**
   - Calls `assetsClient.Schedule()` — returns trading session intervals

**Note**: Need to check actual proto imports for `BarsRequest`, `GetAssetParamsRequest`, `ScheduleRequest`. May need new imports from the SDK.

---

## Phase 3: Candlestick Chart Renderer (new file `ui/chart.go`)

Pure function: `RenderCandlestickChart(bars []models.Bar, width, height int) string`

- Takes bars, available width/height in characters
- Returns string with tview color tags (`[green]`, `[red]`, `[white]`)
- Unicode chars: `█` (body), `│` (wick), `▄▀` (half-blocks for precision)
- Green = bullish (close >= open), Red = bearish
- Y-axis: price labels (8-char gutter left)
- X-axis: date/time labels at bottom
- Auto-scales to fit available dimensions
- Visible candles = `(width - 8) / 2` (1 char candle + 1 char gap)
- Graceful empty state: "No data" centered message when bars empty

---

## Phase 4: Profile Panel Component (new file `ui/profile.go`)

```go
type ProfilePanel struct {
    Layout    *tview.Flex       // main container (vertical: content + footer)
    InfoPanel *tview.TextView   // left: instrument info (fixed 42 cols)
    ChartView *tview.TextView   // right: candlestick chart (flex)
    app       *tview.Application
    profile   *models.InstrumentProfile
    timeframe int               // index into timeframeOptions
}
```

**Layout**:
```
┌─ Instrument Info (42w) ─┬─── Chart (flex) ──────────────┐
│ Name: Сбербанк           │                                │
│ Ticker: SBER              │   Unicode candlestick chart    │
│ ISIN: RU0009029540        │   with Y-axis price labels     │
│ Type: Stock / Board: TQBR │                                │
│ Currency: RUB / Lot: 10   │                                │
│ Min Step: 1 / Decimals: 2 │                                │
│───────────────────────────│                                │
│ QUOTE                     │                                │
│ Bid: 290.5  Ask: 290.7    │                                │
│ Last: 290.6  Vol: 12.3M   │                                │
│ O: 289 H: 291 L: 288 C:290│                               │
│───────────────────────────│                                │
│ TRADING                   │                                │
│ Tradable: Yes             │                                │
│ Long: Available (25%)     │                                │
│ Short: HTB (40%)          │                                │
│───────────────────────────│                                │
│ SCHEDULE                  │                                │
│ Main: 10:00-18:45         │                                │
│ Evening: 19:00-23:50      │                                │
└───────────────────────────┴────────────────────────────────┘
 [1] M5  [2] H1  [3] D  [4] W  | [A] Buy  [ESC] Back
```

**Methods**:
- `NewProfilePanel(app *tview.Application) *ProfilePanel`
- `Update(profile *models.InstrumentProfile)` — refreshes both panels
- `UpdateChart(bars []models.Bar, label string)` — refreshes only chart (for timeframe switch)
- `renderInfoPanel()` — formats instrument data with tview color tags
- `renderChart()` — calls `RenderCandlestickChart()`, uses `ChartView.GetInnerRect()` for dimensions

**Timeframe options**:
```go
var timeframeOptions = []struct {
    Label string; Value int; Duration time.Duration
}{
    {"M5", TF_M5, 7 * 24 * time.Hour},
    {"H1", TF_H1, 30 * 24 * time.Hour},
    {"D",  TF_D,  365 * 24 * time.Hour},
    {"W",  TF_W,  5 * 365 * 24 * time.Hour},
}
```

---

## Phase 5: Wire Into TabbedView (`ui/components.go`)

1. Add `TabProfile TabType = iota` (4th constant)
2. Add `ProfilePanel *ProfilePanel` field to `TabbedView`
3. Update `NewTabbedView()` → accept `app *tview.Application`, create `ProfilePanel`, add page `"profile"`
4. Update `NewPortfolioView()` → pass `app` to `NewTabbedView(app)`
5. Update `UpdateHeader()` → tabs: `[" Positions ", " History ", " Orders ", " Profile "]`
6. Update `SetTab()` → add `case TabProfile: tv.Content.SwitchToPage("profile")`

---

## Phase 6: App State & Interface (`ui/app.go`)

1. Extend `APIClient` interface with 4 new methods:
   ```go
   GetBars(symbol string, timeframe int, from, to time.Time) ([]models.Bar, error)
   GetAssetInfo(symbol, accountID string) (*models.AssetDetails, error)
   GetAssetParams(symbol, accountID string) (*models.AssetParams, error)
   GetSchedule(symbol string) ([]models.TradingSession, error)
   ```

2. Add profile state to `App` struct:
   ```go
   profileSymbol    string
   profileTimeframe int  // default 2 (Daily)
   ```

3. Add methods:
   - `OpenProfile()` — gets symbol from selected positions row, switches to Profile tab, triggers load
   - `OpenProfileForSymbol(symbol string)` — opens profile for arbitrary symbol (from search)
   - `switchProfileTimeframe(idx int)` — reloads only bars for new timeframe

4. Update `NewSearchModal` call to pass `onViewProfile` callback

---

## Phase 7: Async Data Loading (`ui/data.go`)

Add `loadProfileAsync(accountID, symbol string, timeframe int)`:
- Launches 5 parallel goroutines (WaitGroup):
  - `GetAssetInfo` → `profile.Details`
  - `GetAssetParams` → `profile.Params`
  - `GetQuotes` → `profile.Quote`
  - `GetSchedule` → `profile.Schedule`
  - `GetBars` → `profile.Bars`
- Each failure is logged but doesn't block others (partial data is OK)
- On completion: `QueueUpdateDraw` → update `ProfilePanel`, set status

---

## Phase 8: Input Handling (`ui/input.go`)

1. Update tab modulo: `% 3` → `% 4` in `nextTab`/`prevTab`
2. Add `case TabProfile:` to `switchToTab` — set focus to `ProfilePanel.ChartView`, show cached data
3. Add `Enter` key handler on PositionsTable → `app.OpenProfile()`
4. Set `InputCapture` on `ProfilePanel.ChartView`:
   - `1/2/3/4` → switch timeframe
   - `a/A` → open order modal for current profile symbol
   - `q/Q` → quit
   - `r/R` → refresh profile
   - `s/S` → open search
   - `←/→` → tab navigation
5. Add `TabProfile` case to global `Tab/BackTab` handler for focus

---

## Phase 9: Search Integration (`ui/search.go`)

1. Add `onViewProfile func(symbol string)` callback to `SearchModal`
2. Update `NewSearchModal` signature — add `onViewProfile` parameter
3. Add `P` key binding in `Table.SetInputCapture` → calls `onViewProfile`
4. Update footer text: add `[P] Profile`
5. Update caller in `app.go` `NewApp()`

---

## Phase 10: Update Mock & Tests

1. `ui/mock_client_test.go` — add 4 mock methods matching new interface
2. Update any tests calling `NewTabbedView()` → pass `nil` app
3. Update any tests calling `NewSearchModal` → add nil `onViewProfile` param

---

## Files to Create
- `ui/chart.go` — candlestick chart renderer (~150 lines)
- `ui/profile.go` — ProfilePanel component (~250 lines)

## Files to Modify
- `models/models.go` — add 5 new structs
- `api/client.go` — add 4 new API methods
- `ui/components.go` — extend TabbedView (TabProfile, ProfilePanel, header, SetTab)
- `ui/app.go` — extend APIClient interface, App state, OpenProfile methods, NewSearchModal call
- `ui/data.go` — add loadProfileAsync
- `ui/input.go` — update tab count, add Enter/timeframe handlers, profile focus
- `ui/search.go` — add onViewProfile callback, P key, update footer
- `ui/render.go` — update status bar for profile shortcuts
- `ui/mock_client_test.go` — add mock methods

---

## Verification

1. `go build ./...` — компиляция без ошибок
2. `go test ./...` — все тесты проходят
3. Manual testing:
   - Run app, select a position, press Enter → Profile tab opens with data
   - Press 1/2/3/4 → chart switches timeframe
   - Press S → search → select instrument → press P → Profile opens
   - Press A on profile tab → order modal opens for current symbol
   - ←/→ switches between all 4 tabs
   - Partial API failures gracefully show "N/A" for failed sections
