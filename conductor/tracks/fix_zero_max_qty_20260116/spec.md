# Specification: Fix Zero Max Quantity in Close Modal

## Problem Description
The user reports an error "Invalid quantity. Must be > 0 and <= 0" when trying to close a position, even though they possess shares (e.g., 1 share) and entered a valid quantity (e.g., 1). This implies the application calculates the `maxQuantity` available to close as 0.

## Current Behavior
1. User selects a position in the portfolio table.
2. User opens the Close Modal.
3. The `maxQuantity` state in the modal is incorrectly set to 0.
4. Validation fails because input `1` > `0`.

## Expected Behavior
1. The `maxQuantity` should correctly reflect the available quantity of the selected position.
2. Validation should pass for valid quantities <= `maxQuantity`.

## Root Cause Analysis (Hypothesis)
1. **Index Mismatch:** The row selected in the table (`row - 1`) does not correspond to the index in the `a.positions` slice. This could happen if the table rendering logic (filtering/sorting/headers) differs from the raw slice order.
2. **Parsing Error:** The `Quantity` string in `models.Position` might contain characters that `strconv.ParseFloat` cannot handle (e.g., whitespace, commas, or specific formatting from `formatDecimal`), causing it to return 0.
3. **Data Staleness:** The `a.positions` map might not be updated or might be empty when `OpenCloseModal` is called, though this is unlikely if the table shows data.

## Requirements
*   Ensure the correct position is retrieved corresponding to the selected table row.
*   Ensure the `Quantity` string is correctly parsed into a float.
