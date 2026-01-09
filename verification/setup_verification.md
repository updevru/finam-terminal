# Manual Verification: Startup Setup

## Objective
Verify the automatic startup setup screen logic, token validation, and persistence.

## Steps

### 1. Clean Environment
Ensure no API token is configured.
```powershell
# Windows PowerShell
Remove-Item -Force "$HOME\.finam-cli\.env" -ErrorAction SilentlyContinue
Remove-Item -Force ".env" -ErrorAction SilentlyContinue
```

### 2. Launch Application
Run the application.
```bash
go run main.go
```
**Expected Result:** The application should NOT crash. It should display the **Finam Terminal Setup** screen with instructions and an input field.

### 3. Verify UI Elements
- Header: "Finam Terminal" logo.
- Welcome Text: "Welcome to Finam Terminal!"
- Instructions: Links to open accounts and get token.
- Input: "API Token: " field.
- Button: "Save & Continue".

### 4. Test Validation (Negative)
- Enter `invalid_token`.
- Press Enter or click Save.
- **Expected Result:** Error message in red (e.g., "Client init error" or "Validation failed").

### 5. Test Validation (Positive)
- Enter a valid Finam Trade API token.
- Press Enter.
- **Expected Result:**
    - "Validating token..." status.
    - Setup screen closes.
    - Application proceeds to "Validating configuration...".
    - Main TUI (Portfolio/Accounts view) loads.

### 6. Verify Persistence
- Exit the application (Ctrl+C).
- Run `go run main.go` again.
- **Expected Result:** The Setup Screen is SKIPPED. The application launches directly into the main TUI.

### 7. Verify File Creation
- Check existence of token file.
```powershell
Test-Path "$HOME\.finam-cli\.env"
```
- **Expected Result:** True.
