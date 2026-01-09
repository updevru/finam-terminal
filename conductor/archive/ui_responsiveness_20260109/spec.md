# Specification: UI Responsiveness & Error Handling

## 1. Overview
Currently, the application freezes when Finam Trade API calls time out or fail, blocking the main UI thread. This track aims to refactor the data fetching layer to be asynchronous, ensuring the UI remains responsive at all times. Additionally, it will introduce a visual feedback mechanism (Status Bar) to inform the user of loading states and errors.

## 2. Functional Requirements
- **Non-Blocking UI:** The Terminal User Interface (TUI) MUST remain fully responsive (navigable, reactive to keys) during all network operations.
- **Loading Indicator:** A global status indicator (e.g., in the bottom status bar) MUST display a visual cue (text or spinner) whenever a background data fetch is in progress.
- **Error Feedback:** If a data fetch fails (e.g., timeout, network error), the application MUST:
    - Log the error internally.
    - Display a user-friendly error message in the status bar (e.g., "Connection lost", "Data fetch failed").
    - **Crucial:** The application MUST NOT panic or crash due to closed channels or unhandled API errors.

## 3. Technical Strategy
- **Concurrency:** Move all blocking gRPC calls (Quotes, Portfolio, etc.) into separate goroutines.
- **UI Synchronization:** Use `tview.Application.QueueUpdateDraw` to safely update UI components from background goroutines.
- **Panic Prevention:** Audit and fix the race condition causing the `panic: close of closed channel` error observed in the logs.

## 4. Acceptance Criteria
1.  **Responsiveness:** User can switch tabs or highlight items in the list *while* the application is simulating a slow network request (e.g., 5-second delay).
2.  **Visual Feedback:** "Loading..." status appears immediately when a request starts and disappears when it finishes.
3.  **Error Handling:** Disconnecting the network and triggering a refresh results in a "Network Error" message in the UI, not a frozen screen or crash.
