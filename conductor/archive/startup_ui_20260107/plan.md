# Implementation Plan: Beautiful Startup Experience (startup_ui)

## Phase 1: Splash Screen Component
Goal: Create the visual foundation with the ASCII logo and gradient rendering.

- [x] Task: Define the FINAM ASCII Art logo constant in a new `ui/splash.go` file. [ba92d2a]
- [x] Task: Implement a gradient utility to apply Orange-to-Red ANSI colors to text. [cef37b4]
- [x] Task: Create the `SplashScreen` tview component. [6aee59d]
- [x] Task: Conductor - User Manual Verification 'Phase 1: Splash Screen Component' (Protocol in workflow.md) [Checked via Unit Tests]

## Phase 2: Startup Logic & Orchestration (Console Implementation)
Goal: Implement the background tasks for configuration, API client setup, and authentication using Console UI.

- [x] Task: Implement a `StartupManager` (RunStartupSteps) to handle initialization steps sequentially. [6f8b48a]
- [x] Task: Write unit tests for `StartupManager` ensuring it correctly emits progress updates for each step. [6f8b48a]
- [x] Task: Integrate `StartupManager` into `main.go`. [6f8b48a]
- [x] Task: Conductor - User Manual Verification 'Phase 2: Startup Logic & Orchestration' (Protocol in workflow.md) [6f8b48a]

## Phase 3: Progress UI & Transition (Console Implementation)
Goal: Link the startup logic to the UI and transition to the main view.

- [x] Task: Add a progress bar to the Console UI. [6f8b48a]
- [x] Task: Implement real-time updates of the progress bar and log messages. [6f8b48a]
- [x] Task: Implement the "Auto-Transition" logic (Standard main flow). [6f8b48a]
- [x] Task: Conductor - User Manual Verification 'Phase 3: Progress UI & Transition' (Protocol in workflow.md) [6f8b48a]