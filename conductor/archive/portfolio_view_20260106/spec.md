# Spec: Portfolio View

## Overview
The goal of this track is to implement the "Portfolio" view in the Finam Trade TUI application. This view will allow users to see their account list, select an account, and view detailed information about that account including its equity, unrealized profit/loss, and a table of current positions.

## Requirements
- **Account Selection:** Users should be able to see a list of available accounts and select one.
- **Account Details:** For the selected account, display:
    - Account ID
    - Type and Status
    - Open Date
    - Total Equity
    - Unrealized Profit/Loss
- **Positions Table:** Display a table of current positions with the following columns:
    - Symbol (Ticker@MIC)
    - Quantity
    - Average Price
    - Current Price
    - Daily PnL
    - Unrealized PnL
- **Navigation:**
    - Use arrow keys to navigate the account list/positions table.
    - Shortcuts to switch between "Portfolio" and other views (like "Quotes").
- **Real-time Updates:** While not strictly required for the first pass, the architecture should support periodic refreshing of data.

## Technical Details
- **API Methods:** Use `GetAccounts` and `GetAccountDetails` from `api/client.go`.
- **UI Library:** Use `tview.Table` for lists and data displays.
- **Data Binding:** Implement a data fetching loop or event-driven update mechanism within `ui/app.go`.
