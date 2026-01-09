package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"finam-terminal/models"

	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
)

// Client is a client for the Finam Trade API
type Client struct {
	conn             *grpc.ClientConn
	authClient       auth.AuthServiceClient
	accountsClient   accounts.AccountsServiceClient
	marketDataClient marketdata.MarketDataServiceClient
	assetsClient     assets.AssetsServiceClient

	token       string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex

	// Cache for instrument MIC codes
	assetMicCache map[string]string // ticker -> symbol@mic
	assetMutex    sync.RWMutex
}

// NewClient creates a new Finam API client
func NewClient(grpcAddr string, apiToken string) (*Client, error) {
	tlsConfig := tls.Config{MinVersion: tls.VersionTLS12}

	connCtx, connCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer connCancel()

	conn, err := grpc.DialContext(
		connCtx,
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
		assetMicCache:    make(map[string]string),
	}

	// Authenticate
	if err := client.authenticate(apiToken); err != nil {
		conn.Close()
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Load asset MIC cache
	if err := client.loadAssetCache(); err != nil {
		log.Printf("[WARN] Failed to load asset cache: %v", err)
	}

	return client, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// loadAssetCache loads all available instruments and their MIC codes
func (c *Client) loadAssetCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.assetsClient.Assets(ctx, &assets.AssetsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get assets: %w", err)
	}

	c.assetMutex.Lock()
	defer c.assetMutex.Unlock()

	for _, asset := range resp.Assets {
		c.assetMicCache[asset.Ticker] = asset.Symbol
	}

	log.Printf("[INFO] Loaded %d instruments into cache", len(c.assetMicCache))
	return nil
}

// getFullSymbol converts a ticker to full symbol with MIC
func (c *Client) getFullSymbol(ticker string) string {
	c.assetMutex.RLock()
	defer c.assetMutex.RUnlock()

	if strings.Contains(ticker, "@") {
		return ticker
	}

	if fullSymbol, ok := c.assetMicCache[ticker]; ok {
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

	c.tokenMutex.Lock()
	c.token = resp.Token
	c.tokenExpiry = time.Now().Add(50 * time.Minute)
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
		fullSymbol := c.getFullSymbol(ticker)

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

		positions = append(positions, position)
	}

	return account, positions, nil
}

// GetQuotes returns quotes for multiple symbols
func (c *Client) GetQuotes(symbols []string) (map[string]*models.Quote, error) {
	ctx, cancel := c.getContext()
	defer cancel()

	quotes := make(map[string]*models.Quote)
	for _, symbol := range symbols {
		fullSymbol := c.getFullSymbol(symbol)
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
