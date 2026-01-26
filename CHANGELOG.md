# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
