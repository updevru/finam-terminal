# Specification: Close Position Functionality (close_position_20260111)

## Overview
Implement the ability for users to close existing positions directly from the Portfolio view. This involves selecting a position, viewing a confirmation modal with calculated totals, and executing a market order to close the specified quantity.

## Functional Requirements
- **Trigger:** Pressing the 'c' key while a position is selected in the Positions Table.
- **Confirmation Modal:**
    - Type: Centered modal window overlaying the position list.
    - Fields:
        - Symbol (Read-only)
        - Quantity (Editable, defaults to full position size)
        - Last Price (Read-only)
        - Estimated Total (Read-only, calculated as Qty * LastPrice)
        - Unrealized PnL (Read-only)
    - Interactions:
        - `Enter`: Submit the market order.
        - `Esc`: Close the modal and cancel the operation.
- **Execution:**
    - Order Type: Market Order.
    - Direction: Inverse of current position (Sell for Long, Buy for Short).
- **Post-Execution:**
    - Automatically refresh the positions table to reflect the updated state.
- **Error Handling:**
    - Display clear, user-friendly error messages for API failures or invalid inputs.

## Non-Functional Requirements
- **UI Responsiveness:** The modal must appear instantly without blocking the main event loop.
- **Data Accuracy:** Calculated totals must reflect the current state of the selected position.

## Acceptance Criteria
- [ ] Pressing 'c' on a position opens the modal.
- [ ] Modal correctly calculates 'Estimated Total' based on user-edited quantity.
- [ ] 'Enter' successfully submits a market order to the Finam API.
- [ ] 'Esc' closes the modal without placing an order.
- [ ] After successful execution, the positions table is updated.
- [ ] Failed orders show a specific error message to the user.

## Out of Scope
- Limit orders for closing positions.
- Stop-loss/Take-profit settings within the close modal.
