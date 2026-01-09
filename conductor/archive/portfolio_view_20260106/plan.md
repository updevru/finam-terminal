# Plan: Portfolio View

## Phase 1: Data Model and API Integration [checkpoint: 27a9e5b]
- [x] Task: Verify `models.AccountInfo` and `models.Position` cover all required fields for the UI. [9adc012]
- [x] Task: Write unit tests for `Client.GetAccounts` and `Client.GetAccountDetails` using mocks if possible. [993346c]
- [x] Task: Conductor - User Manual Verification 'Phase 1: Data Model and API Integration' (Protocol in workflow.md) [27a9e5b]

## Phase 2: UI Component Development [checkpoint: 23902a2]
- [x] Task: Create a new `PortfolioView` component in `ui/components.go`. [b066ac6]
- [x] Task: Implement a table for the Account List. [5f5be82]
- [x] Task: Implement a summary area for Account Details (Equity, PnL). [0839c36]
- [x] Task: Implement a table for Positions. [1322497]
- [x] Task: Conductor - User Manual Verification 'Phase 2: UI Component Development' (Protocol in workflow.md) [23902a2]

## Phase 3: Application Integration [checkpoint: 6f58cd7]
- [x] Task: Integrate `PortfolioView` into the main application layout in `ui/app.go`. [a087a49]
- [x] Task: Implement the data fetching logic to populate the Portfolio view on startup or selection. [58473d3]
- [x] Task: Implement keyboard shortcuts to switch to the Portfolio view. [522f5d2]
- [x] Task: Conductor - User Manual Verification 'Phase 3: Application Integration' (Protocol in workflow.md) [6f58cd7]
