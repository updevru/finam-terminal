package models

import (
	"strconv"
	"strings"
	"time"
)

// AccountInfo represents account information from Finam API
type AccountInfo struct {
	ID            string
	Type          string
	Status        string
	Equity        string
	UnrealizedPnL string
	OpenDate      time.Time
}

// Position represents a trading position
type Position struct {
	Symbol        string
	Ticker        string
	Name          string
	MIC           string
	LotSize       float64
	Quantity      string
	AveragePrice  string
	CurrentPrice  string
	DailyPnL      string
	UnrealizedPnL string
	TotalValue    string
}

// GetCloseDirection returns the inverse direction needed to close the position.
// Returns "Sell" for Long positions (>0), "Buy" for Short positions (<0),
// and empty string for zero or invalid positions.
func (p Position) GetCloseDirection() string {
	val, err := strconv.ParseFloat(strings.ReplaceAll(p.Quantity, ",", "."), 64)
	if err != nil || val == 0 {
		return ""
	}
	if val > 0 {
		return "Sell"
	}
	return "Buy"
}

// Quote represents a market quote
type Quote struct {
	Symbol    string
	Bid       string
	BidSize   string
	Ask       string
	AskSize   string
	Last      string
	LastSize  string
	Volume    string
	Open      string
	High      string
	Low       string
	Close     string
	Timestamp time.Time
}

// AccountSummary contains calculated account statistics
type AccountSummary struct {
	TotalValue     float64
	TotalDailyPnL  float64
	TotalUnrealPnL float64
	PositionsCount int
}

// SecurityInfo represents basic information about a security from search results
type SecurityInfo struct {
	Ticker   string
	Symbol   string
	Name     string
	Lot      float64
	Currency string
}

// Trade represents a trade in history
type Trade struct {
	ID        string
	Symbol    string
	Name      string
	Side      string
	Price     string
	Quantity  string
	Total     string
	Timestamp time.Time
}

// Bar represents a single candlestick bar for chart rendering
type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// AssetDetails represents detailed instrument information from GetAsset API
type AssetDetails struct {
	Board          string
	ID             string
	Ticker         string
	MIC            string
	ISIN           string
	Type           string
	Name           string
	Decimals       int32
	MinStep        int64
	LotSize        string
	ExpirationDate string // formatted date string, empty if not applicable
	QuoteCurrency  string
}

// AssetParams represents trading parameters for an instrument
type AssetParams struct {
	IsTradable         bool
	Longable           string // "Available", "Not Available", "N/A"
	Shortable          string // "Available", "Not Available", "N/A"
	LongRiskRate       string
	ShortRiskRate      string
	LongInitialMargin  string // formatted as "amount currency"
	ShortInitialMargin string // formatted as "amount currency"
}

// TradingSession represents a single trading session window
type TradingSession struct {
	Type      string
	StartTime time.Time
	EndTime   time.Time
}

// InstrumentProfile aggregates all instrument data for the profile view
type InstrumentProfile struct {
	Symbol   string
	Details  *AssetDetails
	Params   *AssetParams
	Quote    *Quote
	Schedule []TradingSession
	Bars     []Bar
}

// Order represents an active order
type Order struct {
	ID           string
	Symbol       string
	Name         string
	Side         string
	Type         string
	Status       string
	Quantity     string
	Executed     string
	Price        string
	CreationTime time.Time
}
