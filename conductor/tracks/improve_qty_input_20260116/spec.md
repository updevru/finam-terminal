# Specification: Improve Quantity Input in Modals

## Problem Description
User reports that the quantity field in the "Close Position" (and potentially "New Order") modal contains "0" by default, which requires manual erasure before entering a new value. They prefer the field to be empty by default.

## Requirements
- In the "New Order" modal (OrderModal), the quantity field should be empty when opened.
- In the "Close Position" modal (ClosePositionModal), the quantity field should be empty when opened, allowing the user to type immediately.
- Validation must still ensure a value is entered.

## Proposed Changes
- Update `OrderModal.SetQuantity(0)` (or similar) to use an empty string if 0.
- Update `ClosePositionModal.SetPositionData` to NOT pre-fill the quantity field with the current position size, or set it to empty.
  * *Note:* For "Close Position", pre-filling with max is often a feature, but user specifically asked for "empty" (пусто). I will follow the user request.
