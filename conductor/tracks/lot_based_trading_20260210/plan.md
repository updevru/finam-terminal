# Implementation Plan - Lot-Based Trading and Display

This plan implements lot-based display and trading logic across the application to align with professional trading standards.

## Phase 1: API & Models Enhancement [checkpoint: 37ea4b8]
Goal: Ensure the application has access to lot size metadata for all instruments and positions.

- [x] Task: Update `models.Position` to include lot size information. 2688281
    - [ ] Write failing tests in `models/models_test.go` to verify `Position` can store and handle `LotSize`.
    - [ ] Add `LotSize float64` field to `Position` struct in `models/models.go`.
- [x] Task: Enhance `api.Client` to retrieve and cache lot sizes. 56b30bc
    - [ ] Write failing tests in `api/client_test.go` to verify `GetAsset` (and `getFullSymbol`) captures `LotSize`.
    - [ ] Update `api/client.go` to extract `LotSize` from `GetAssetResponse` and store it in the `assetMicCache` or a new dedicated cache.
    - [ ] Update `GetAccountDetails` in `api/client.go` to populate the `LotSize` for each returned `Position`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: API & Models Enhancement' (Protocol in workflow.md)

## Phase 2: UI Search & Portfolio Display
Goal: Update the main display components to show quantities in lots.

- [x] Task: Update `SearchModal` to display Lot Size. f4aa60b
    - [ ] Write failing tests in `ui/search_test.go` to verify the "Lot" column exists and shows correct data.
    - [ ] Update `ui/search.go` to add a "Lot" column to the results table and populate it from `SecurityInfo`.
- [ ] Task: Update Portfolio/Positions table for lot-based display.
    - [ ] Write failing tests in `ui/portfolio_test.go` to verify the quantity column shows lots instead of shares.
    - [ ] Update `ui/render.go` (or wherever the positions table is rendered) to rename "Qty" to "Qty (Lots)".
    - [ ] Implement calculation: `DisplayQty = TotalShares / LotSize`.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: UI Search & Portfolio Display' (Protocol in workflow.md)

## Phase 3: Trading Modals & Logic
Goal: Implement lot-based input and validation in Buy and Close modals.

- [ ] Task: Update `BuyModal` (or Order placement UI) for lot-based input.
    - [ ] Write failing tests in `ui/modal_test.go` for the new lot-based calculation logic.
    - [ ] Update UI to display the multiplier (e.g., "1 lot = 10").
    - [ ] Implement real-time calculation of "Total Shares" and "Estimated Cost" based on lot input.
- [ ] Task: Update `CloseModal` for lot-based closing.
    - [ ] Write failing tests in `ui/close_modal_test.go` to verify closing quantities are handled as lots.
    - [ ] Update UI to show current position in lots and allow entering close quantity in lots.
- [ ] Task: Update `api.Client.PlaceOrder` to handle lot multiplication.
    - [ ] Write failing tests in `api/client_test.go` to verify that placing an order with quantity 1 (lot) results in an API call with `1 * LotSize` shares.
    - [ ] Modify `PlaceOrder` in `api/client.go` to perform the multiplication.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Trading Modals & Logic' (Protocol in workflow.md)

## Phase 4: History & Consistency
Goal: Ensure all other quantity displays are consistent.

- [ ] Task: Update Trade History and Active Orders views.
    - [ ] Write failing tests to verify history quantities are shown in lots.
    - [ ] Update rendering logic for history and orders to use lot-based quantities.
- [ ] Task: Conductor - User Manual Verification 'Phase 4: History & Consistency' (Protocol in workflow.md)
