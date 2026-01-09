# Specification: Beautiful Startup Experience (Track: startup_ui)

## Overview
This track implements a visually appealing and informative startup sequence for the Finam Terminal TUI. It aims to improve the "first-run" experience by providing a branded splash screen and a clear progress indicator during the initialization and authentication phases.

## Functional Requirements
1.  **Splash Screen Logo:**
    -   Display a large "FINAM" logo using static ASCII art.
    -   Apply a horizontal/vertical gradient to the logo transition from Orange to Red, mimicking the aesthetics of the Gemini CLI.
2.  **Initialization Log:**
    -   Display a series of status messages below the logo as the application starts:
        -   "Loading configuration..."
        -   "Initializing API client..."
        -   "Authenticating with Finam..."
        -   "Fetching account list..."
        -   "Checking market data connection..."
3.  **Progress Indicator:**
    -   Implement a visual progress bar that fills incrementally as each of the five initialization steps completes.
4.  **Automatic Transition:**
    -   Once all steps are successful, the splash screen should automatically transition into the main Portfolio view.

## Non-Functional Requirements
-   **Responsiveness:** The startup sequence should not hang the UI. API calls must be performed asynchronously while the progress bar updates.
-   **Error Handling:** If a step fails (e.g., authentication), the progress bar should stop, and a clear error message should be displayed with an option to exit.

## Acceptance Criteria
- [ ] On launch, the "FINAM" ASCII logo appears with an Orange-to-Red gradient.
- [ ] Five distinct startup steps are listed sequentially.
- [ ] A progress bar fills from 0% to 100% as the steps complete.
- [ ] The application successfully enters the main UI after 100% completion.

## Out of Scope
- Interactive login (entering token manually).
- Detailed "retry" logic for every single network failure.
