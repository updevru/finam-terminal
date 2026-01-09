# Specification: Startup API Key Setup (`startup_setup`)

## Overview
Currently, the application requires a `FINAM_API_TOKEN` to be present in a `.env` file to function. If missing, the app likely fails or requires manual file editing. This feature introduces a user-friendly startup experience for first-time users, guiding them to obtain their API key and allowing them to enter and validate it directly within the TUI.

## Functional Requirements
- **First-Run Detection:** On startup, the application must check for the existence of an API key in standard locations.
- **Dedicated Setup Screen:** If no key is found, a dedicated screen must be displayed providing:
    - Instructions on how to obtain a Finam Trade API token (including the URL: https://tradeapi.finam.ru/docs/about/).
    - An input field for the token.
- **Input Features:**
    - **Clipboard Support:** Support for pasting the token from the terminal clipboard.
- **Live Validation:** Before saving, the application must attempt to initialize a Finam API client with the provided token to verify its validity.
- **Configuration Storage:**
    - **Read Priority:** 1) `~/.finam-cli/.env`, 2) Local `./.env`.
    - **Creation:** If no key is found, the application will create and save the token to `~/.finam-cli/.env` (and create the directory if it doesn't exist).

## Non-Functional Requirements
- **Security:** Ensure the token is handled securely in memory (though it will be stored in plain text in `.env` as per existing project conventions).
- **UX:** Clear error messaging if the token validation fails.

## Acceptance Criteria
- [ ] Application starts successfully without a `.env` file and shows the Setup Screen.
- [ ] User can paste a token into the input field.
- [ ] Invalid tokens are rejected with a clear error message.
- [ ] Valid tokens are saved to `~/.finam-cli/.env`.
- [ ] Subsequent launches with the saved token skip the Setup Screen and proceed to the main app.

## Out of Scope
- Support for multiple accounts/keys in this phase.
- Encrypted storage of the API token.
