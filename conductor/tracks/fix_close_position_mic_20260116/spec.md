# Specification: Fix 'Mic must not be empty' Error (fix_close_position_mic_20260116)

## Overview
Users are reporting a "Mic must not be empty" error when attempting to close positions. This is caused by the application passing only the Ticker (e.g., "SIH6") instead of the full Symbol (e.g., "SIH6@FUT") to the API. The API requires the MIC (Market Identifier Code) to identify the correct trading board.

## Functional Requirements
- **Close Position:** The `ClosePosition` operation must utilize the full symbol (Ticker + MIC) to ensure the order is correctly routed by the Finam API.
- **UI Interaction:** The user interaction ('c' key -> Modal -> Confirm) remains unchanged, but the underlying data submission must be corrected.

## Technical Changes
- **`ui/app.go`**: Update `SubmitClosePosition` to pass `pos.Symbol` (which typically contains `Ticker@MIC`) instead of `pos.Ticker` to the `ClosePosition` API method.

## Acceptance Criteria
- [ ] `SubmitClosePosition` passes the full symbol to the API.
- [ ] Unit tests for `SubmitClosePosition` verify that the symbol passed contains the MIC (or matches the full symbol format).
