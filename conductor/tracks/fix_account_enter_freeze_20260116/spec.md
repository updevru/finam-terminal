# Specification: Fix Freeze on Account Table Enter

## Problem Description
The user reports that accidentally pressing ENTER while the Accounts table is focused causes the entire accounts section to disappear and the program to hang/freeze.

## Current Behavior
1. User navigates focus to the Accounts table.
2. User presses ENTER.
3. **Result:** Accounts section disappears, UI becomes unresponsive.

## Expected Behavior
1. User presses ENTER on the Accounts table.
2. **Result:** Either nothing happens (ignore input), or it selects the account (if that's the intended feature), but it MUST NOT crash or freeze the UI.

## Root Cause Analysis (Hypothesis)
*   There is likely a `SetSelectedFunc` or similar handler on the `AccountTable` that triggers an invalid UI update or blocking operation.
*   The "disappearance" suggests a layout issue, possibly removing the table from the flex container or setting its size to 0.
*   The "freeze" suggests a deadlock or an infinite loop, potentially related to `tview`'s event loop or a mutex lock.

## Requirements
*   Identify the input handler for the Accounts table.
*   Fix the logic to prevent the crash/freeze.
*   Ensure the UI remains stable.
