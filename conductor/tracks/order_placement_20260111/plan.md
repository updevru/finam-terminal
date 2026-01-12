# Implementation Plan - Exchange Order Placement

## Phase 1: API Layer - Orders Service Integration [checkpoint: 5155d26]
- [x] Task: Implement `PlaceOrder` method in `api/client.go` 267da5b
    - [ ] Write failing tests for `PlaceOrder` (success and error cases) in `api/client_test.go`
    - [ ] Implement `PlaceOrder` using `OrdersClient` from Finam SDK
    - [ ] Verify tests pass and check coverage
- [ ] Task: Conductor - User Manual Verification 'API Layer - Orders Service Integration' (Protocol in workflow.md)

## Phase 2: UI Component - Order Entry Modal [checkpoint: 882b680]
- [x] Task: Create the `OrderModal` primitive and layout 48b7df6
    - [ ] Write tests for modal field initialization and validation logic
    - [ ] Implement `OrderModal` in `ui/` with Instrument, Quantity, Direction, and Validity fields
    - [ ] Implement cycling toggle buttons for Direction (Buy/Sell) and Validity
    - [ ] Implement reactive "Create" button (disabled if inputs invalid)
- [ ] Task: Conductor - User Manual Verification 'UI Component - Order Entry Modal' (Protocol in workflow.md)

## Phase 3: Integration and User Interaction
- [x] Task: Wire up the 'A' key and instrument pre-filling da28407
    - [ ] Write tests for the input handler and context-aware pre-filling
    - [ ] Update `ui/input.go` to trigger the `OrderModal` on 'A' press
    - [ ] Pass the currently selected position's symbol to the modal
- [x] Task: Implement submission and error handling fa24d56
    - [ ] Write tests for the submission flow (success/failure handling)
    - [ ] Connect "Create" button to the API client's `PlaceOrder`
    - [ ] Implement error popup for API failures and refresh logic for success
- [ ] Task: Conductor - User Manual Verification 'Integration and User Interaction' (Protocol in workflow.md)
