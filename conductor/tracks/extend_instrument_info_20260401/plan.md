# Plan: Extended Instrument Info & Open Interest (Trade API 2.14.0)

## Overview
Update the Finam Trade API SDK dependency, extend models and API client to extract new instrument-type-specific fields (futures, options, bonds) and open interest, then display them contextually on the instrument profile screen.

## Phase 1: SDK Update
- [x] Task: Update `github.com/FinamWeb/finam-trade-api/go` to latest version
  - Run `go get github.com/FinamWeb/finam-trade-api/go@latest && go mod tidy`
  - Acceptance: `go build ./...` succeeds with the new dependency

## Phase 2: Model Extensions
- [x] Task: Add derivative/bond fields to `models.AssetDetails`
  - Add `ContractSize string` — formatted contract size (futures, options)
  - Add `Strike string` — formatted strike price (options only)
  - Add `BondFaceValue string` — formatted face value (bonds only)
  - Add `BondFaceCurrency string` — currency of face value (bonds only)
  - Acceptance: Model compiles, fields available for use
- [x] Task: Add `OpenInterest string` field to `models.Quote`
  - Acceptance: Model compiles, field available for use

## Phase 3: API Client — Parse New Fields
- [x] Task: Extract FutureDetails/OptionDetails/BondDetails in `GetAssetInfo()`
  - Use `resp.GetFutureDetails()`, `resp.GetOptionDetails()`, `resp.GetBondDetails()`
  - For futures: set `ContractSize`, override `ExpirationDate` from `Timestamp`
  - For options: set `ContractSize`, `Strike`, override `ExpirationDate` from `Timestamp`
  - For bonds: set `BondFaceValue`, `BondFaceCurrency`
  - Files: `api/client.go` (~line 1176)
  - Acceptance: New fields populated when API returns type-specific data
- [x] Task: Extract `OpenInterest` in `GetQuotes()`
  - Use `q.GetOpenInterest()` and `formatDecimal()`
  - File: `api/client.go` (~line 792)
  - Acceptance: `Quote.OpenInterest` populated from API response

## Phase 4: UI — Display New Fields
- [x] Task: Add instrument-type-specific section to profile info panel
  - In `ui/profile.go` `renderInfoPanel()`, after the Details section:
    - If `ContractSize` is set → show "Contract" field
    - If `Strike` is set → show "Strike" field
    - If `BondFaceValue` is set → show "Face Value" with currency
  - Use existing `writeField()` helper
  - Section header: contextual (e.g., "─── Futures ───", "─── Options ───", "─── Bond ───")
  - Acceptance: Futures/options/bonds show extra section; stocks unchanged
- [x] Task: Show Open Interest in Quote section
  - In `renderInfoPanel()` Quote section, add `OpenInterest` field
  - Only show when value is non-empty (derivatives)
  - Acceptance: OI visible for derivatives, hidden for stocks

## Phase 5: Verification
- [x] Task: Build and verify
  - Run `go build ./...`
  - Manual verification: open profile for a stock, future, option, bond
  - Acceptance: Application compiles and runs correctly
