# Specification: Security Search Window (S-Key)

## Overview
Implement a dedicated full-width search window to allow users to find securities (stocks, bonds, etc.) and initiate buy orders. This window is triggered by pressing 'S' and provides a real-time search interface with live market data for results.

## Functional Requirements
1.  **Activation**:
    -   Pressing 'S' anywhere in the main application opens the Search Window.
    -   The window must occupy the full width of the terminal.
    -   Closing the window (e.g., via `Esc`) returns focus to the previous view.
2.  **Search Interface**:
    -   **Top Section**: A text input field for searching by Ticker or Security Name.
    -   **Bottom Section**: A table displaying search results.
    -   **Incremental Search**: Results update automatically as the user types, with a ~300ms debounce to optimize API calls.
3.  **Search Results Table**:
    -   **Columns**: Ticker, Full Name, Lot Size, Currency, Last Price, Change %.
    -   **Live Data**: Once results are displayed, the application must fetch snapshots and maintain live market data subscriptions for the visible items in the list.
4.  **Navigation & Interaction**:
    -   `Tab` key toggles focus between the search input field and the results table.
    -   Keyboard arrows (`Up`/`Down`) navigate through the results list when focused.
    -   Pressing 'A' while a result is focused opens the existing "Buy Order" modal, pre-populating it with the selected security.
5.  **Integration**:
    -   Use the Finam API for searching assets.
    -   Integrate with the existing `OrderModal` for trade execution.

## Non-Functional Requirements
-   **Performance**: Search results should be rendered efficiently; live updates must not cause UI flickering or lag.
-   **Reliability**: Handle API timeouts or empty search results gracefully.

## Acceptance Criteria
-   [ ] Pressing 'S' opens a full-width modal with a search input.
-   [ ] Typing in the input updates the list below after a short delay.
-   [ ] Search results include Ticker, Name, Lot, Currency, and Live Price/Change.
-   [ ] 'Tab' switches focus between Input and Table.
-   [ ] Pressing 'A' on a selected result opens the Buy Modal with that ticker.
-   [ ] 'Esc' closes the search window.

## Out of Scope
-   Advanced filtering (e.g., filter by sector, asset class).
-   Persisting search history.
