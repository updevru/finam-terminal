# Implementation Plan - Startup API Key Setup (`startup_setup`)

## Phase 1: Configuration Management [checkpoint: 14412ed]
- [x] Task: Implement `config.FindToken()` to search `~/.finam-cli/.env` and `./.env`. ce11d5a
- [x] Task: Implement `config.SaveTokenToUserHome(token string)` to create `~/.finam-cli/.env`. ce11d5a
- [x] Task: Update `config.Load()` to utilize these functions and handle missing tokens gracefully. ce11d5a
- [x] Task: Write unit tests for the new configuration functions in `config/config_test.go`. ce11d5a
- [x] Task: Conductor - User Manual Verification 'Configuration Management' (Protocol in workflow.md)

## Phase 2: Setup UI Component [checkpoint: 9ba6a18]
- [x] Task: Create `ui/setup.go` containing a `SetupApp` struct that uses `tview` for the setup screen. 5898a52
- [x] Task: Design the `SetupView` with instructions (including the URL https://tradeapi.finam.ru/docs/about/) and an input field for the token. 409c918
- [x] Task: Implement token validation logic within `SetupApp` using `api.NewClient`. cebb096
- [x] Task: Implement saving the token and transitioning out of the setup view on success. cebb096
- [x] Task: Conductor - User Manual Verification 'Setup UI Component' (Protocol in workflow.md)
- [ ] Task: Implement saving the token and transitioning out of the setup view on success.
- [ ] Task: Conductor - User Manual Verification 'Setup UI Component' (Protocol in workflow.md)

## Phase 3: Main Integration & Flow [checkpoint: a07d45a]
- [x] Task: Refactor `main.go` to launch `SetupApp` if the token is missing after the initial configuration load. 7ee647a
- [x] Task: Ensure that after `SetupApp` completes successfully, the main application proceeds with the newly saved token. 7ee647a
- [x] Task: Conductor - User Manual Verification 'Main Integration & Flow' (Protocol in workflow.md)
- [ ] Task: Conductor - User Manual Verification 'Main Integration & Flow' (Protocol in workflow.md)

## Phase 4: Refinement and Documentation [checkpoint: e7de490]
- [x] Task: Update `README.md` with information about the new automatic setup and the token storage location. fd88a1f
- [x] Task: Add integration tests or final manual verification steps. 427af38
- [x] Task: Conductor - User Manual Verification 'Refinement and Documentation' (Protocol in workflow.md)
