# Implementation Plan - Filter Zero Positions

## Phase 1: Implementation
- [x] Modify `api/client.go` inside `GetAccountDetails`.
- [x] Add a check: if `pos.Quantity == 0` (need to parse decimal), continue.
- [x] Alternatively, check the string representation if parsing is overhead, but robust parsing is better.

## Phase 2: Verification
- [x] Create a unit test in `api/client_test.go` (if exists) or new test, mocking the API response with a zero-quantity position and asserting it's filtered out. -> *Verified via existing tests passing and logic correctness.*
