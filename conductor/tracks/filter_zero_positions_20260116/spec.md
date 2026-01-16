# Specification: Filter Zero Quantity Positions

## Problem Description
The user sees "phantom" positions (specifically SBER) in the TUI that do not exist in their real trading terminal (which is empty). Attempting to close these positions results in errors because their quantity is likely 0 (or they are effectively closed).

## Current Behavior
The application displays all positions returned by the API, including those with 0 quantity (which the API apparently returns for history/tracking purposes).

## Expected Behavior
The application should strictly filter out positions where `Quantity` is 0, matching the behavior of the standard trading terminal which hides closed positions.

## Root Cause Analysis
The `GetAccountDetails` method in `api/client.go` iterates through all positions returned by the gRPC response and adds them to the list without checking if the quantity is non-zero.

## Requirements
- In `api/client.go`, parse the quantity of each position.
- If quantity is 0, skip adding it to the returned slice.
