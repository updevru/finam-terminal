# Implementation Plan: Security Search Window (S-Key)

## Phase 1: API & Data Model Enhancements [ ]
- [ ] Task: Update `models/models.go` to include `SecurityInfo` for search results (Ticker, Name, Lot, Currency).
- [ ] Task: Implement `SearchSecurities` in `api/client.go` to wrap the assets search API.
- [ ] Task: Implement `GetSnapshots` in `api/client.go` to fetch initial prices for search results.
- [ ] Task: Write unit tests for `SearchSecurities` and `GetSnapshots` in `api/client_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: API & Data Model Enhancements' (Protocol in workflow.md)

## Phase 2: Search UI Component [ ]
- [ ] Task: Create `ui/search.go` and define `SearchModal` struct and layout.
- [ ] Task: Implement basic rendering of the search window with input field and empty table.
- [ ] Task: Implement debounced input handling (~300ms) to trigger search.
- [ ] Task: Implement `Tab` key navigation between search input and results table.
- [ ] Task: Write unit tests for `SearchModal` navigation and debouncing in `ui/search_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Search UI Component' (Protocol in workflow.md)

## Phase 3: Integration & Live Data [ ]
- [ ] Task: Integrate 'S' key in `ui/app.go` to open the `SearchModal`.
- [ ] Task: Implement live market data updates for the visible search results in `ui/search.go`.
- [ ] Task: Implement 'A' key handler in `SearchModal` to trigger `OrderModal` with selected security.
- [ ] Task: Write integration tests for opening Search and transitioning to Buy Modal in `ui/search_integration_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration & Live Data' (Protocol in workflow.md)

## Phase 4: Refinement & Final Polish [ ]
- [ ] Task: Polish search result formatting (colors for change %, column alignment).
- [ ] Task: Add error handling/loading indicators for search operations.
- [ ] Task: Final code review and documentation update (GoDoc).
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Refinement & Final Polish' (Protocol in workflow.md)
