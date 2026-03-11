# Plan: Order Management — Cancel, Modify, and Enhanced Display

## Overview

Add order cancellation and modification capabilities to the Orders tab, along with enhanced order display showing all relevant fields per order type. The API already provides `CancelOrder`; modification is implemented as cancel + re-place. The Orders tab will become a full management panel with keyboard shortcuts and confirmation dialogs.

## Phase 1: API Layer — CancelOrder

- [ ] Task: Add `CancelOrder` method to `api/client.go`
  - Implement `CancelOrder(accountID, orderID string) error` using `ordersClient.CancelOrder`
  - Use structured error logging via `logGRPCError`
  - Acceptance: Method compiles, handles gRPC errors, returns nil on success

- [ ] Task: Add `CancelOrder` to `APIClient` interface
  - Update `APIClient` interface in `ui/app.go`
  - Update `MockClient` in `ui/mock_client_test.go` if it exists
  - Acceptance: All code compiles with updated interface

## Phase 2: Enhanced Order Model & Data

- [ ] Task: Extend `models.Order` with detailed fields
  - Add fields: `StopCondition`, `LimitPrice`, `StopPrice`, `Validity`, `ExecutedQty`, `RemainingQty`, `SLQty`, `TPQty`, `SLPrice`, `TPPrice`
  - Acceptance: Model compiles, no breaking changes to existing code

- [ ] Task: Update `GetActiveOrders` to populate new fields
  - Parse stop condition from `o.Order.StopCondition` -> "Last Up" / "Last Down"
  - Parse both `LimitPrice` and `StopPrice` separately (currently only one is shown as `Price`)
  - Parse `ValidBefore` -> "GTC" / "Day" / etc.
  - For SL/TP orders: extract `SLPrice`, `TPPrice`, `SLQty`, `TPQty` from `o.SltpOrder`
  - Parse executed quantity from `o.FilledQuantity` or similar field
  - Acceptance: Orders loaded from API have all new fields populated

## Phase 3: Enhanced Orders Table Display

- [ ] Task: Redesign `updateOrdersTable` with richer columns
  - Show columns: Instrument, Side, Type, Status, Qty, Executed, Price/Condition, Validity, Time
  - For Stop orders: show "SL: 100.50" or "TP: 150.00" with condition
  - For SL/TP linked: show "SL:100.50 / TP:150.00" with separate quantities
  - For Limit: show limit price
  - Color-code: cancellable orders in normal colors, non-cancellable (filled/cancelled) dimmed
  - Acceptance: All order types display correctly with full details

- [ ] Task: Update status bar for Orders tab context
  - When Orders tab is focused, show: `X Cancel  E Modify  R Refresh`
  - Only show Cancel/Modify when a cancellable order is selected
  - Acceptance: Status bar updates dynamically based on active tab and selection

## Phase 4: Cancel Order Flow

- [ ] Task: Create cancel confirmation modal
  - Simple Yes/No modal: "Cancel order TYPE SIDE TICKER @ PRICE? [Yes/No]"
  - On Yes: call `CancelOrder`, refresh orders list, show status
  - On No: return to orders table
  - Acceptance: Modal appears, handles Yes/No, order gets cancelled

- [ ] Task: Wire cancel to keyboard shortcut
  - `X` or `Del` on Orders tab triggers cancel flow
  - Only for orders with status "New" or "Partial"
  - Show error if order is not cancellable
  - Acceptance: Pressing X on a cancellable order shows confirmation, completes cancel

## Phase 5: Modify Order Flow

- [ ] Task: Implement modify flow (cancel + re-place)
  - `E` on Orders tab opens the order modal pre-filled with order's current parameters
  - Extract: symbol, side, type, quantity, price(s) from the selected order
  - On submit: cancel old order first, then place new order
  - If cancel fails: show error, don't place new order
  - If cancel succeeds but placement fails: show error explaining old order was cancelled
  - Acceptance: User can modify order parameters, old order cancelled, new order placed

- [ ] Task: Pre-fill order modal from existing order data
  - Map order fields back to modal fields (type dropdown, price inputs, quantity)
  - Handle all order types: Market, Limit, Stop, Take-Profit, SL/TP
  - For SL/TP: pre-fill both SL and TP prices and quantities
  - Acceptance: Modal opens with all fields correctly pre-filled from selected order

## Phase 6: Polish & Edge Cases

- [ ] Task: Handle error scenarios gracefully
  - Order already executed/cancelled when user tries to cancel -> friendly message
  - Network timeout during cancel -> show error, suggest refresh
  - Race condition: order fills between display and cancel -> handle 404/400 from API
  - Acceptance: All error paths produce user-friendly messages

- [ ] Task: Ensure background refresh compatibility
  - Orders tab auto-refreshes alongside positions
  - Cancel/modify operations trigger immediate refresh
  - No race conditions between background refresh and user-initiated cancel
  - Acceptance: Background refresh works, no duplicate API calls during cancel
