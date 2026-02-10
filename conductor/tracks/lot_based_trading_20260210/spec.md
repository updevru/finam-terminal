# Specification: Lot-Based Trading and Display

## Overview
This feature transforms the application's units of measurement from individual shares to lots. This aligns the TUI with professional trading terminals where "Quantity" typically refers to the number of lots, preventing errors when trading instruments with large multipliers (e.g., 1 lot = 1000 shares).

## Functional Requirements

### 1. Data Layer Enhancements
- **Instrument Metadata**: Update the API client to consistently retrieve and cache the `LotSize` for all instruments.
- **Position Enrichment**: Ensure each position in the portfolio is associated with its instrument's lot size.

### 2. UI - Search & Discovery
- **Search Table**: Add a "Lot" column to the `SearchModal` results table.
- **Dynamic Updates**: Ensure the lot size is displayed alongside the price and change percentage.

### 3. UI - Portfolio & History
- **Positions Table**: Rename the "Qty" column to "Qty (Lots)". Display the quantity calculated as `Total Shares / Lot Size`.
- **Active Orders & History**: Update "Quantity" columns to display lot counts.

### 4. UI - Trading Modals (Buy/Close)
- **Input Logic**: The "Quantity" input field will now represent the number of **Lots**.
- **Real-time Feedback**: 
    - Display the lot size multiplier (e.g., "1 lot = 10").
    - Dynamically show "Total Shares" and "Estimated Cost" as the user types the lot quantity.
- **Action**: When the user clicks "Buy" or "Sell", the application will multiply the input value by the lot size before sending the order to the API.

## Non-Functional Requirements
- **Consistency**: All tables must use the "Qty (Lots)" or similar labeling to avoid confusion.
- **Accuracy**: Ensure floating-point precision is handled correctly when converting shares to lots (especially for fractional positions if applicable, though usually lots are integers).

## Acceptance Criteria
- [ ] `SearchModal` shows the lot size for every result.
- [ ] `Positions` table displays quantities in lots.
- [ ] Entering "1" in the Buy Modal for a security with lot size 10 results in an order for 10 shares.
- [ ] Trade History shows quantities consistent with the "Lots" unit.

## Out of Scope
- Support for "Odd Lot" trading (trading fewer shares than a single lot) if the broker/exchange allows it.
- Changing the underlying API storage; this is purely a display and input abstraction layer.
