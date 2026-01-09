# Plan: UI Responsiveness & Error Handling

Refactor the TUI data fetching layer to be asynchronous and provide visual feedback to the user, preventing interface freezes during network timeouts.

## Phase 1: Robust Lifecycle & Panic Prevention [checkpoint: a3d4b25]
Fix the immediate panic and ensure the application lifecycle is managed safely.

- [x] Task: Fix `Stop()` panic in `ui/app.go`
    - Sub-task: Use `sync.Once` to ensure `close(a.stopChan)` and `a.app.Stop()` are only called once.
- [x] Task: Verify Lifecycle Fix
    - Sub-task: Create a unit test `ui/app_lifecycle_test.go` to simulate multiple `Stop()` calls.
- [x] Task: Conductor - User Manual Verification 'Phase 1: Robust Lifecycle & Panic Prevention' (Protocol in workflow.md)

## Phase 2: Asynchronous Data Fetching [checkpoint: f4922be]
Refactor `loadData` and `backgroundRefresh` to move network I/O out of the main UI thread.

- [x] Task: Refactor `loadData` for non-blocking execution
    - Sub-task: Create `loadDataAsync` which runs the fetch in a goroutine and updates the UI via `QueueUpdateDraw` only *after* data is received.
- [x] Task: Fix `backgroundRefresh` in `ui/data.go`
    - Sub-task: Move `a.loadData(acc.ID)` OUT of the `QueueUpdateDraw` closure.
    - Sub-task: Ensure the background loop only triggers UI updates when data changes or at the end of a fetch.
- [x] Task: Update input handlers to use async fetching
    - Sub-task: Refactor `refresh` and `switchAccount` in `ui/input.go` to use the new async mechanism.
- [x] Task: Conductor - User Manual Verification 'Phase 2: Asynchronous Data Fetching' (Protocol in workflow.md)

## Phase 3: Visual Feedback & Error Handling [checkpoint: ]
Implement the status bar indicators and friendly error messages.

- [x] Task: Enhance Status Bar UI
    - Sub-task: Update `updateStatusBar` to accept an optional status message/type (Loading, Error, Success).
    - Sub-task: Implement a "loading" state that displays a spinner or "Updating..." text.
- [x] Task: Implement Error Propagation
    - Sub-task: Update `loadData` to return errors or send them to a dedicated UI error handler.
    - Sub-task: Ensure gRPC timeouts (DeadlineExceeded) are caught and displayed as "Connection Timeout" instead of just logging.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Visual Feedback & Error Handling' (Protocol in workflow.md)

## Phase 4: Final Polishing & Verification [checkpoint: 13d9e57]
Optimization and end-to-end testing.

- [x] Task: Optimize Refresh Strategy
    - Sub-task: Prioritize refreshing the *active* account more frequently than background accounts.
- [x] Task: End-to-End Responsiveness Test
    - Sub-task: Simulate a 10-second API delay and verify that the UI remains interactive (switching accounts, scrolling).
- [x] Task: Conductor - User Manual Verification 'Phase 4: Final Polishing & Verification' (Protocol in workflow.md)
