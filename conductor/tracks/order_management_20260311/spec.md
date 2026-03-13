# Spec: Order Management — Cancel, Modify, and Enhanced Display

## Problem

After implementing advanced order types (Limit, Stop-Loss, Take-Profit, SL/TP), users can now place various orders but have no way to manage them from the terminal. If an order is placed with incorrect parameters or market conditions change, the user must use a different tool (e.g. Finam web interface) to cancel or modify it. Additionally, the Orders tab lacks detail — it doesn't show all relevant fields for each order type (e.g. stop conditions, validity, executed quantity, SL/TP quantities separately).

## Solution

Enhance the Orders tab to become a full order management panel:

1. **Cancel Order** — select an active (non-executed) order and cancel it via the `CancelOrder` gRPC API
2. **Modify Order** — cancel the existing order and place a new one with updated parameters (the API has no native modify — this is cancel + re-place)
3. **Enhanced Display** — show all relevant order fields per type (stop condition, limit/stop prices, validity, executed vs. remaining quantity, linked SL/TP quantities)

## API Support

### CancelOrder (already in SDK)
- **Service:** `OrdersServiceClient`
- **Method:** `CancelOrder`
- **Request:** `CancelOrderRequest { account_id, order_id }`
- **Response:** `OrderState` (updated order with status `ORDER_STATUS_CANCELED`)
- **HTTP:** `DELETE /v1/accounts/{account_id}/orders/{order_id}`
- **Errors:** 404 (not found), 400 (cannot cancel — already executed/filled)

### No native Modify
- To modify an order: cancel the old one, then place a new one with updated params
- The UI should handle this as a two-step atomic operation with proper error handling

## Requirements

### Functional
- User can select an order in the Orders tab and press a key (e.g. `X` or `Del`) to cancel it
- Confirmation dialog before cancellation ("Cancel order #ID for TICKER? [Yes/No]")
- After cancellation, the Orders tab refreshes automatically
- User can select an order and press a key (e.g. `E`) to modify it — opens the order modal pre-filled with the order's current parameters
- Modify flow: cancel old order -> place new order -> refresh. If cancel succeeds but new placement fails, show error (the old order is already cancelled)
- Orders table shows additional columns/details per order type:
  - **All orders:** Order ID (or truncated), Executed/Remaining quantity
  - **Stop orders:** Stop condition (Last Up / Last Down), stop price
  - **Limit orders:** Limit price
  - **Stop-Limit orders:** Both stop and limit prices
  - **SL/TP orders:** SL price, TP price, SL quantity, TP quantity (shown separately)
  - **All conditional orders:** Validity (GTC / End of Day / etc.)
- Status bar shows context-aware shortcuts when Orders tab is focused (e.g. `X Cancel  E Modify`)
- Only cancellable orders (status: New, Partial) show cancel/modify actions

### Technical
- Add `CancelOrder(accountID, orderID string) error` to `api/client.go`
- Add `CancelOrder` to `APIClient` interface in `ui/app.go`
- Extend `models.Order` with additional fields: `StopCondition`, `LimitPrice`, `StopPrice`, `Validity`, `ExecutedQty`, `RemainingQty`, `SLQty`, `TPQty`, `SLPrice`, `TPPrice`
- Update `GetActiveOrders` in `api/client.go` to populate the new fields
- Create cancel confirmation modal in `ui/`
- Pre-fill order modal for modify flow
- Update `updateOrdersTable` in `ui/render.go` with enhanced columns
- Update `updateStatusBar` to show order-specific shortcuts

## Acceptance Criteria
- [ ] CancelOrder API method implemented and wired
- [ ] User can cancel an active order from the Orders tab with confirmation
- [ ] After cancellation, order list refreshes and shows updated status
- [ ] User can modify an order (cancel + re-place) with pre-filled modal
- [ ] Orders table displays all relevant fields per order type
- [ ] SL/TP linked orders show both SL and TP prices and quantities
- [ ] Stop orders show stop condition direction
- [ ] Status bar shows Cancel/Modify shortcuts when Orders tab is focused
- [ ] Non-cancellable orders (Filled, Executed, Cancelled) cannot be cancelled or modified
- [ ] Error handling: network errors, order already executed, etc.

## Edge Cases
- User tries to cancel an already-executed order -> show error message from API
- User tries to cancel an already-cancelled order -> show error message
- Modify flow: cancel succeeds but re-place fails -> show error, don't silently lose the order
- Order disappears (filled) between display and cancel attempt -> API returns 404, show friendly message
- SL/TP linked order: cancelling cancels both SL and TP sides
- Orders tab refreshed by another account switch during cancel -> no race condition

## Dependencies
- Finam Trade API SDK v0.0.0-20260304141016-0a6a1b5d008c (already updated)
- Existing `OrdersServiceClient` with `CancelOrder` method
- Existing order placement infrastructure (PlaceOrder, PlaceSLTPOrder)
