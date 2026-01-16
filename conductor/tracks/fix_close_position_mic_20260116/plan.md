# Implementation Plan: Fix 'Mic must not be empty' Error (fix_close_position_mic_20260116)

## Phase 1: Bug Fix
- [x] Task: Reproduce the issue with a failing test case in `ui/submission_test.go` (asserting that `ClosePosition` receives the full symbol).
- [x] Task: Fix the issue in `ui/app.go` by using `pos.Symbol`. d89a699
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Bug Fix' (Protocol in workflow.md)
