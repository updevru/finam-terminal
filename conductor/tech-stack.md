# Technology Stack - Finam Trade TUI

## Core Technologies
- **Programming Language:** Go (v1.24.1)
- **API Protocol:** gRPC (Google Remote Procedure Call) for high-performance communication with Finam services.
- **TUI Library:** `github.com/rivo/tview` - A rich terminal UI library for building the application's interface.
- **TUI Base:** `github.com/gdamore/tcell/v2` - The underlying terminal handling library used by tview.

## Dependencies & Frameworks
- **Finam SDK:** `github.com/FinamWeb/finam-trade-api/go` - Official Finam Trade API client for Go.
- **Configuration:** `github.com/joho/godotenv` - For loading environment variables from `.env` files.
- **gRPC/Proto:** 
    - `google.golang.org/grpc` - Core gRPC implementation.
    - `google.golang.org/genproto` - Generated protocol buffer definitions.

## Project Management
- **Dependency Management:** Go Modules (`go.mod`, `go.sum`).
- **Environment Management:** `.env` file for local development configuration.
