# Implementation Plan: Close Position (close_position_20260111)

## Phase 1: Confirmation Modal UI [checkpoint: 52123a1]
- [x] Task: Create `ClosePositionModal` component in `ui/components.go` (Implemented in ui/close_modal.go) 4b379bb
- [x] Task: Implement input validation for the Quantity field 4b379bb
- [x] Task: Conductor - User Manual Verification 'Phase 1: Confirmation Modal UI' (Protocol in workflow.md) 52123a1

## Phase 2: Business Logic & Data Integration
- [x] Task: Extend `models.Position` or create a helper to determine order direction (Buy/Sell) 622687a
- [x] Task: Implement logic to handle partial vs full position closing 2b8c561
- [x] Task: Conductor - User Manual Verification 'Phase 2: Business Logic & Data Integration' (Protocol in workflow.md) 2b8c561

## Phase 3: Order Execution (API Client)
- [x] Task: Add `ClosePosition` method to `api/client.go` 9ee90b4
- [x] Task: Write TDD tests for `ClosePosition` in `api/client_test.go` 9ee90b4
- [x] Task: Conductor - User Manual Verification 'Phase 3: Order Execution (API Client)' (Protocol in workflow.md) 9ee90b4

## Phase 4: Main UI Integration
- [x] Task: Add 'c' key handler to the Positions Table in `ui/input.go` f274d1c
- [x] Task: Implement the "Show Modal" flow: (Early implementation) f274d1c
    - [x] Sub-task: Capture selected position data
    - [x] Sub-task: Populate and display modal
- [x] Task: Implement "Confirm" (Execute) handler: db6c40a
    - [x] Sub-task: Call API client
    - [x] Sub-task: Show success/error feedback
    - [x] Sub-task: Trigger data refresh on success
- [x] Task: Conductor - User Manual Verification 'Phase 4: Main UI Integration' (Protocol in workflow.md) 4726de6