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
	MIC           string
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
	Name     string
	Lot      int32
	Currency string
}

// Trade represents a trade in history
type Trade struct {
	ID        string
	Symbol    string
	Side      string
	Price     string
	Quantity  string
	Total     string
	Timestamp time.Time
}

// Order represents an active order
type Order struct {
	ID           string
	Symbol       string
	Side         string
	Type         string
	Status       string
	Quantity     string
	Executed     string
	Price        string
	CreationTime time.Time
}
