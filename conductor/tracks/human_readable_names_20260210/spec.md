# Specification - Human-Readable Instrument Names

## Overview
Currently, the application displays ticker symbols (e.g., `SBER`, `GAZP`) in the Positions, History, and Orders tables. This track aims to improve the user experience by displaying human-readable instrument names (e.g., `Sberbank`, `Gazprom`) instead of just the symbols.

## Functional Requirements
- **Display Upgrade**: Replace the "Symbol" column with an "Instrument" column in the following UI components:
    - Positions Table
    - Trade History Table
    - Active Orders Table
    - Order Confirmation Modals (Buy/Sell/Close)
- **Descriptive Names**: Use the `ShortName` or `Name` field from the Finam API for display.
- **Global Mapping Cache**: Implement a centralized cache (Map/Dictionary) to store `Symbol -> Name` mappings. This cache should be populated whenever instrument data is retrieved (e.g., during search).
- **Fallback Logic**: If a descriptive name is not found in the cache or is empty, fall back to displaying the ticker symbol.
- **Primary Display**: When a name is available, it should replace the symbol entirely in the table cells.

## Non-Functional Requirements
- **Performance**: Accessing the cache should be O(1) to avoid UI lag during table rendering.
- **Consistency**: The same instrument should display the same name across all views.

## Acceptance Criteria
- [ ] The "Symbol" column header is renamed to "Instrument" in Positions, History, and Orders tables.
- [ ] Open positions show full names (e.g., "Sberbank") instead of tickers ("SBER") when the name is known.
- [ ] The "Close Position" confirmation modal displays the instrument's descriptive name.
- [ ] Symbols are still displayed if the human-readable name hasn't been cached yet.

## Out of Scope
- Modifying the underlying data models (the `Symbol` field in gRPC requests should remain unchanged).
- Automatic bulk-fetching of all instrument names at startup (names will be cached as they are discovered or specifically requested for active positions).
