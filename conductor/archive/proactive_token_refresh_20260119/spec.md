# Specification: Proactive Token Refresh (Bug Fix)

## Overview
Currently, the application fails with a "Token is expired" error (Unauthenticated) during long-running sessions because the initial JWT token is not refreshed. This track implements a proactive background process to automatically refresh the authentication token before it expires.

## Functional Requirements
1.  **Secret Persistence:** Store the `apiToken` (secret) within the `Client` struct to enable automated re-authentication.
2.  **JWT Expiry Detection:**
    -   Parse the JWT token payload (split by `.` and base64 decode) to extract the `exp` (expiration) claim.
    -   Calculate the refresh time based on the `exp` claim (e.g., 2 minutes before actual expiry).
    -   **Fallback:** If the `exp` claim is missing or invalid, default to a refresh interval of 10 minutes.
3.  **Background Refresh Process:**
    -   Launch a dedicated goroutine when the `Client` is initialized.
    -   The goroutine will wait for the calculated interval and then call the `authenticate` method.
    -   Use a `context.Context` (derived from a `context.WithCancel`) to ensure the goroutine terminates when `Client.Close()` is called.
4.  **Status Tracking & Logging:**
    -   Add a `LastRefresh` timestamp to the `Client` struct.
    -   Log successful refreshes with `[INFO] Token refreshed successfully`.
    -   Log failed refresh attempts with `[ERROR] Token refresh failed: <error>` and implement a retry mechanism (e.g., retry after 30 seconds).

## Non-Functional Requirements
-   **Reliability:** The refresh process must be resilient to transient network failures.
-   **Resource Efficiency:** The background goroutine should remain idle most of the time, consuming negligible CPU and memory.

## Acceptance Criteria
-   [ ] The application can run for significantly longer than 1 hour without encountering a `Token is expired` error.
-   [ ] Logs verify that token refreshes are occurring as expected.
-   [ ] Closing the client successfully stops the background refresh process without leaving orphaned goroutines.

## Out of Scope
-   Refreshing other types of credentials if any are added in the future.
-   Implementing a full-blown JWT validation library (simple manual payload parsing is preferred to keep dependencies minimal).
