# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.10.0] - 2026-02-24

### Added
- **Instrument Profile**: Full-screen instrument profile overlay opened via Enter on positions or P on search results, displaying asset details, trading parameters, quotes, trading schedule, and a Unicode candlestick chart with switchable timeframes (M5/H1/D/W) (`instrument_profile`).
- **Candlestick Chart**: Unicode-based price chart with smart time labels on X-axis and support for multiple timeframes (`instrument_profile`).

### Fixed
- **Local Timezone**: All dates in History and Orders tables now display in the user's local timezone instead of UTC (`local_timezone_dates`).
- **Code Formatting**: Fixed `gofmt` formatting across all Go source files.

## [v0.9.0] - 2026-02-13

### Added
- **Portfolio Tabs**: Tabbed interface within the Positions window with History (trade history) and Orders (pending orders) views, switchable via arrow keys and Tab (`portfolio_tabs`).
- **Lot-Based Trading**: Quantities displayed in lots across Positions, History, and Orders tables; lot-based input in Buy/Close modals with real-time cost calculation and lot size display (`lot_based_trading`).
- **Human-Readable Names**: Descriptive instrument names (e.g., "Sberbank" instead of "SBER") displayed across all tables and modal titles, with automatic caching and fallback to ticker symbols (`human_readable_names`).

## [v0.8.1] - 2026-02-04

### Added
- **Security Search**: Dedicated full-width search window for finding assets and initiating orders (`security_search`).

## [v0.8.0] - 2026-01-26

### Added
- **Community Health**: Added `CONTRIBUTING.md` with detailed development guidelines.
- **Community Health**: Added `LICENSE` file (Apache 2.0).
- **Documentation**: Added `CHANGELOG.md` to track project history.
- **Documentation**: Added status badges (CI, Go Report, License, Version) to `README.md`.
- **Documentation**: Added "Development with Gemini and Conductor" section to `README.md`.

## [v0.7.0] - 2026-01-26

### Added
- **Portfolio View**: Comprehensive view of current portfolio holdings (`portfolio_view`).
- **Order Placement**: Ability to place market and limit orders (`order_placement`).
- **Position Closing**: Dedicated modal and logic for closing existing positions (`close_position`).
- **Startup Wizard**: Interactive initial setup and UI for API token configuration (`startup_setup`, `startup_ui`).
- **Token Management**: Proactive token refresh to maintain session validity (`proactive_token_refresh`).
- **UI Layout**: Full-width positions table for better visibility (`full_width_positions_table`).
- **CI/CD**: GitHub Actions pipeline for automated testing and builds (`github_actions_pipeline`).

### Changed
- **UI Responsiveness**: Improved interface adaptation to terminal resizing (`ui_responsiveness`).
- **UX**: Enhanced quantity input handling in order forms (`improve_qty_input`).
- **Filtering**: Automatically filter out positions with zero quantity (`filter_zero_positions`).
