# Implementation Plan: Proactive Token Refresh

This plan implements a background process to refresh the Finam API JWT token before it expires, preventing "Unauthenticated" errors during long sessions.

## Phase 1: Foundation & Data Structure Changes [checkpoint: b92e021]
- [x] Task: Update Client struct and implement JWT parsing [commit: b3f2865]
    - [ ] Write failing tests in `api/client_test.go` for extracting expiry from a JWT string.
    - [ ] Update `Client` struct in `api/client.go` to store `apiToken` (string), `lastRefresh` (time.Time), and a `refreshCancel` (context.CancelFunc).
    - [ ] Implement a private helper `getExpiryFromToken(token string) (time.Time, error)` in `api/client.go`.
    - [ ] Ensure tests pass by correctly decoding the JWT payload.
- [x] Task: Conductor - User Manual Verification 'Foundation & Data Structure Changes' (Protocol in workflow.md) [commit: 2c9783d]

## Phase 2: Background Refresh Implementation [checkpoint: f0b5f97]
- [x] Task: Implement background refresh lifecycle [commit: 3d6be9e]
    - [ ] Write failing tests (or descriptive stubs) for the refresh loop triggering authentication.
    - [ ] Implement `startTokenRefresh(ctx context.Context)` method in `api/client.go` containing the `for` loop and `time.After`.
    - [ ] Update `NewClient` to initialize the refresh context and launch the goroutine.
    - [ ] Update `Close()` in `api/client.go` to invoke the `refreshCancel` function.
    - [ ] Make tests pass by verifying the authentication call is made before expiry.
- [x] Task: Conductor - User Manual Verification 'Background Refresh Implementation' (Protocol in workflow.md) [commit: 5e3201d]

## Phase 3: Robustness & Logging
- [ ] Task: Add retries and observability
    - [ ] Implement a simple retry mechanism (e.g., 30s delay) if `authenticate` fails within the refresh loop.
    - [ ] Add `[INFO]` logging for successful refreshes and `[ERROR]` logging for failures.
    - [ ] Update `lastRefresh` timestamp on every success.
    - [ ] Verify unit test coverage for `api` package remains >80%.
- [ ] Task: Conductor - User Manual Verification 'Robustness & Logging' (Protocol in workflow.md)
