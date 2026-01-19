# Implementation Plan - Fix Account Table Freeze

## Phase 1: Investigation
- [x] Analyze `ui/portfolio.go` (or where `AccountTable` is initialized) to check for `SetSelectedFunc`.
- [x] Analyze `ui/input.go` to check for global input handlers that might affect the accounts table.
- [x] Create a reproduction test case (if possible in TUI testing) or simulate the callback logic.

## Phase 2: Fix
- [x] Remove or fix the problematic `SelectedFunc` (in this case, removed `refresh()` from Enter handler).
- [x] Ensure input handling for the Account Table is safe.

## Phase 3: Verification
- [x] Verify via test or manual simulation.
