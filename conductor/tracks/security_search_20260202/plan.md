# Implementation Plan: Security Search Window (S-Key)

## Phase 1: API & Data Model Enhancements [x] [checkpoint: a55d3f1]
- [x] Task: Update `models/models.go` to include `SecurityInfo` for search results (Ticker, Name, Lot, Currency). a299c7e
- [x] Task: Implement `SearchSecurities` in `api/client.go` to wrap the assets search API. 16d23a0
- [x] Task: Implement `GetSnapshots` in `api/client.go` to fetch initial prices for search results. 16d23a0
- [x] Task: Write unit tests for `SearchSecurities` and `GetSnapshots` in `api/client_test.go`. 16d23a0
- [x] Task: Conductor - User Manual Verification 'Phase 1: API & Data Model Enhancements' (Protocol in workflow.md) a55d3f1

## Phase 2: Search UI Component [x] [checkpoint: c013d66]
- [x] Task: Create `ui/search.go` and define `SearchModal` struct and layout. bbd2d29
- [x] Task: Implement basic rendering of the search window with input field and empty table. 377d524
- [x] Task: Implement debounced input handling (~300ms) to trigger search. 377d524
- [x] Task: Implement `Tab` key navigation between search input and results table. 377d524
- [x] Task: Write unit tests for `SearchModal` navigation and debouncing in `ui/search_test.go`. 377d524
- [x] Task: Conductor - User Manual Verification 'Phase 2: Search UI Component' (Protocol in workflow.md) c013d66

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
