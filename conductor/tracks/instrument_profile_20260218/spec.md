# Spec: Instrument Profile Screen

## Problem
When a user sees an instrument in the positions table or search results, there's no way to view detailed information about it — financial parameters, trading conditions, price chart. The user has to rely on external tools to check instrument details before trading.

## Solution
Add a full-screen Instrument Profile view that opens when the user presses Enter on a position row or `P` on a search result. The profile displays all available instrument data from the Finam Trade API: asset details, trading parameters, quote data, trading schedule, and a Unicode candlestick price chart.

## Requirements

### Functional Requirements
- **F1**: Pressing Enter on a position row in the Positions table opens the instrument profile
- **F2**: Pressing `P` on a search result in the Search modal opens the instrument profile
- **F3**: The profile displays asset details: name, ticker, ISIN, type, board, currency, lot size, min step, decimals, expiration date
- **F4**: The profile displays the current quote: bid, ask, last, volume, OHLC
- **F5**: The profile displays trading parameters: tradability, long/short availability, risk rates, initial margins
- **F6**: The profile displays the trading schedule (sessions with start/end times)
- **F7**: The profile displays a Unicode candlestick chart with price data
- **F8**: The user can switch chart timeframes using keys `1` (M5), `2` (H1), `3` (D), `4` (W)
- **F9**: The profile opens as a full-screen overlay via `tview.Pages`, hiding the portfolio view. ESC closes and returns to the previous view
- **F10**: Standard keyboard shortcuts work in the profile: `Q` quit, `S` search, `A` open order modal (with Buy/Sell choice, ticker pre-filled), `R` refresh
- **F11**: Profile data auto-refreshes periodically (every 5-10 seconds) while the overlay is open

### Technical Requirements
- **T1**: Use 4 new API methods: `GetAssetInfo`, `GetAssetParams`, `GetBars`, `GetSchedule` via Finam gRPC SDK
- **T2**: All 5 data sources (asset, params, quote, schedule, bars) load in parallel via goroutines
- **T3**: Partial failures are tolerated — show "N/A" for failed sections without blocking others
- **T4**: Chart rendering is a pure function taking bars + dimensions, returning tview-tagged string
- **T5**: New files: `ui/chart.go` (chart renderer), `ui/profile.go` (profile panel component)
- **T6**: The `APIClient` interface must be extended with 4 new methods; mock client updated for tests

## Acceptance Criteria
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./...` passes all tests (existing + new mock methods)
- [ ] Enter on position row opens full-screen profile overlay with all available data
- [ ] Keys 1/2/3/4 switch candlestick chart timeframe
- [ ] `P` key in search modal opens profile for selected instrument
- [ ] ESC closes the profile overlay and returns to the portfolio view
- [ ] Partial API failure shows "N/A" for failed sections, rest loads normally
- [ ] `A` key on profile opens order modal with Buy/Sell choice, ticker pre-filled
- [ ] Profile data auto-refreshes while the overlay is open

## Edge Cases
- Instrument has no bars data (e.g., newly listed) — show "No data" message in chart area
- API returns empty schedule — show "Schedule unavailable"
- Asset params not available for account — show "N/A" for trading parameters
- Very long instrument names — truncate with ellipsis in info panel (42 char width)
- Futures with expiration date vs stocks without — conditional display

## Dependencies
- Finam Trade API SDK (`github.com/FinamWeb/finam-trade-api/go`):
  - `marketdata.MarketDataServiceClient.Bars()` — candle data
  - `assets.AssetsServiceClient.GetAsset()` — instrument details
  - `assets.AssetsServiceClient.GetAssetParams()` — trading parameters
  - `assets.AssetsServiceClient.Schedule()` — trading sessions
- `github.com/rivo/tview` — TUI components (TextView, Flex, Pages)
- Google proto types: `decimal.Decimal`, `interval.Interval`, `date.Date`, `money.Money`, `timestamppb.Timestamp`
