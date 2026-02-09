package api

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"finam-terminal/models"

	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/genproto/googleapis/type/interval"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	tradeapiv1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
)

// Client is a client for the Finam Trade API
type Client struct {
	conn             *grpc.ClientConn
	authClient       auth.AuthServiceClient
	accountsClient   accounts.AccountsServiceClient
	marketDataClient marketdata.MarketDataServiceClient
	assetsClient     assets.AssetsServiceClient
	ordersClient     orders.OrdersServiceClient

	token       string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex

	// New fields for auto-refresh
	apiToken      string
	lastRefresh   time.Time
	refreshCancel context.CancelFunc

	// Cache for instrument MIC codes
	assetMicCache map[string]string // ticker -> symbol@mic
	securityCache []models.SecurityInfo
	assetMutex    sync.RWMutex
}

// NewClient creates a new Finam API client
func NewClient(grpcAddr string, apiToken string) (*Client, error) {
	tlsConfig := tls.Config{MinVersion: tls.VersionTLS12}

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tlsConfig)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &Client{
		conn:             conn,
		authClient:       auth.NewAuthServiceClient(conn),
		accountsClient:   accounts.NewAccountsServiceClient(conn),
		marketDataClient: marketdata.NewMarketDataServiceClient(conn),
		assetsClient:     assets.NewAssetsServiceClient(conn),
		ordersClient:     orders.NewOrdersServiceClient(conn),
		apiToken:         apiToken,
		assetMicCache:    make(map[string]string),
		securityCache:    make([]models.SecurityInfo, 0),
	}

	// Authenticate
	if err := client.authenticate(apiToken); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Start background token refresh
	refreshCtx, cancel := context.WithCancel(context.Background())
	client.refreshCancel = cancel
	go client.startTokenRefresh(refreshCtx)

	// Load asset MIC cache
	if err := client.loadAssetCache(); err != nil {
		log.Printf("[WARN] Failed to load asset cache: %v", err)
	}

	return client, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.refreshCancel != nil {
		c.refreshCancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// startTokenRefresh runs in a goroutine and proactively refreshes the token
func (c *Client) startTokenRefresh(ctx context.Context) {
	log.Printf("[INFO] Background token refresh process started")
	for {
		duration := c.getRefreshDuration()
		log.Printf("[DEBUG] Next token refresh in %v", duration)

		select {
		case <-ctx.Done():
			log.Printf("[INFO] Background token refresh process stopped")
			return
		case <-time.After(duration):
			if err := c.authenticate(c.apiToken); err != nil {
				log.Printf("[ERROR] Token refresh failed: %v. Retrying in 30s...", err)
				// Retry after a short delay on failure
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					continue
				}
			}
			c.tokenMutex.Lock()
			c.lastRefresh = time.Now()
			c.tokenMutex.Unlock()
			log.Printf("[INFO] Token refreshed successfully")
		}
	}
}

// getRefreshDuration calculates how long to wait before the next refresh
func (c *Client) getRefreshDuration() time.Duration {
	c.tokenMutex.RLock()
	expiry := c.tokenExpiry
	c.tokenMutex.RUnlock()

	// Refresh 2 minutes before actual expiry
	refreshAt := expiry.Add(-2 * time.Minute)
	duration := time.Until(refreshAt)

	// If already past refresh time or expiry is too soon, refresh in 1 second
	if duration <= 0 {
		return 1 * time.Second
	}

	// Fallback/Safety: If expiry is very far or missing, default to 10 minutes
	if duration > 10*time.Hour || expiry.IsZero() {
		return 10 * time.Minute
	}

	return duration
}

// getExpiryFromToken extracts the expiration time from a JWT token
func (c *Client) getExpiryFromToken(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}

	payload := parts[1]
	// Add padding if needed (JWTs are raw url encoded, but robustness helps)
	if l := len(payload) % 4; l > 0 {
		payload += strings.Repeat("=", 4-l)
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[1]) // Try RawURLEncoding first (standard)
	if err != nil {
		// Fallback to standard URL encoding if raw fails
		data, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}

	if err := json.Unmarshal(data, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("exp claim missing")
	}

	return time.Unix(claims.Exp, 0), nil
}

// loadAssetCache loads all available instruments and their MIC codes
func (c *Client) loadAssetCache() error {
	ctx, cancel := c.getContext()
	defer cancel()

	// Use empty request to get all assets (subject to API limits)
	resp, err := c.assetsClient.Assets(ctx, &assets.AssetsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get assets: %w", err)
	}

	c.assetMutex.Lock()
	defer c.assetMutex.Unlock()

	c.securityCache = make([]models.SecurityInfo, 0, len(resp.Assets))
	for _, asset := range resp.Assets {
		c.assetMicCache[asset.Ticker] = asset.Symbol
		c.securityCache = append(c.securityCache, models.SecurityInfo{
			Ticker: asset.Ticker,
			Name:   asset.Name,
		})
	}

	log.Printf("[INFO] Loaded %d instruments into cache", len(c.assetMicCache))
	return nil
}

// getFullSymbol converts a ticker to full symbol with MIC
func (c *Client) getFullSymbol(ticker string, accountID string) string {
	// First check local cache
	c.assetMutex.RLock()
	if strings.Contains(ticker, "@") {
		c.assetMutex.RUnlock()
		return ticker
	}
	if fullSymbol, ok := c.assetMicCache[ticker]; ok {
		c.assetMutex.RUnlock()
		return fullSymbol
	}
	c.assetMutex.RUnlock()

	// Fallback: Fetch specific asset from API
	log.Printf("[DEBUG] Cache miss for ticker: %s. Fetching from API...", ticker)

	ctx, cancel := c.getContext()
	defer cancel()

	// Pass AccountId to GetAssetRequest
	resp, err := c.assetsClient.GetAsset(ctx, &assets.GetAssetRequest{
		Symbol:    ticker,
		AccountId: accountID,
	})
	if err != nil {
		log.Printf("[WARN] Failed to fetch asset %s: %v", ticker, err)
		return ticker // Return original if failed
	}

	if resp.Ticker != "" && resp.Board != "" {
		fullSymbol := fmt.Sprintf("%s@%s", resp.Ticker, resp.Board)

		c.assetMutex.Lock()
		c.assetMicCache[ticker] = fullSymbol
		c.assetMutex.Unlock()

		log.Printf("[DEBUG] Resolved %s via API: %s", ticker, fullSymbol)
		return fullSymbol
	}

	return ticker
}

// authenticate performs authentication and stores the JWT token
func (c *Client) authenticate(apiToken string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.authClient.Auth(ctx, &auth.AuthRequest{Secret: apiToken})
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}

	// The original code had c.tokenExpiry = time.Now().Add(50 * time.Minute)
	// This has been updated to parse expiry from the token.
	expiry, err := c.getExpiryFromToken(resp.Token)
	if err != nil {
		log.Printf("[WARN] Could not parse expiry from token: %v. Using default 50m.", err)
		expiry = time.Now().Add(50 * time.Minute)
	}

	c.tokenMutex.Lock()
	c.token = resp.Token
	c.tokenExpiry = expiry
	c.tokenMutex.Unlock()

	log.Println("[INFO] Authentication successful")
	return nil
}

// getContext returns a context with authentication metadata
func (c *Client) getContext() (context.Context, context.CancelFunc) {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", c.token)
	return ctx, cancel
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(accountID string, symbol string, buySell string, quantity float64) (string, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	fullSymbol := c.getFullSymbol(symbol, accountID)
	log.Printf("[DEBUG] PlaceOrder: input='%s', resolved='%s'", symbol, fullSymbol)

	var side tradeapiv1.Side
	switch strings.ToLower(buySell) {
	case "buy":
		side = tradeapiv1.Side_SIDE_BUY
	case "sell":
		side = tradeapiv1.Side_SIDE_SELL
	default:
		return "", fmt.Errorf("invalid direction: %s", buySell)
	}

	qtyDecimal := &decimal.Decimal{Value: fmt.Sprintf("%v", quantity)}

	req := &orders.Order{
		AccountId: accountID,
		Symbol:    fullSymbol,
		Quantity:  qtyDecimal,
		Side:      side,
		Type:      orders.OrderType_ORDER_TYPE_MARKET,
	}

	resp, err := c.ordersClient.PlaceOrder(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to place order: %w", err)
	}

	return resp.OrderId, nil
}

// ClosePosition closes (fully or partially) an existing position
func (c *Client) ClosePosition(accountID string, symbol string, currentQuantity string, closeQuantity float64) (string, error) {
	// Determine direction
	pos := models.Position{Quantity: currentQuantity}
	dir := pos.GetCloseDirection()
	if dir == "" {
		return "", fmt.Errorf("could not determine close direction for quantity %s", currentQuantity)
	}

	return c.PlaceOrder(accountID, symbol, dir, closeQuantity)
}

// GetAccounts returns a list of all accounts
func (c *Client) GetAccounts() ([]models.AccountInfo, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	resp, err := c.authClient.TokenDetails(ctx, &auth.TokenDetailsRequest{Token: c.token})
	if err != nil {
		return nil, fmt.Errorf("failed to get token details: %w", err)
	}

	var accountsList []models.AccountInfo
	for _, accountID := range resp.AccountIds {
		accountResp, err := c.accountsClient.GetAccount(ctx, &accounts.GetAccountRequest{
			AccountId: accountID,
		})
		if err != nil {
			log.Printf("[WARN] Failed to get account %s: %v", accountID, err)
			continue
		}

		account := models.AccountInfo{
			ID:       accountID,
			Type:     accountResp.Type,
			Status:   accountResp.Status,
			OpenDate: accountResp.OpenAccountDate.AsTime(),
		}

		if equity := accountResp.Equity; equity != nil {
			account.Equity = formatDecimal(equity)
		}
		if unrealized := accountResp.UnrealizedProfit; unrealized != nil {
			account.UnrealizedPnL = formatDecimal(unrealized)
		}

		accountsList = append(accountsList, account)
	}

	return accountsList, nil
}

// GetAccountDetails returns detailed information for a specific account
func (c *Client) GetAccountDetails(accountID string) (*models.AccountInfo, []models.Position, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	accountResp, err := c.accountsClient.GetAccount(ctx, &accounts.GetAccountRequest{
		AccountId: accountID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get account: %w", err)
	}

	account := &models.AccountInfo{
		ID:       accountID,
		Type:     accountResp.Type,
		Status:   accountResp.Status,
		OpenDate: accountResp.OpenAccountDate.AsTime(),
	}

	if equity := accountResp.Equity; equity != nil {
		account.Equity = formatDecimal(equity)
	}
	if unrealized := accountResp.UnrealizedProfit; unrealized != nil {
		account.UnrealizedPnL = formatDecimal(unrealized)
	}

	var positions []models.Position
	for _, pos := range accountResp.Positions {
		ticker := pos.Symbol
		fullSymbol := c.getFullSymbol(ticker, accountID)

		mic := ""
		if strings.Contains(fullSymbol, "@") {
			parts := strings.SplitN(fullSymbol, "@", 2)
			ticker = parts[0]
			mic = parts[1]
		}

		position := models.Position{
			Symbol:        fullSymbol,
			Ticker:        ticker,
			MIC:           mic,
			Quantity:      formatDecimal(pos.Quantity),
			AveragePrice:  formatDecimal(pos.AveragePrice),
			CurrentPrice:  formatDecimal(pos.CurrentPrice),
			DailyPnL:      formatDecimal(pos.DailyPnl),
			UnrealizedPnL: formatDecimal(pos.UnrealizedPnl),
		}

		// Filter out zero positions (historical or closed)
		if qtyVal, err := strconv.ParseFloat(position.Quantity, 64); err == nil && qtyVal == 0 {
			continue
		}

		positions = append(positions, position)
	}

	return account, positions, nil
}

// GetQuotes returns quotes for multiple symbols
func (c *Client) GetQuotes(accountID string, symbols []string) (map[string]*models.Quote, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	quotes := make(map[string]*models.Quote)
	for _, symbol := range symbols {
		fullSymbol := c.getFullSymbol(symbol, accountID)
		if !strings.Contains(fullSymbol, "@") {
			continue
		}

		resp, err := c.marketDataClient.LastQuote(ctx, &marketdata.QuoteRequest{
			Symbol: fullSymbol,
		})
		if err != nil {
			log.Printf("[WARN] Failed to get quote for %s: %v", fullSymbol, err)
			continue
		}

		q := resp.Quote
		if q == nil {
			continue
		}

		quotes[fullSymbol] = &models.Quote{
			Symbol:    fullSymbol,
			Bid:       formatDecimal(q.Bid),
			BidSize:   formatDecimal(q.BidSize),
			Ask:       formatDecimal(q.Ask),
			AskSize:   formatDecimal(q.AskSize),
			Last:      formatDecimal(q.Last),
			LastSize:  formatDecimal(q.LastSize),
			Volume:    formatDecimal(q.Volume),
			Open:      formatDecimal(q.Open),
			High:      formatDecimal(q.High),
			Low:       formatDecimal(q.Low),
			Close:     formatDecimal(q.Close),
			Timestamp: q.Timestamp.AsTime(),
		}
	}

	return quotes, nil
}

// SearchSecurities searches for securities by ticker or name
func (c *Client) SearchSecurities(query string) ([]models.SecurityInfo, error) {
	c.assetMutex.RLock()
	defer c.assetMutex.RUnlock()

	if len(c.securityCache) == 0 {
		return nil, nil
	}

	query = strings.ToLower(query)
	var results []models.SecurityInfo

	for _, sec := range c.securityCache {
		if strings.Contains(strings.ToLower(sec.Ticker), query) || strings.Contains(strings.ToLower(sec.Name), query) {
			results = append(results, sec)
			if len(results) >= 50 { // Limit results
				break
			}
		}
	}

	return results, nil
}

// GetTradeHistory returns trade history for an account
func (c *Client) GetTradeHistory(accountID string) ([]models.Trade, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	now := time.Now()
	startTime := now.AddDate(0, 0, -30) // Last 30 days

	resp, err := c.accountsClient.Trades(ctx, &accounts.TradesRequest{
		AccountId: accountID,
		Interval: &interval.Interval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(now),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	var trades []models.Trade
	for _, t := range resp.Trades {
		side := "Unknown"
		if t.Side == tradeapiv1.Side_SIDE_BUY {
			side = "Buy"
		} else if t.Side == tradeapiv1.Side_SIDE_SELL {
			side = "Sell"
		}

		priceStr := formatDecimal(t.Price)
		qtyStr := formatDecimal(t.Size)

		price, _ := strconv.ParseFloat(priceStr, 64)
		qty, _ := strconv.ParseFloat(qtyStr, 64)
		total := price * qty

		trades = append(trades, models.Trade{
			ID:        t.TradeId,
			Symbol:    t.Symbol,
			Side:      side,
			Price:     priceStr,
			Quantity:  qtyStr,
			Total:     fmt.Sprintf("%.2f", total),
			Timestamp: t.Timestamp.AsTime(),
		})
	}
	return trades, nil
}

// GetActiveOrders returns active orders for an account
func (c *Client) GetActiveOrders(accountID string) ([]models.Order, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	resp, err := c.ordersClient.GetOrders(ctx, &orders.OrdersRequest{
		AccountId: accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	var activeOrders []models.Order
	for _, o := range resp.Orders {
		side := "Unknown"
		if o.Order != nil {
			if o.Order.Side == tradeapiv1.Side_SIDE_BUY {
				side = "Buy"
			} else if o.Order.Side == tradeapiv1.Side_SIDE_SELL {
				side = "Sell"
			}
		}

		status := "Unknown"
		switch o.Status {
		case orders.OrderStatus_ORDER_STATUS_UNSPECIFIED:
			status = "Unspecified"
		case orders.OrderStatus_ORDER_STATUS_NEW:
			status = "New"
		case orders.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED:
			status = "Partial"
		case orders.OrderStatus_ORDER_STATUS_FILLED:
			status = "Filled"
		case orders.OrderStatus_ORDER_STATUS_CANCELED:
			status = "Cancelled"
		case orders.OrderStatus_ORDER_STATUS_REJECTED:
			status = "Rejected"
		case orders.OrderStatus_ORDER_STATUS_EXECUTED:
			status = "Executed"
		}

		order := models.Order{
			ID:     o.OrderId,
			Status: status,
			Side:   side,
		}

		if o.Order != nil {
			order.Symbol = o.Order.Symbol
			order.Type = o.Order.Type.String()
			order.Quantity = formatDecimal(o.Order.Quantity)
			order.Price = formatDecimal(o.Order.LimitPrice)
			if order.Price == "0" || order.Price == "" {
				order.Price = "Market"
			}
		}

		if o.TransactAt != nil {
			order.CreationTime = o.TransactAt.AsTime()
		}

		activeOrders = append(activeOrders, order)
	}
	return activeOrders, nil
}

// GetSnapshots returns initial prices for a list of securities
func (c *Client) GetSnapshots(symbols []string) (map[string]models.Quote, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	// We can reuse GetQuotes logic but return values directly
	// Or just use GetQuotes as it already fetches LastQuote
	// However, GetQuotes takes accountID and resolves symbols.
	// For search results, we usually have Tickers, need to resolve to full symbol.
	// Search results already come from cache which has full symbol?
	// No, SecurityInfo has Ticker.

	// Helper to get quotes without account ID context if possible?
	// MarketDataServiceClient.LastQuote needs Symbol.

	// Let's rely on the cache to resolve tickers to full symbols.
	// But GetQuotes uses getFullSymbol which uses accountID to fetch if missing.
	// Here we might not have accountID context easily in the UI search component?
	// The search component will likely pass tickers.
	// If we don't have accountID, we can pass empty string to getFullSymbol?
	// getFullSymbol needs accountID only for "Fallback: Fetch specific asset from API".
	// If it's in cache (which it should be for search results), it returns immediately.

	// So we can use GetQuotes with empty accountID?
	// GetQuotes returns *models.Quote.

	// Let's implement a simplified version or wrapper.
	ctx, cancel := c.getContext()
	defer cancel()

	quotes := make(map[string]models.Quote)
	for _, ticker := range symbols {
		fullSymbol := c.getFullSymbol(ticker, "") // Empty accountID
		if !strings.Contains(fullSymbol, "@") {
			continue
		}

		resp, err := c.marketDataClient.LastQuote(ctx, &marketdata.QuoteRequest{
			Symbol: fullSymbol,
		})
		if err != nil {
			log.Printf("[WARN] Failed to get snapshot for %s: %v", fullSymbol, err)
			continue
		}

		q := resp.Quote
		if q == nil {
			continue
		}

		quotes[ticker] = models.Quote{
			Symbol:    fullSymbol,
			Last:      formatDecimal(q.Last),
			LastSize:  formatDecimal(q.LastSize),
			Volume:    formatDecimal(q.Volume),
			Close:     formatDecimal(q.Close),
			Timestamp: q.Timestamp.AsTime(),
		}
	}

	return quotes, nil
}

// formatDecimal formats a google decimal value
func formatDecimal(d *decimal.Decimal) string {
	if d == nil {
		return "N/A"
	}
	s := fmt.Sprintf("%v", d)
	s = strings.TrimPrefix(s, "value:")
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}
