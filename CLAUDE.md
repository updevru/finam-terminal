# Finam Terminal Project

## Project Overview

**finam-terminal** is a Go-based Terminal User Interface (TUI) application designed to interact with the Finam Trade API. It demonstrates how to authenticate, retrieve account information, and fetch market data (quotes, positions) using gRPC.

### Key Technologies
*   **Language:** Go (v1.24.1)
*   **API Protocol:** gRPC
*   **TUI Library:** `github.com/rivo/tview`
*   **Configuration:** `github.com/joho/godotenv`
*   **SDK:** `github.com/FinamWeb/finam-trade-api/go`

## Architecture

The project follows a clean modular structure:

*   **`main.go`**: The entry point. Handles configuration loading, API client initialization, and starting the UI loop.
*   **`api/`**: Contains the `Client` struct and methods for interacting with the Finam gRPC services. Encapsulates the complexity of the raw API calls.
*   **`ui/`**: Manages the Terminal User Interface.
    *   `app.go`: Main `App` struct, state management, tabbed view (Positions/History/Orders), and lifecycle (Run/Stop).
    *   `render.go` / `components.go`: Responsible for drawing UI elements (tables, lists, headers).
    *   `data.go`: Data fetching logic for trades history and active orders.
    *   `search.go`: Dedicated search window for finding securities.
*   **`config/`**: Handles loading environment variables from `.env` or system environment.
*   **`models/`**: Shared data structures used across the application to represent accounts, quotes, positions, trades, and orders. Key fields include `LotSize` and `Name` for instrument metadata.

## Getting Started

### Prerequisites
*   Go 1.24 or higher
*   Finam Trade API Token (obtain from Finam Developer Portal)

### Installation

1.  Clone the repository.
2.  Install dependencies:
    ```bash
    go mod tidy
    ```

### Configuration

The application requires an API token.

1.  Copy the example configuration:
    ```bash
    cp .env.example .env
    ```
2.  Edit `.env` and add your token:
    ```env
    FINAM_API_TOKEN=your_actual_token_here
    ```

### Building and Running

**Run directly:**
```bash
go run main.go
```

**Run with specific account (by index):**
```bash
go run main.go -account 0
```

**Build executable:**
```bash
go build -o finam-trade.exe main.go
./finam-trade.exe
```

## Development Conventions

*   **Style:** Standard Go formatting (`gofmt`).
*   **Logging:** Use standard `log` package with prefixes like `[INFO]` and `[ERROR]`.
*   **UI Updates:** The TUI is event-driven. Ensure UI updates happen on the main thread or using `app.QueueUpdateDraw` (implied by `tview` usage).
*   **Configuration:** Always use `config.Load()` to access settings; do not hardcode credentials.

## Directory Structure

*   `api/`: gRPC client wrapper.
*   `config/`: Configuration loader.
*   `models/`: Data types.
*   `ui/`: TUI implementation (views, controllers).
*   `.env`: Local configuration (git-ignored).

## API Implementation Details

### Retrieving Security Prices

1.  **Market Data (Quotes)**
    *   **Service:** `MarketDataServiceClient`
    *   **Method:** `LastQuote`
    *   **File:** `api/client.go` (`GetQuotes`)
    *   **Key Field:** `Last` (Last trade price)
    *   **Usage:** Ticker lookup, general price checks.

2.  **Security Search**
    *   **Service:** `InstrumentsServiceClient`
    *   **Method:** `GetSecurities`
    *   **File:** `api/client.go` (`SearchSecurities`)
    *   **Usage:** Finding assets by ticker or name.

3.  **Portfolio Positions**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetAccount`
    *   **File:** `api/client.go` (`GetAccountDetails`)
    *   **Key Field:** `CurrentPrice` (Broker's valuation price)
    *   **Usage:** Calculating equity, PnL, and position value. Positions are enriched with `LotSize` and human-readable `Name` from the instrument cache.

4.  **Trade History**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetTradeHistory`
    *   **File:** `api/client.go` (`GetTradeHistory`)
    *   **Usage:** Fetching completed trades for display in the History tab.

5.  **Active Orders**
    *   **Service:** `AccountsServiceClient`
    *   **Method:** `GetOrders`
    *   **File:** `api/client.go` (`GetActiveOrders`)
    *   **Usage:** Fetching pending/active orders for display in the Orders tab.

6.  **Instrument Name Cache**
    *   **File:** `api/client.go` (`InstrumentCache`, `GetInstrumentName`, `UpdateInstrumentCache`)
    *   **Usage:** Centralized O(1) cache mapping ticker symbols to human-readable names. Populated during asset loading and search operations.

# Conductor Context

If a user mentions a "plan" or asks about the plan, and they have used the conductor extension in the current session, they are likely referring to the `conductor/tracks.md` file or one of the track plans (`conductor/tracks/<track_id>/plan.md`).

## Universal File Resolution Protocol

**PROTOCOL: How to locate files.**
To find a file (e.g., "**Product Definition**") within a specific context (Project Root or a specific Track):

1.  **Identify Index:** Determine the relevant index file:
    -   **Project Context:** `conductor/index.md`
    -   **Track Context:**
        a. Resolve and read the **Tracks Registry** (via Project Context).
        b. Find the entry for the specific `<track_id>`.
        c. Follow the link provided in the registry to locate the track's folder. The index file is `<track_folder>/index.md`.
        d. **Fallback:** If the track is not yet registered (e.g., during creation) or the link is broken:
            1. Resolve the **Tracks Directory** (via Project Context).
            2. The index file is `<Tracks Directory>/<track_id>/index.md`.

2.  **Check Index:** Read the index file and look for a link with a matching or semantically similar label.

3.  **Resolve Path:** If a link is found, resolve its path **relative to the directory containing the `index.md` file**.
    -   *Example:* If `conductor/index.md` links to `./workflow.md`, the full path is `conductor/workflow.md`.

4.  **Fallback:** If the index file is missing or the link is absent, use the **Default Path** keys below.

5.  **Verify:** You MUST verify the resolved file actually exists on the disk.

**Standard Default Paths (Project):**
- **Product Definition**: `conductor/product.md`
- **Tech Stack**: `conductor/tech-stack.md`
- **Workflow**: `conductor/workflow.md`
- **Product Guidelines**: `conductor/product-guidelines.md`
- **Tracks Registry**: `conductor/tracks.md`
- **Tracks Directory**: `conductor/tracks/`

**Standard Default Paths (Track):**
- **Specification**: `conductor/tracks/<track_id>/spec.md`
- **Implementation Plan**: `conductor/tracks/<track_id>/plan.md`
- **Metadata**: `conductor/tracks/<track_id>/metadata.json`
