# Plan: Advanced Order Types (Limit, Stop-Loss, Take-Profit)

## Overview

Extend the trading terminal with limit, stop-loss, take-profit, and linked SL+TP orders. Update the Finam SDK to the latest version which includes the new `PlaceSLTPOrder` gRPC method. Modify the API layer to support all order types and update the UI modal with dynamic price fields.

---

## Phase 1: SDK Update & Foundation

- [ ] Task: Update `finam-trade-api/go` dependency to `v0.0.0-20260304141016-0a6a1b5d008c`
  - Run `go get github.com/FinamWeb/finam-trade-api/go@v0.0.0-20260304141016-0a6a1b5d008c && go mod tidy`
  - Verify project compiles with `go build ./...`
  - Acceptance: Project builds successfully with the new SDK

- [ ] Task: Extend `models/models.go` with order type constants and new fields
  - Add order type constants: `OrderTypeMarket`, `OrderTypeLimit`, `OrderTypeStop`, `OrderTypeSLTP`
  - No new model structs needed — reuse existing `Order` model which already has `Type` and `Price` fields
  - Acceptance: Constants defined, existing code still compiles

---

## Phase 2: API Layer — Limit & Stop Orders

- [ ] Task: Refactor `api/client.go` `PlaceOrder` to accept order parameters
  - Change signature to accept a struct or additional params: order type, limit price, stop price, stop condition, time-in-force
  - For `ORDER_TYPE_MARKET`: current behavior (no price)
  - For `ORDER_TYPE_LIMIT`: set `LimitPrice` on the proto `Order`
  - For `ORDER_TYPE_STOP`: set `StopPrice` + `StopCondition` (LAST_DOWN for sell-stop, LAST_UP for buy-stop)
  - For `ORDER_TYPE_STOP_LIMIT`: set both `StopPrice` + `LimitPrice` + `StopCondition`
  - Set `ValidBefore` to `VALID_BEFORE_GOOD_TILL_CANCEL` for conditional orders
  - Acceptance: PlaceOrder correctly builds proto messages for all order types

- [ ] Task: Add `PlaceSLTPOrder` method to `api/client.go`
  - New method: `PlaceSLTPOrder(accountID, symbol, side string, slQty, slPrice, tpQty, tpPrice float64, opts ...SLTPOption) (string, error)`
  - Build `SLTPOrder` proto message with proper fields
  - Handle lot-size multiplication for both SL and TP quantities
  - Resolve full symbol via `getFullSymbol()`
  - Call `ordersClient.PlaceSLTPOrder()` via gRPC
  - Log errors using `logGRPCError`
  - Acceptance: SL/TP orders can be placed via the API client

---

## Phase 3: UI — Order Type Selection & Dynamic Fields

- [ ] Task: Add order type dropdown to `ui/modal.go`
  - Add a `DropDown` field for order type: "Market", "Limit", "Stop-Loss", "Take-Profit", "SL + TP"
  - Store selected order type in the modal state
  - Acceptance: Dropdown appears in the order modal

- [ ] Task: Add dynamic price input fields
  - Add `limitPrice` input field (shown for Limit)
  - Add `stopPrice` input field (shown for Stop-Loss)
  - Add `tpPrice` input field (shown for Take-Profit)
  - Add `slPrice` + `tpPrice` fields (shown for SL+TP pair)
  - Show/hide fields dynamically when order type changes
  - Display current price as reference label
  - Acceptance: Price fields appear/disappear based on order type selection

- [ ] Task: Update validation and submission logic in modal
  - Validate price fields are positive numbers when required
  - For Limit: require limit price
  - For Stop-Loss: require stop price
  - For Take-Profit: require TP price
  - For SL+TP: require at least one of SL price or TP price
  - Acceptance: Invalid orders are rejected with clear messages

---

## Phase 4: UI — Wiring Submission to API

- [ ] Task: Update `ui/app.go` `SubmitOrder` to handle all order types
  - Accept order type and price parameters from the modal
  - For Market/Limit/Stop: call extended `PlaceOrder`
  - For SL+TP: call `PlaceSLTPOrder`
  - Preserve existing refresh-after-order behavior
  - Acceptance: All order types submit correctly through the UI

- [ ] Task: Update Orders tab display to show order type and prices
  - Show order type column (Market, Limit, Stop, SL/TP) in the orders table
  - Show trigger/limit price where applicable
  - Acceptance: Users can see order types and prices in the Orders tab

---

## Phase 5: Testing & Polish

- [ ] Task: Manual end-to-end verification
  - Test market order (regression — still works)
  - Test limit order placement
  - Test stop-loss order placement
  - Test take-profit order placement
  - Test SL+TP linked order placement
  - Verify orders appear in Orders tab with correct type and price
  - Test validation (missing prices, zero prices, negative prices)
  - Acceptance: All order types work end-to-end

- [ ] Task: Update CLAUDE.md with new API methods documentation
  - Document `PlaceSLTPOrder` API method
  - Document order type support in `PlaceOrder`
  - Acceptance: CLAUDE.md reflects current capabilities
