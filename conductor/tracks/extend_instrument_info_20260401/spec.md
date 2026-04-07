# Spec: Extended Instrument Info & Open Interest (Trade API 2.14.0)

## Problem
The Finam Trade API SDK has been updated with new fields for derivatives and bonds in `GetAssetResponse`, and `open_interest` in Quote/Trade messages. The terminal currently doesn't display this instrument-specific data, losing valuable context for futures, options, and bond traders.

## Solution
1. Update the `finam-trade-api/go` SDK to the latest version.
2. Parse and display instrument-type-specific details (futures, options, bonds) on the profile screen.
3. Extract and display open interest from quote data for derivatives.

## Requirements

### Functional
- **Futures profile** shows: expiration date, contract size
- **Options profile** shows: expiration date, contract size, strike price
- **Bonds profile** shows: face value, face value currency
- **Stocks** — no changes to the profile screen
- **Open Interest** displayed in the Quote section of the profile for futures and options
- All new fields are conditionally rendered based on instrument type

### Technical
- Update `github.com/FinamWeb/finam-trade-api/go` to latest (`v0.0.0-20260401112026-402e726d2b7f`)
- Use `GetAssetResponse.GetFutureDetails()`, `GetOptionDetails()`, `GetBondDetails()` oneof accessors
- Use `Quote.GetOpenInterest()` for open interest data
- Extend `models.AssetDetails` with new fields: `ContractSize`, `Strike`, `BondFaceValue`, `BondFaceCurrency`
- Extend `models.Quote` with `OpenInterest` field
- New fields use `formatDecimal()` for consistent formatting

## Acceptance Criteria
- [ ] SDK updated to latest version, `go mod tidy` passes
- [ ] Futures instruments show expiration date and contract size in profile
- [ ] Options instruments show expiration date, contract size, and strike in profile
- [ ] Bond instruments show face value and face value currency in profile
- [ ] Stock instruments profile unchanged
- [ ] Open interest shown in Quote section for derivatives
- [ ] Open interest hidden for non-derivative instruments
- [ ] Application compiles and runs without errors

## Edge Cases
- `GetFutureDetails()` / `GetOptionDetails()` / `GetBondDetails()` returns nil for non-matching types — handled via nil checks
- `OpenInterest` field may be nil even for derivatives — show nothing or "N/A"
- `ContractSize`, `Strike`, `BondFaceValue` may be nil — show "N/A"
- Existing `ExpirationDate` field in `AssetDetails` is populated from old API field; new details provide a `Timestamp`-typed expiration — prefer the new typed field when available

## Dependencies
- `github.com/FinamWeb/finam-trade-api/go` latest version
