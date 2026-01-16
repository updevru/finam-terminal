# Implementation Plan - Improve Quantity Input

## Phase 1: Implementation
- [x] Update `ui/close_modal.go` to set quantity field to empty string in `SetPositionData`.
- [x] Update `ui/modal.go` (or wherever `OrderModal` is) to set quantity field to empty string.
- [x] Update `GetQuantity` logic in both to handle empty strings (already returns 0 for parse error).

## Phase 2: Verification
- [x] Run unit tests to ensure validation still works (it should fail for empty/0).
- [x] Manual verification.
