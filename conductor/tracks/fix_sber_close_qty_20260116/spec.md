# Specification: Fix Zero Quantity Error for SBER

## Problem Description
The user selected "SBER" in the portfolio and pressed 'C' to close. The application showed an error stating the quantity is 0 ("Position SBER has non-positive quantity: ..."), despite the position being visible in the table with a valid quantity.

## Current Behavior
1. User selects SBER position.
2. User presses 'C'.
3. Error modal appears: "Position SBER has non-positive quantity: [value]".

## Expected Behavior
1. Modal opens with the correct max quantity.

## Root Cause Analysis (Hypothesis)
*   **Parsing:** The `Quantity` string for SBER might contain characters that `parseFloat` handles incorrectly (e.g., non-breaking spaces `\u00A0` often used as thousand separators in Russian locales, or just regular spaces).
*   **Data Mismatch:** The `pos` object retrieved might be the wrong one, or the data in `a.positions` is stale/corrupt.

## Requirements
*   Identify the exact string value of `Quantity` that causes failure.
*   Improve `parseFloat` to handle spaces or other common separators.
