# Implementation Plan - Fix Zero Max Quantity Bug

## Phase 1: Investigation
- [x] Analyze `ui/render.go` to understand how the positions table is built (header rows, sorting, etc.).
- [x] Analyze `ui/app.go` (`OpenCloseModal`) to verify the row-to-index logic.
- [x] Debug/Verify `formatDecimal` and `strconv.ParseFloat` behavior with likely inputs.

## Phase 2: Fix
- [x] **If Index Mismatch:** Implement a reliable way to map table selection to the position model (e.g., store ID/Symbol in the table cell reference or user object). -> *Issue was parsing/validation.*
- [x] **If Parsing Error:** Fix the parsing logic or the `formatDecimal` function. -> *Implemented `parseFloat` helper handling commas.*

## Phase 3: Verification
- [x] Create a test case simulating the table selection and modal opening with specific data. -> *Verified via unit tests and logic check.*
- [x] Manual verification.
