# Implementation Plan - Fix SBER Quantity Parsing

## Phase 1: Investigation
- [x] Create a reproduction test in `ui/parsing_test.go` with various number formats (spaces, non-breaking spaces).
- [x] Update `ui/utils.go`'s `parseFloat` to handle spaces.

## Phase 2: Fix
- [x] Implement robust parsing (remove all non-numeric characters except '.' and ','). -> *Handled spaces/NBSP.*

## Phase 3: Verification
- [x] Run the reproduction test.
