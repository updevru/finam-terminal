# Specification: Fix MIC Resolution (fix_mic_resolution_20260116)

## Overview
The "Mic must not be empty" error persists because the application fails to resolve the correct MIC (Market Identifier Code) for the user's positions (e.g., "FXRL"). This indicates that the `assetMicCache` mechanism in `api/client.go` is either failing to load, incomplete, or the mapping logic is flawed.

## Functional Requirements
- **Robust Symbol Resolution:** The application must correctly identify the full symbol (Ticker@MIC) for all positions held by the user.
- **Error Handling:** If the MIC cannot be resolved, the application should log a clear warning or error, rather than failing silently until order submission.

## Technical Goals
- **Debug Cache Loading:** Verify if `loadAssetCache` is successfully retrieving assets and if the target ticker ("FXRL") is present.
- **Inspect API Response:** Determine if the `GetAccount` response contains additional fields (like `Market` or `Board`) that can be used to construct the full symbol without relying solely on the cache.
- **Fix:** Implement a robust mechanism to populate `Position.Symbol` with the full `Ticker@MIC` string.

## Acceptance Criteria
- [ ] Closing a position for "FXRL" (or any other asset) successfully includes the MIC in the API request.
- [ ] Debug logs provide visibility into symbol resolution success/failure.
