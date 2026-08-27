package models

import (
	"strconv"
	"strings"
	"time"
)

// Order type constants for the order modal
const (
	OrderTypeMarket     = "Market"
	OrderTypeLimit      = "Limit"
	OrderTypeStop       = "Stop-Loss"
	OrderTypeTakeProfit = "Take-Profit"
	OrderTypeSLTP       = "SL + TP"
)

// OrderParams holds parameters for placing an order beyond basic market orders.
type OrderParams struct {
	OrderType  string  // OrderTypeMarket, OrderTypeLimit, OrderTypeStop, OrderTypeTakeProfit
	LimitPrice float64 // Required for Limit orders
	StopPrice  float64 // Required for Stop-Loss and Take-Profit orders
}

// AccountInfo represents account information from Finam API
type AccountInfo struct {
	ID            string
	Type          string
	Status        string
	Equity        string
	UnrealizedPnL string
	OpenDate      time.Time
	LoadError     string // Non-empty if account failed to load from broker
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
	Symbol       string
	Bid          string
	BidSize      string
	Ask          string
	AskSize      string
	Last         string
	LastSize     string
	Volume       string
	Open         string
	High         string
	Low          string
	Close        string
	Change       string
	OpenInterest string
	Timestamp    time.Time
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
	ID              string
	Symbol          string
	Name            string
	Side            string
	Price           string
	Quantity        string
	Total           string
	AccruedInterest string // formatted accrued interest (bonds only), empty for others
	Currency        string // currency of the trade price
	Timestamp       time.Time
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
	Board            string
	ID               string
	Ticker           string
	MIC              string
	ISIN             string
	Type             string
	Name             string
	Decimals         int32
	MinStep          int64
	LotSize          string
	ExpirationDate   string // formatted date string, empty if not applicable
	QuoteCurrency    string
	ContractSize     string // formatted contract size (futures, options)
	Strike           string // formatted strike price (options only)
	BondFaceValue    string // formatted face value (bonds only)
	BondFaceCurrency string // currency of face value (bonds only)
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
	// TradeLotSize is the lot size the broker uses for trading operations
	// (GetAssetParams.trade_lot_size, Trade API 2.18.1). Zero means the API has
	// no value for this instrument and callers fall back to the asset lot size.
	TradeLotSize int64
}

// TradingSession represents a single trading session window
type TradingSession struct {
	Type      string
	StartTime time.Time
	EndTime   time.Time
}

// Corporate action Kind values for BondEvent.
const (
	BondEventCoupon       = "Coupon"
	BondEventAmortization = "Amortization"
	BondEventOffer        = "Offer"
)

// Dividend represents a single dividend payment (past or future) for an equity.
// All fields are pre-formatted for display.
type Dividend struct {
	Date     string // record/close date
	Amount   string // amount per share
	Currency string
	IsFuture bool
}

// Split represents a single stock split event (past or future) for an equity.
// All fields are pre-formatted for display.
type Split struct {
	Date     string
	OldRatio string
	NewRatio string
	NewLot   string
	ConvType string
	IsFuture bool
}

// BondEvent represents a single bond corporate-action event (coupon,
// amortization, or offer). Kind selects which of the flat detail groups is
// populated. All fields are pre-formatted for display; unused detail fields
// are empty strings.
type BondEvent struct {
	Date     string
	Kind     string // BondEventCoupon, BondEventAmortization, BondEventOffer
	Value    string // primary value (coupon amount, amortization value, offer price)
	Currency string
	IsFuture bool

	// Coupon details (Kind == BondEventCoupon)
	RecordDate string
	StartDate  string
	FaceValue  string
	Percent    string // coupon rate percent, also reused for amortization percent

	// Amortization details (Kind == BondEventAmortization)
	NewFaceValue     string
	InitialFaceValue string

	// Offer details (Kind == BondEventOffer)
	Type  string // PUT / CALL
	Price string
	Start string
	End   string
	Agent string
}

// InstrumentProfile aggregates all instrument data for the profile view
type InstrumentProfile struct {
	Symbol     string
	Details    *AssetDetails
	Params     *AssetParams
	Quote      *Quote
	Schedule   []TradingSession
	Bars       []Bar
	Dividends  []Dividend
	Splits     []Split
	BondEvents []BondEvent
}

// Order represents an active order
type Order struct {
	ID               string
	Symbol           string
	Name             string
	Side             string
	Type             string
	Status           string
	Quantity         string
	Executed         string
	Price            string
	StopCondition    string
	LimitPrice       string
	StopPrice        string
	Validity         string
	ExecutedQty      string
	RemainingQty     string
	SLQty            string
	TPQty            string
	SLPrice          string
	TPPrice          string
	TriggeredOrderID string // ID of the exchange order spawned by this stop order (2.17.0)
	CreationTime     time.Time
}

// IndexConstituent is one component of a stock index, as
// AssetsService.GetConstituents returns it. Ticker is the part of Symbol before
// "@" and is what the UI shows; Symbol is the full ticker@mic the quote stream
// and the order path need. Weight is 0 when the API sends none — the value is
// only used for sorting, since its normalisation is not documented.
type IndexConstituent struct {
	Symbol string
	Ticker string
	Name   string
	Sector string
	Weight float64
}
