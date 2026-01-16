# Product Guide - Finam Terminal TUI

## Initial Concept
A Go-based Terminal User Interface (TUI) application designed to interact with the Finam Trade API via gRPC. It serves as both a practical tool for traders and a reference implementation for developers.

## Target Audience
- **Developers:** Those seeking a robust reference implementation for interacting with the Finam Trade gRPC API using Go.
- **Traders:** Individuals who prefer a lightweight, keyboard-centric terminal environment for real-time market monitoring and portfolio management.

## Primary Goals
- **API Reference:** Demonstrate best practices for integrating the Finam Trade API with Go, focusing on efficient gRPC communication and robust data handling.
- **Functional Monitoring:** Provide a high-performance TUI for real-time tracking of market quotes, order books, and personal account portfolios.

## Key Features
- **Startup Experience:**
    - **First-Run Setup:** Automated, guided setup screen for API token configuration if not detected.
    - **Branded Splash Screen:** Large, gradient-colored "FINAM" logo on launch.
    - **Visual Progress:** Detailed initialization log with a progress bar for configuration and network steps.
- **Order Execution:**
    - **Quick Order Entry:** Press 'A' to open a context-aware order entry modal.
    - **Smart Pre-filling:** Automatically detects the selected instrument from the portfolio view.
    - **Validation:** Real-time client-side validation of order parameters.
    - **Live Feedback:** Instant confirmation or error handling for submitted orders.
    - **Position Closing:**
        - **One-Key Action:** Press 'C' on any open position to initiate a close order.
        - **Safety Modal:** Confirmation dialog displaying current price, PnL, and estimated total before execution.
- **Portfolio View:** 
    - **Account Selection:** Interactive list of available accounts with real-time equity and status.
    - **Account Details:** Summary area showing Account ID, Type, Status, Equity, and Unrealized PnL.
    - **Positions Table:** Detailed view of current positions including Symbol, Quantity, Average/Current Price, and PnL.
 - **Interactive TUI:** A responsive interface built with `tview`, featuring intuitive keyboard shortcuts for rapid navigation and data filtering.
    - **Live Status Feedback:** Bottom status bar indicating data loading states, network health, and detailed error messages without interrupting workflow.
## Non-Functional Requirements
- **Performance:** Optimized for low-latency market data updates and smooth UI responsiveness.
- **Cross-Platform:** Full compatibility across Windows, macOS, and Linux terminal environments.

## Visual Identity & UX
- **High-Density Minimalist:** A layout designed to maximize information density while maintaining clarity, ensuring key metrics are visible at a glance.
- **Modern & Dynamic:** Uses a rich color palette to visually distinguish price movements, trends, and different data categories.
- **Menu-Driven Navigation:** An intuitive structure with clear headers and a built-in help system to minimize the learning curve.
