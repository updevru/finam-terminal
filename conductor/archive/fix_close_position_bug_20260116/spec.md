# Specification: Fix Close Position Button Inactivity

## Problem Description
The user reports that when the "Close Position" modal is open, clicking the "Close Position" button (or pressing Enter on it) results in no action. The position is not closed, and the modal remains open or behaves unexpectedly (effectively "nothing happens").

## Current Behavior
1.  User selects a position.
2.  User presses 'C' to open the Close Modal.
3.  Modal appears with details.
4.  User selects "Close Position" button.
5.  **Result:** No action is taken. The application does not send a request, and no error or success message is displayed.

## Expected Behavior
1.  User selects "Close Position" button.
2.  **Validation:** The input quantity is validated.
3.  **Action:** If valid, the `callback` function provided to the modal is executed with the specified quantity.
4.  **Feedback:** The application attempts to close the position via the API.
    *   On Success: Status bar shows success, modal closes, position list updates.
    *   On Error: Error message is displayed.

## Root Cause Analysis (Hypothesis)
*   The `tview.Form` button callback might not be triggering the `m.callback` correctly.
*   The `Validate` function might be failing silently.
*   The input field focus might be trapping the event.

## Requirements
*   Ensure the "Close Position" button correctly triggers the submission logic.
*   Ensure validation failures provide some visual feedback (or at least don't silently fail if possible, though strict TUI limitations apply).
*   Verify the integration between `ClosePositionModal` and `App`.
