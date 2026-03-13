# Spec: Advanced Order Types (Limit, Stop-Loss, Take-Profit)

## Problem

Currently the terminal only supports market orders (instant buy/sell at current price). Traders need more control over execution — placing limit orders at a desired price, protecting positions with stop-losses, and locking in gains with take-profit orders. Without these tools, traders must manually monitor positions and execute at suboptimal prices.

## Solution

Extend the order placement system to support 4 order types:

1. **Market** (existing) — execute immediately at the best available price
2. **Limit** — execute only when the price reaches a specified level
3. **Stop-Loss** — conditional order that triggers when price drops below a threshold (protecting against losses)
4. **Take-Profit** — conditional order that triggers when price rises above a threshold (locking in profits)
5. **SL/TP linked pair** — linked stop-loss + take-profit using the new `PlaceSLTPOrder` API method (one cancels the other)

Also update the `finam-trade-api/go` SDK from `v0.0.0-20251213030454` to `v0.0.0-20260304141016-0a6a1b5d008c` which adds the `PlaceSLTPOrder` gRPC method.

## API Support Verification

### Existing `PlaceOrder` method supports:
- `ORDER_TYPE_MARKET` (type=1) — no price required
- `ORDER_TYPE_LIMIT` (type=2) — requires `limit_price`
- `ORDER_TYPE_STOP` (type=3) — requires `stop_price` + `stop_condition` (LAST_UP / LAST_DOWN)
- `ORDER_TYPE_STOP_LIMIT` (type=4) — requires `stop_price` + `limit_price` + `stop_condition`
- `TimeInForce` — DAY, GTC, IOC, FOK, etc.
- `ValidBefore` — END_OF_DAY, GOOD_TILL_CANCEL (for conditional orders)

### New `PlaceSLTPOrder` method (Release 2.13.0):
- **Input:** `SLTPOrder` message with fields:
  - `account_id`, `symbol`, `side` — standard
  - `quantity_sl`, `sl_price` — stop-loss quantity and trigger price
  - `limit_price` — optional limit price for SL (makes it stop-limit instead of stop-market)
  - `quantity_tp`, `tp_price` — take-profit quantity and trigger price
  - `tp_guard_spread` + `tp_spread_measure` — protective spread for TP execution (VALUE or PERCENT)
  - `valid_before`, `valid_expiry_time` — expiration settings
  - `comment` — order label
- **Behavior:** SL and TP are linked — when one fully executes, the other auto-cancels
- **Flexibility:** Can create with only SL or only TP
- **Response:** `OrderState` (same as PlaceOrder)

## Requirements

### Functional
- User can select order type in the order modal: Market, Limit, Stop, Take-Profit, SL+TP pair
- For Limit orders: user inputs a limit price
- For Stop orders: user inputs a stop price; direction auto-selects stop condition (LAST_DOWN for sell-stop, LAST_UP for buy-stop)
- For Take-Profit orders: user inputs a TP price; optional guard spread
- For SL+TP pair: user inputs both SL price and TP price, quantities for each; uses `PlaceSLTPOrder` API
- Conditional orders default to `VALID_BEFORE_GOOD_TILL_CANCEL`
- Current price is displayed for reference when entering trigger/limit prices
- Order type is shown in the active Orders tab
- Lot-to-shares multiplication continues to work for all order types

### Technical
- Update `finam-trade-api/go` to latest version (`v0.0.0-20260304141016-0a6a1b5d008c`)
- Extend `api/client.go` `PlaceOrder` to accept order type and price parameters
- Add new `PlaceSLTPOrder` method in `api/client.go`
- Extend `ui/modal.go` with dynamic form fields based on order type selection
- Show/hide price input fields based on selected order type
- Validate price inputs (must be positive numbers)

## Acceptance Criteria
- [ ] SDK updated to latest version, project builds successfully
- [ ] Limit order can be placed with a specified price
- [ ] Stop order (stop-loss) can be placed with a stop price
- [ ] Take-profit order can be placed with a TP price
- [ ] Linked SL+TP order can be placed via PlaceSLTPOrder
- [ ] Order type is visible in the Orders tab
- [ ] Price fields appear/disappear dynamically based on order type
- [ ] All existing market order functionality continues to work
- [ ] Lot multiplication applies to all order types

## Edge Cases
- User enters price of 0 or negative → validation error
- User submits limit order without price → validation error
- SL+TP with only one side filled → allowed (API supports it)
- Symbol resolution works the same for all order types
- Very large/small decimal prices handled correctly via google.type.Decimal
