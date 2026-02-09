# Implementation Plan - Portfolio Tabs (History & Orders)

This plan outlines the steps to implement a tabbed interface for the Portfolio view, adding History and Orders tabs alongside the existing Positions view.

## Phase 1: API Client Extensions
Enhance the API client to fetch trade history and active orders from the Finam gRPC services.

- [x] Task: Define `Trade` and `Order` models in `models/models.go` (4546289)
- [x] Task: Implement `GetTradeHistory(accountID string)` in `api/client.go` (4546289)
- [x] Task: Implement `GetActiveOrders(accountID string)` in `api/client.go` (4546289)
- [x] Task: Write unit tests for new API methods in `api/client_test.go` (4546289)
- [ ] Task: Conductor - User Manual Verification 'Phase 1: API Client Extensions' (Protocol in workflow.md)

## Phase 2: UI Tab Infrastructure
Refactor the existing Positions view to support multiple tabs and navigation logic.

- [ ] Task: Create a `TabbedView` component in `ui/components.go` that manages active tab state
- [ ] Task: Implement tab navigation logic (Left/Right arrows, Tab/Shift+Tab) in `ui/input.go`
- [ ] Task: Update `ui/render.go` to display tab headers with the specified highlight style
- [ ] Task: Write tests for tab switching logic in `ui/input_handler_test.go`
- [ ] Task: Conductor - User Manual Verification 'Phase 2: UI Tab Infrastructure' (Protocol in workflow.md)

## Phase 3: History & Orders Views
Implement the specific table renderings and data fetching for the new tabs.

- [ ] Task: Implement `renderHistoryTable()` in `ui/render.go` with requested columns
- [ ] Task: Implement `renderOrdersTable()` in `ui/render.go` with requested columns
- [ ] Task: Implement "Refresh on Entry" logic in `ui/app.go` when switching tabs
- [ ] Task: Implement manual refresh ('R' key) in `ui/input.go`
- [ ] Task: Write integration tests for data loading in History/Orders tabs
- [ ] Task: Conductor - User Manual Verification 'Phase 3: History & Orders Views' (Protocol in workflow.md)

## Phase 4: Refinement & Polishing
Ensure visual consistency and smooth user experience.

- [ ] Task: Ensure consistent column alignment and styling across all three tables
- [ ] Task: Handle empty states (e.g., "No active orders") gracefully in the UI
- [ ] Task: Final verification of keyboard shortcuts and responsiveness
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Refinement & Polishing' (Protocol in workflow.md)
