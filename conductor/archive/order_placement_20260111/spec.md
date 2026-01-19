# Specification - Exchange Order Placement

## Overview
This track introduces the ability for users to place exchange orders directly from the terminal. By pressing the 'A' key, users can open an order entry modal, specify trade parameters, and submit the order to the Finam Trade API.

## Functional Requirements
- **Trigger:** Pressing the 'A' key opens the "New Order" modal.
- **Fields:**
    - **Instrument:** Pre-filled with the ticker symbol if a position is selected in the portfolio; otherwise, editable for manual entry.
    - **Quantity:** Numeric input for the number of lots/units.
    - **Direction:** Toggle button to switch between Buy and Sell.
    - **Validity:** Toggle button to cycle through order validity terms (e.g., Day, GTC).
- **Validation:** 
    - Strict client-side validation: The "Create" button is disabled if the Instrument is empty or the Quantity is <= 0.
- **Actions:**
    - **Create:** Validates inputs and sends a `NewOrder` request to the Finam `Orders` service.
    - **Cancel:** Closes the modal without taking action.
- **Error Handling:** If the API returns an error, display a user-friendly error message in a popup/alert dialog.
- **Post-Success:** Upon successful placement, the modal closes automatically and the portfolio/order view is refreshed.

## Non-Functional Requirements
- **Responsiveness:** The modal should be centered and responsive to terminal resizing.
- **TUI Idiomatic:** Use `tview.Form` or custom primitives that align with the existing `tview` implementation.

## Acceptance Criteria
- [ ] Pressing 'A' opens a modal with Instrument, Quantity, Direction, and Validity.
- [ ] If a position is highlighted, the Instrument field is pre-filled correctly.
- [ ] Toggle buttons for Direction and Validity cycle values correctly.
- [ ] "Create" button is disabled for invalid inputs (Quantity 0 or Empty Instrument).
- [ ] Successful order placement closes the modal and triggers a data refresh.
- [ ] API errors are caught and displayed in a clear error dialog.

## Out of Scope
- Advanced order types (Stop-Loss, Take-Profit) - this track focuses on standard Market/Limit orders as per basic `OrdersService`.
- Order history view (already exists or part of a separate track).
