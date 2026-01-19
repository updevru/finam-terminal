# Implementation Plan - Fix Close Position Bug

## Phase 1: Diagnosis and Reproduction
- [x] Create a reproduction test case in `ui/close_modal_test.go` that simulates a button press and asserts the callback is invoked.
- [x] Verify if `Validate()` is returning false unexpectedly or if the callback is nil.
  *   **Result:** Validation logic works for valid inputs, but the button handler silently fails if validation returns false. Also, `InputFieldInteger` restricts fractional inputs which is incorrect.

## Phase 2: Implementation Fix
- [x] Update `NewClosePositionModal` to accept an `onError` callback.
- [x] Remove `SetAcceptanceFunc(tview.InputFieldInteger)` from `ui/close_modal.go`.
- [x] Update button handler in `ui/close_modal.go` to call `onError` if validation fails.
- [x] Update `ui/app.go` to provide the `onError` callback (calling `ShowError`).
- [x] Add logging or visual feedback if validation fails.

## Phase 3: Verification
- [x] Run tests.
- [x] Manual verification steps.
