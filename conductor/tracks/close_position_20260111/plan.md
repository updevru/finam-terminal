# Implementation Plan: Close Position (close_position_20260111)

## Phase 1: Confirmation Modal UI
- [ ] Task: Create `ClosePositionModal` component in `ui/components.go`
    - [ ] Sub-task: Define the struct and basic layout using `tview.Form` or `tview.Flex`
    - [ ] Sub-task: Implement fields: Symbol (Text), Quantity (InputField), Last Price (Text), Estimated Total (Text), PnL (Text)
- [ ] Task: Implement input validation for the Quantity field
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Confirmation Modal UI' (Protocol in workflow.md)

## Phase 2: Business Logic & Data Integration
- [ ] Task: Extend `models.Position` or create a helper to determine order direction (Buy/Sell)
- [ ] Task: Implement calculation logic for "Estimated Total" based on quantity input
- [ ] Task: Write TDD tests for quantity-to-total calculations in `ui/data_test.go`
- [ ] Task: Implement the update logic for the modal when quantity changes
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Business Logic & Data Integration' (Protocol in workflow.md)

## Phase 3: Order Execution (API Client)
- [ ] Task: Add `ClosePosition` method to `api/client.go`
    - [ ] Sub-task: Construct the `NewOrderRequest` (Market Order, Inverse direction)
    - [ ] Sub-task: Handle gRPC call and map response/errors
- [ ] Task: Write TDD tests for `ClosePosition` in `api/client_test.go` using mocks
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Order Execution (API Client)' (Protocol in workflow.md)

## Phase 4: Main UI Integration
- [ ] Task: Add 'c' key handler to the Positions Table in `ui/portfolio.go`
- [ ] Task: Implement the "Show Modal" flow:
    - [ ] Sub-task: Capture selected position data
    - [ ] Sub-task: Populate and display modal
- [ ] Task: Implement "Confirm" (Enter) handler:
    - [ ] Sub-task: Call API client
    - [ ] Sub-task: Show success/error feedback
    - [ ] Sub-task: Trigger data refresh on success
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Main UI Integration' (Protocol in workflow.md)
