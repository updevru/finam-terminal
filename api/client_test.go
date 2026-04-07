package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"finam-terminal/models"

	tradeapiv1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/genproto/googleapis/type/interval"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockMarketDataServiceClient is a manual mock for marketdata.MarketDataServiceClient
type mockMarketDataServiceClient struct {
	marketdata.MarketDataServiceClient
	LastQuoteFunc func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error)
	BarsFunc      func(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error)
}

func (m *mockMarketDataServiceClient) LastQuote(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
	return m.LastQuoteFunc(ctx, in, opts...)
}

func (m *mockMarketDataServiceClient) Bars(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error) {
	return m.BarsFunc(ctx, in, opts...)
}

// mockAssetsServiceClient is a manual mock for assets.AssetsServiceClient
type mockAssetsServiceClient struct {
	assets.AssetsServiceClient
	AssetsFunc         func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error)
	GetAssetFunc       func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error)
	GetAssetParamsFunc func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error)
	ScheduleFunc       func(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error)
}

func (m *mockAssetsServiceClient) Assets(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
	return m.AssetsFunc(ctx, in, opts...)
}

func (m *mockAssetsServiceClient) GetAsset(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
	return m.GetAssetFunc(ctx, in, opts...)
}

func (m *mockAssetsServiceClient) GetAssetParams(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
	return m.GetAssetParamsFunc(ctx, in, opts...)
}

func (m *mockAssetsServiceClient) Schedule(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error) {
	return m.ScheduleFunc(ctx, in, opts...)
}

// mockAccountsServiceClient is a manual mock for accounts.AccountsServiceClient
type mockAccountsServiceClient struct {
	accounts.AccountsServiceClient
	GetAccountFunc func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error)
	TradesFunc     func(ctx context.Context, in *accounts.TradesRequest, opts ...grpc.CallOption) (*accounts.TradesResponse, error)
}

func (m *mockAccountsServiceClient) GetAccount(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
	return m.GetAccountFunc(ctx, in, opts...)
}

func (m *mockAccountsServiceClient) Trades(ctx context.Context, in *accounts.TradesRequest, opts ...grpc.CallOption) (*accounts.TradesResponse, error) {
	return m.TradesFunc(ctx, in, opts...)
}

// mockAuthServiceClient is a manual mock for auth.AuthServiceClient
type mockAuthServiceClient struct {
	auth.AuthServiceClient
	TokenDetailsFunc func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error)
	AuthFunc         func(ctx context.Context, in *auth.AuthRequest, opts ...grpc.CallOption) (*auth.AuthResponse, error)
}

func (m *mockAuthServiceClient) TokenDetails(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
	return m.TokenDetailsFunc(ctx, in, opts...)
}

func (m *mockAuthServiceClient) Auth(ctx context.Context, in *auth.AuthRequest, opts ...grpc.CallOption) (*auth.AuthResponse, error) {
	return m.AuthFunc(ctx, in, opts...)
}

// mockOrdersServiceClient is a manual mock for orders.OrdersServiceClient
type mockOrdersServiceClient struct {
	orders.OrdersServiceClient
	PlaceOrderFunc     func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error)
	PlaceSLTPOrderFunc func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error)
	GetOrdersFunc      func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error)
	CancelOrderFunc    func(ctx context.Context, in *orders.CancelOrderRequest, opts ...grpc.CallOption) (*orders.OrderState, error)
}

func (m *mockOrdersServiceClient) PlaceOrder(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
	return m.PlaceOrderFunc(ctx, in, opts...)
}

func (m *mockOrdersServiceClient) GetOrders(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
	return m.GetOrdersFunc(ctx, in, opts...)
}

func (m *mockOrdersServiceClient) PlaceSLTPOrder(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
	return m.PlaceSLTPOrderFunc(ctx, in, opts...)
}

func (m *mockOrdersServiceClient) CancelOrder(ctx context.Context, in *orders.CancelOrderRequest, opts ...grpc.CallOption) (*orders.OrderState, error) {
	return m.CancelOrderFunc(ctx, in, opts...)
}

func TestPlaceOrder_Success(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.AccountId != "test-acc" {
				return nil, context.Canceled // Just a dummy error
			}
			if in.Symbol != "SBER@TQBR" {
				return nil, context.Canceled
			}
			if in.Side != tradeapiv1.Side_SIDE_BUY {
				return nil, context.Canceled
			}
			if in.Quantity == nil || in.Quantity.Value != "10" {
				return nil, context.Canceled
			}

			return &orders.OrderState{
				OrderId: "12345",
			}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
		},
		assetLotCache: map[string]float64{
			"SBER": 1,
		},
	}

	txID, err := client.PlaceOrder("test-acc", "SBER", "Buy", 10, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if txID != "12345" {
		t.Errorf("Expected TransactionID 12345, got %s", txID)
	}
}

func TestPlaceOrder_Error(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			return nil, context.Canceled
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
		},
		assetLotCache: map[string]float64{
			"SBER": 1,
		},
	}

	_, err := client.PlaceOrder("test-acc", "SBER", "Buy", 10, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestClosePosition_Success(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			// If we are closing a Long position (Quantity "10"), it should be a SELL
			if in.Side != tradeapiv1.Side_SIDE_SELL {
				return nil, context.Canceled
			}
			if in.Quantity.Value != "5" {
				return nil, context.Canceled
			}
			return &orders.OrderState{OrderId: "999"}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
		},
		assetLotCache: map[string]float64{
			"SBER": 1,
		},
	}

	// Long position
	id, err := client.ClosePosition("test-acc", "SBER", "10", 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "999" {
		t.Errorf("Expected OrderId 999, got %s", id)
	}
}

func TestGetAccountDetails(t *testing.T) {
	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId: in.AccountId,
				Type:      "test-type",
				Status:    "test-status",
				Equity: &decimal.Decimal{
					Value: "100",
				},
				UnrealizedProfit: &decimal.Decimal{
					Value: "10.5",
				},
				Positions: []*accounts.Position{
					{
						Symbol:        "GAZP",
						Quantity:      &decimal.Decimal{Value: "10"},
						AveragePrice:  &decimal.Decimal{Value: "150"},
						CurrentPrice:  &decimal.Decimal{Value: "160"},
						DailyPnl:      &decimal.Decimal{Value: "100"},
						UnrealizedPnl: &decimal.Decimal{Value: "100"},
					},
				},
			}, nil
		},
	}

	client := &Client{
		accountsClient: mockAccounts,
		assetMicCache: map[string]string{
			"GAZP": "GAZP@TQBR",
		},
		assetLotCache: map[string]float64{
			"GAZP": 1,
		},
		instrumentNameCache: map[string]string{
			"GAZP":      "Газпром",
			"GAZP@TQBR": "Газпром",
		},
	}

	account, positions, err := client.GetAccountDetails("test-acc")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.ID != "test-acc" {
		t.Errorf("Expected ID test-acc, got %s", account.ID)
	}
	if account.Equity != "100" {
		t.Errorf("Expected Equity 100, got %s", account.Equity)
	}
	if account.UnrealizedPnL != "10.5" {
		t.Errorf("Expected UnrealizedPnL 10.5, got %s", account.UnrealizedPnL)
	}
	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != "GAZP@TQBR" {
		t.Errorf("Expected symbol GAZP@TQBR, got %s", positions[0].Symbol)
	}
	if positions[0].Ticker != "GAZP" || positions[0].MIC != "TQBR" {
		t.Errorf("Ticker/MIC mismatch")
	}
	if positions[0].Name != "Газпром" {
		t.Errorf("Expected Name Газпром, got '%s'", positions[0].Name)
	}
}

func TestGetAccounts(t *testing.T) {
	mockAuth := &mockAuthServiceClient{
		TokenDetailsFunc: func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
			return &auth.TokenDetailsResponse{
				AccountIds: []string{"acc1", "acc2"},
			}, nil
		},
	}

	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId: in.AccountId,
				Type:      "test-type",
				Status:    "test-status",
			}, nil
		},
	}

	client := &Client{
		authClient:     mockAuth,
		accountsClient: mockAccounts,
	}

	accs, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(accs) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accs))
	}
	if accs[0].ID != "acc1" || accs[1].ID != "acc2" {
		t.Errorf("Account IDs mismatch")
	}
}

func TestSearchSecurities(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		AssetsFunc: func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
			return &assets.AssetsResponse{
				Assets: []*assets.Asset{
					{Ticker: "AAPL", Name: "Apple Inc."},
					{Ticker: "MSFT", Name: "Microsoft Corp."},
					{Ticker: "SBER", Name: "Sberbank"},
				},
			}, nil
		},
	}
	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		instrumentNameCache: make(map[string]string),
	}

	// Load cache manually for test
	if err := client.loadAssetCache(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Test Search "App" (should find Apple)

	results, err := client.SearchSecurities("App")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'App', got %d", len(results))
	}
	if results[0].Ticker != "AAPL" {
		t.Errorf("Expected AAPL, got %s", results[0].Ticker)
	}

	// Test Search "sber" (case insensitive)
	results, err = client.SearchSecurities("sber")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'sber', got %d", len(results))
	}
	if results[0].Ticker != "SBER" {
		t.Errorf("Expected SBER, got %s", results[0].Ticker)
	}
}

func TestGetSnapshots(t *testing.T) {
	mockMarketData := &mockMarketDataServiceClient{
		LastQuoteFunc: func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
			if in.Symbol == "AAPL" { // AAPL is not resolved in cache for this test unless we add it
				return nil, context.Canceled
			}
			if in.Symbol == "SBER@TQBR" {
				return &marketdata.QuoteResponse{
					Quote: &marketdata.Quote{
						Symbol:    "SBER@TQBR",
						Last:      &decimal.Decimal{Value: "250.50"},
						LastSize:  &decimal.Decimal{Value: "10"},
						Timestamp: timestamppb.Now(),
					},
				}, nil
			}
			return &marketdata.QuoteResponse{}, nil
		},
	}

	client := &Client{
		marketDataClient: mockMarketData,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
		},
		assetLotCache: map[string]float64{
			"SBER": 1,
		},
	}

	// Test GetSnapshots
	quotes, err := client.GetSnapshots("acc1", []string{"SBER"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(quotes) != 1 {
		t.Errorf("Expected 1 quote, got %d", len(quotes))
	}

	q, ok := quotes["SBER"]
	if !ok {
		t.Errorf("Expected quote for SBER")
	}

	if q.Last != "250.50" {
		t.Errorf("Expected Last 250.50, got %s", q.Last)
	}
}

func TestGetTradeHistory(t *testing.T) {
	mockAccounts := &mockAccountsServiceClient{
		TradesFunc: func(ctx context.Context, in *accounts.TradesRequest, opts ...grpc.CallOption) (*accounts.TradesResponse, error) {
			if in.Interval == nil || in.Interval.StartTime == nil || in.Interval.EndTime == nil {
				return nil, fmt.Errorf("interval fields are required")
			}
			return &accounts.TradesResponse{
				Trades: []*tradeapiv1.AccountTrade{
					{
						TradeId:   "T1",
						Symbol:    "SBER",
						Price:     &decimal.Decimal{Value: "250.00"},
						Size:      &decimal.Decimal{Value: "10"},
						Side:      tradeapiv1.Side_SIDE_BUY,
						Timestamp: timestamppb.Now(),
					},
				},
			}, nil
		},
	}

	client := &Client{
		accountsClient: mockAccounts,
		instrumentNameCache: map[string]string{
			"SBER": "Сбербанк",
		},
	}

	trades, err := client.GetTradeHistory("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].ID != "T1" {
		t.Errorf("Expected trade ID T1, got %s", trades[0].ID)
	}
	if trades[0].Side != "Buy" {
		t.Errorf("Expected Side Buy, got %s", trades[0].Side)
	}
	if trades[0].Total != "2500.00" {
		t.Errorf("Expected Total 2500.00, got %s", trades[0].Total)
	}
	if trades[0].Name != "Сбербанк" {
		t.Errorf("Expected Name Сбербанк, got '%s'", trades[0].Name)
	}
}

func TestGetTradeHistory_LocalTimezone(t *testing.T) {
	// Create a known UTC timestamp
	utcTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockAccounts := &mockAccountsServiceClient{
		TradesFunc: func(ctx context.Context, in *accounts.TradesRequest, opts ...grpc.CallOption) (*accounts.TradesResponse, error) {
			return &accounts.TradesResponse{
				Trades: []*tradeapiv1.AccountTrade{
					{
						TradeId:   "T-TZ",
						Symbol:    "SBER",
						Price:     &decimal.Decimal{Value: "100"},
						Size:      &decimal.Decimal{Value: "1"},
						Side:      tradeapiv1.Side_SIDE_BUY,
						Timestamp: ts,
					},
				},
			}, nil
		},
	}

	client := &Client{
		accountsClient:      mockAccounts,
		instrumentNameCache: map[string]string{},
	}

	trades, err := client.GetTradeHistory("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	// The timestamp should be in local timezone, not UTC
	expectedLocal := utcTime.Local()
	if !trades[0].Timestamp.Equal(expectedLocal) {
		t.Errorf("Timestamps should represent the same instant")
	}
	if trades[0].Timestamp.Location() != time.Local {
		t.Errorf("Expected trade timestamp in local timezone (%s), got %s",
			time.Local, trades[0].Timestamp.Location())
	}
}

func TestGetActiveOrders(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		GetOrdersFunc: func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
			return &orders.OrdersResponse{
				Orders: []*orders.OrderState{
					{
						OrderId: "O1",
						Status:  orders.OrderStatus_ORDER_STATUS_NEW,
						Order: &orders.Order{
							Symbol:     "GAZP",
							Side:       tradeapiv1.Side_SIDE_SELL,
							Quantity:   &decimal.Decimal{Value: "100"},
							LimitPrice: &decimal.Decimal{Value: "150.00"},
						},
						TransactAt: timestamppb.Now(),
					},
				},
			}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		instrumentNameCache: map[string]string{
			"GAZP": "Газпром",
		},
	}

	activeOrders, err := client.GetActiveOrders("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(activeOrders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(activeOrders))
	}
	if activeOrders[0].ID != "O1" {
		t.Errorf("Expected order ID O1, got %s", activeOrders[0].ID)
	}
	if activeOrders[0].Status != "Active" {
		t.Errorf("Expected Status Active, got %s", activeOrders[0].Status)
	}
	if activeOrders[0].Side != "Sell" {
		t.Errorf("Expected Side Sell, got %s", activeOrders[0].Side)
	}
	if activeOrders[0].Name != "Газпром" {
		t.Errorf("Expected Name Газпром, got '%s'", activeOrders[0].Name)
	}
}

func TestGetAccounts_LocalTimezone(t *testing.T) {
	utcTime := time.Date(2020, 1, 15, 10, 0, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockAuth := &mockAuthServiceClient{
		TokenDetailsFunc: func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
			return &auth.TokenDetailsResponse{AccountIds: []string{"acc1"}}, nil
		},
	}

	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId:       in.AccountId,
				OpenAccountDate: ts,
			}, nil
		},
	}

	client := &Client{
		authClient:     mockAuth,
		accountsClient: mockAccounts,
	}

	accs, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(accs) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accs))
	}

	if accs[0].OpenDate.Location() != time.Local {
		t.Errorf("Expected OpenDate in local timezone (%s), got %s",
			time.Local, accs[0].OpenDate.Location())
	}
}

func TestGetAccountDetails_LocalTimezone(t *testing.T) {
	utcTime := time.Date(2020, 1, 15, 10, 0, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId:       in.AccountId,
				OpenAccountDate: ts,
			}, nil
		},
	}

	client := &Client{
		accountsClient:      mockAccounts,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	account, _, err := client.GetAccountDetails("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.OpenDate.Location() != time.Local {
		t.Errorf("Expected OpenDate in local timezone (%s), got %s",
			time.Local, account.OpenDate.Location())
	}
}

func TestGetActiveOrders_LocalTimezone(t *testing.T) {
	// Create a known UTC timestamp
	utcTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockOrders := &mockOrdersServiceClient{
		GetOrdersFunc: func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
			return &orders.OrdersResponse{
				Orders: []*orders.OrderState{
					{
						OrderId: "O-TZ",
						Status:  orders.OrderStatus_ORDER_STATUS_NEW,
						Order: &orders.Order{
							Symbol:   "SBER",
							Side:     tradeapiv1.Side_SIDE_BUY,
							Quantity: &decimal.Decimal{Value: "10"},
						},
						TransactAt: ts,
					},
				},
			}, nil
		},
	}

	client := &Client{
		ordersClient:        mockOrders,
		instrumentNameCache: map[string]string{},
	}

	activeOrders, err := client.GetActiveOrders("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(activeOrders) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(activeOrders))
	}

	expectedLocal := utcTime.Local()
	if !activeOrders[0].CreationTime.Equal(expectedLocal) {
		t.Errorf("Timestamps should represent the same instant")
	}
	if activeOrders[0].CreationTime.Location() != time.Local {
		t.Errorf("Expected order timestamp in local timezone (%s), got %s",
			time.Local, activeOrders[0].CreationTime.Location())
	}
}

func TestLotSizeRetrieval(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		AssetsFunc: func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
			return &assets.AssetsResponse{
				Assets: []*assets.Asset{
					{Ticker: "SBER", Name: "Sberbank", Symbol: "SBER@TQBR"},
				},
			}, nil
		},
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			if in.Symbol == "SBER" || in.Symbol == "SBER@TQBR" {
				return &assets.GetAssetResponse{
					Ticker:  "SBER",
					Board:   "TQBR",
					LotSize: &decimal.Decimal{Value: "10"},
				}, nil
			}
			return &assets.GetAssetResponse{}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	if err := client.loadAssetCache(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Trigger getFullSymbol to fetch and cache lot size
	client.getFullSymbol("SBER", "acc1")

	client.assetMutex.RLock()
	lot := client.assetLotCache["SBER"]
	client.assetMutex.RUnlock()

	if lot != 10 {
		t.Errorf("Expected cached LotSize 10, got %f", lot)
	}
}

func TestGetAccountDetails_LotSize(t *testing.T) {
	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId: in.AccountId,
				Positions: []*accounts.Position{
					{
						Symbol:   "SBER",
						Quantity: &decimal.Decimal{Value: "100"},
					},
				},
			}, nil
		},
	}

	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return &assets.GetAssetResponse{
				Ticker:  "SBER",
				Board:   "TQBR",
				LotSize: &decimal.Decimal{Value: "10"},
			}, nil
		},
	}

	client := &Client{
		accountsClient:      mockAccounts,
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	_, positions, err := client.GetAccountDetails("test-acc")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(positions) != 1 {
		t.Fatalf("Expected 1 position, got %d", len(positions))
	}

	if positions[0].LotSize != 10 {
		t.Errorf("Expected LotSize 10, got %f", positions[0].LotSize)
	}
}

func TestPlaceOrder_LotMultiplication(t *testing.T) {
	// When placing an order with quantity 1 (lot) and lot size 10,
	// the API should receive quantity = 10 (shares)
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			// Verify the API receives 10 shares (1 lot * 10 lot size)
			if in.Quantity == nil || in.Quantity.Value != "10" {
				t.Errorf("Expected API to receive quantity 10 (shares), got %s", in.Quantity.GetValue())
				return nil, fmt.Errorf("unexpected quantity: %s", in.Quantity.GetValue())
			}
			return &orders.OrderState{OrderId: "LOT-001"}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
		},
		assetLotCache: map[string]float64{
			"SBER":      10,
			"SBER@TQBR": 10,
		},
	}

	// Place order for 1 lot
	txID, err := client.PlaceOrder("test-acc", "SBER", "Buy", 1, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if txID != "LOT-001" {
		t.Errorf("Expected OrderId LOT-001, got %s", txID)
	}
}

func TestInstrumentNameCache(t *testing.T) {
	client := &Client{
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	// Initially empty
	if name := client.GetInstrumentName("SBER"); name != "" {
		t.Errorf("Expected empty name for unknown key, got %s", name)
	}

	// Update cache with ticker, full symbol, and name
	client.UpdateInstrumentCache("SBER", "SBER@TQBR", "Сбербанк")

	// Lookup by ticker
	if name := client.GetInstrumentName("SBER"); name != "Сбербанк" {
		t.Errorf("Expected Сбербанк for ticker, got %s", name)
	}

	// Lookup by full symbol
	if name := client.GetInstrumentName("SBER@TQBR"); name != "Сбербанк" {
		t.Errorf("Expected Сбербанк for full symbol, got %s", name)
	}

	// Unknown key still returns empty
	if name := client.GetInstrumentName("GAZP"); name != "" {
		t.Errorf("Expected empty name for unknown key, got %s", name)
	}
}

func TestLoadAssetCache_PopulatesInstrumentNames(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		AssetsFunc: func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
			return &assets.AssetsResponse{
				Assets: []*assets.Asset{
					{Ticker: "SBER", Name: "Сбербанк", Symbol: "SBER@TQBR", Mic: "TQBR"},
					{Ticker: "GAZP", Name: "Газпром", Symbol: "GAZP@TQBR", Mic: "TQBR"},
				},
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
		securityCache:       make([]models.SecurityInfo, 0),
	}

	if err := client.loadAssetCache(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify names are cached by ticker
	if name := client.GetInstrumentName("SBER"); name != "Сбербанк" {
		t.Errorf("Expected Сбербанк for SBER, got '%s'", name)
	}
	if name := client.GetInstrumentName("GAZP"); name != "Газпром" {
		t.Errorf("Expected Газпром for GAZP, got '%s'", name)
	}

	// Verify names are cached by full symbol
	if name := client.GetInstrumentName("SBER@TQBR"); name != "Сбербанк" {
		t.Errorf("Expected Сбербанк for SBER@TQBR, got '%s'", name)
	}
}

func TestGetQuotes_LocalTimezone(t *testing.T) {
	utcTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockMarketData := &mockMarketDataServiceClient{
		LastQuoteFunc: func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
			return &marketdata.QuoteResponse{
				Quote: &marketdata.Quote{
					Symbol:    in.Symbol,
					Last:      &decimal.Decimal{Value: "100"},
					Timestamp: ts,
				},
			}, nil
		},
	}

	client := &Client{
		marketDataClient: mockMarketData,
		assetMicCache:    map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:    map[string]float64{"SBER": 1},
	}

	quotes, err := client.GetQuotes("acc1", []string{"SBER"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	q, ok := quotes["SBER@TQBR"]
	if !ok {
		t.Fatal("Expected quote for SBER@TQBR")
	}

	if q.Timestamp.Location() != time.Local {
		t.Errorf("Expected quote timestamp in local timezone (%s), got %s",
			time.Local, q.Timestamp.Location())
	}
}

func TestGetSnapshots_LocalTimezone(t *testing.T) {
	utcTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ts := timestamppb.New(utcTime)

	mockMarketData := &mockMarketDataServiceClient{
		LastQuoteFunc: func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
			return &marketdata.QuoteResponse{
				Quote: &marketdata.Quote{
					Symbol:    in.Symbol,
					Last:      &decimal.Decimal{Value: "100"},
					Timestamp: ts,
				},
			}, nil
		},
	}

	client := &Client{
		marketDataClient: mockMarketData,
		assetMicCache:    map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:    map[string]float64{"SBER": 1},
	}

	quotes, err := client.GetSnapshots("acc1", []string{"SBER"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	q, ok := quotes["SBER"]
	if !ok {
		t.Fatal("Expected snapshot for SBER")
	}

	if q.Timestamp.Location() != time.Local {
		t.Errorf("Expected snapshot timestamp in local timezone (%s), got %s",
			time.Local, q.Timestamp.Location())
	}
}

func TestPlaceOrder_LotMultiplication_MultipleLots(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			// 5 lots * 10 lot size = 50 shares
			if in.Quantity == nil || in.Quantity.Value != "50" {
				t.Errorf("Expected API to receive quantity 50 (shares), got %s", in.Quantity.GetValue())
				return nil, fmt.Errorf("unexpected quantity: %s", in.Quantity.GetValue())
			}
			return &orders.OrderState{OrderId: "LOT-002"}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"GAZP": "GAZP@TQBR",
		},
		assetLotCache: map[string]float64{
			"GAZP":      10,
			"GAZP@TQBR": 10,
		},
	}

	txID, err := client.PlaceOrder("test-acc", "GAZP", "Sell", 5, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if txID != "LOT-002" {
		t.Errorf("Expected OrderId LOT-002, got %s", txID)
	}
}

func TestGetActiveOrders_ExtendedFields(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		GetOrdersFunc: func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
			return &orders.OrdersResponse{
				Orders: []*orders.OrderState{
					{
						OrderId: "STOP-1",
						Status:  orders.OrderStatus_ORDER_STATUS_NEW,
						Order: &orders.Order{
							Symbol:        "SBER",
							Side:          tradeapiv1.Side_SIDE_SELL,
							Type:          orders.OrderType_ORDER_TYPE_STOP,
							Quantity:      &decimal.Decimal{Value: "100"},
							StopPrice:     &decimal.Decimal{Value: "240.00"},
							StopCondition: orders.StopCondition_STOP_CONDITION_LAST_DOWN,
							ValidBefore:   orders.ValidBefore_VALID_BEFORE_GOOD_TILL_CANCEL,
						},
						ExecutedQuantity:  &decimal.Decimal{Value: "0"},
						RemainingQuantity: &decimal.Decimal{Value: "100"},
						TransactAt:        timestamppb.Now(),
					},
					{
						OrderId: "LIMIT-1",
						Status:  orders.OrderStatus_ORDER_STATUS_PARTIALLY_FILLED,
						Order: &orders.Order{
							Symbol:      "GAZP",
							Side:        tradeapiv1.Side_SIDE_BUY,
							Type:        orders.OrderType_ORDER_TYPE_LIMIT,
							Quantity:    &decimal.Decimal{Value: "200"},
							LimitPrice:  &decimal.Decimal{Value: "150.00"},
							ValidBefore: orders.ValidBefore_VALID_BEFORE_END_OF_DAY,
						},
						ExecutedQuantity:  &decimal.Decimal{Value: "50"},
						RemainingQuantity: &decimal.Decimal{Value: "150"},
						TransactAt:        timestamppb.Now(),
					},
					{
						OrderId: "SLTP-1",
						Status:  orders.OrderStatus_ORDER_STATUS_NEW,
						SltpOrder: &orders.SLTPOrder{
							Symbol:     "AAPL",
							Side:       tradeapiv1.Side_SIDE_SELL,
							SlPrice:    &decimal.Decimal{Value: "170.00"},
							TpPrice:    &decimal.Decimal{Value: "200.00"},
							QuantitySl: &decimal.Decimal{Value: "10"},
							QuantityTp: &decimal.Decimal{Value: "10"},
							ValidBefore: orders.ValidBefore_VALID_BEFORE_GOOD_TILL_CANCEL,
						},
						TransactAt: timestamppb.Now(),
					},
				},
			}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		instrumentNameCache: map[string]string{
			"SBER": "Сбербанк",
			"GAZP": "Газпром",
		},
	}

	activeOrders, err := client.GetActiveOrders("acc1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(activeOrders) != 3 {
		t.Fatalf("Expected 3 orders, got %d", len(activeOrders))
	}

	// Stop order checks
	stop := activeOrders[0]
	if stop.StopCondition != "Last Down" {
		t.Errorf("Expected StopCondition 'Last Down', got '%s'", stop.StopCondition)
	}
	if stop.StopPrice != "240.00" {
		t.Errorf("Expected StopPrice '240.00', got '%s'", stop.StopPrice)
	}
	if stop.Validity != "GTC" {
		t.Errorf("Expected Validity 'GTC', got '%s'", stop.Validity)
	}
	if stop.ExecutedQty != "0" {
		t.Errorf("Expected ExecutedQty '0', got '%s'", stop.ExecutedQty)
	}
	if stop.RemainingQty != "100" {
		t.Errorf("Expected RemainingQty '100', got '%s'", stop.RemainingQty)
	}

	// Limit order checks
	limit := activeOrders[1]
	if limit.LimitPrice != "150.00" {
		t.Errorf("Expected LimitPrice '150.00', got '%s'", limit.LimitPrice)
	}
	if limit.Validity != "Day" {
		t.Errorf("Expected Validity 'Day', got '%s'", limit.Validity)
	}
	if limit.ExecutedQty != "50" {
		t.Errorf("Expected ExecutedQty '50', got '%s'", limit.ExecutedQty)
	}
	if limit.RemainingQty != "150" {
		t.Errorf("Expected RemainingQty '150', got '%s'", limit.RemainingQty)
	}

	// SL/TP order checks
	sltp := activeOrders[2]
	if sltp.Type != "SL/TP" {
		t.Errorf("Expected Type 'SL/TP', got '%s'", sltp.Type)
	}
	if sltp.SLPrice != "170.00" {
		t.Errorf("Expected SLPrice '170.00', got '%s'", sltp.SLPrice)
	}
	if sltp.TPPrice != "200.00" {
		t.Errorf("Expected TPPrice '200.00', got '%s'", sltp.TPPrice)
	}
	if sltp.SLQty != "10" {
		t.Errorf("Expected SLQty '10', got '%s'", sltp.SLQty)
	}
	if sltp.TPQty != "10" {
		t.Errorf("Expected TPQty '10', got '%s'", sltp.TPQty)
	}
	if sltp.Validity != "GTC" {
		t.Errorf("Expected Validity 'GTC', got '%s'", sltp.Validity)
	}
}

func TestCancelOrder_Success(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		CancelOrderFunc: func(ctx context.Context, in *orders.CancelOrderRequest, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.AccountId != "test-acc" {
				t.Errorf("Expected AccountId test-acc, got %s", in.AccountId)
			}
			if in.OrderId != "order-123" {
				t.Errorf("Expected OrderId order-123, got %s", in.OrderId)
			}
			return &orders.OrderState{
				OrderId: "order-123",
				Status:  orders.OrderStatus_ORDER_STATUS_CANCELED,
			}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
	}

	err := client.CancelOrder("test-acc", "order-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCancelOrder_Error(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		CancelOrderFunc: func(ctx context.Context, in *orders.CancelOrderRequest, opts ...grpc.CallOption) (*orders.OrderState, error) {
			return nil, fmt.Errorf("order not found")
		},
	}

	client := &Client{
		ordersClient: mockOrders,
	}

	err := client.CancelOrder("test-acc", "order-999")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func newTestOrderClient(mock *mockOrdersServiceClient) *Client {
	return &Client{
		ordersClient:        mock,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 1},
		instrumentNameCache: make(map[string]string),
	}
}

func TestPlaceOrder_LimitOrder(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.Type != orders.OrderType_ORDER_TYPE_LIMIT {
				t.Errorf("Expected LIMIT type, got %v", in.Type)
			}
			if in.LimitPrice == nil || in.LimitPrice.Value != "250.5" {
				t.Errorf("Expected LimitPrice 250.5, got %v", in.LimitPrice)
			}
			return &orders.OrderState{OrderId: "LIM-1"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	id, err := client.PlaceOrder("acc1", "SBER", "Buy", 1, &models.OrderParams{
		OrderType:  models.OrderTypeLimit,
		LimitPrice: 250.5,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "LIM-1" {
		t.Errorf("Expected OrderId LIM-1, got %s", id)
	}
}

func TestPlaceOrder_StopLossOrder(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.Type != orders.OrderType_ORDER_TYPE_STOP {
				t.Errorf("Expected STOP type, got %v", in.Type)
			}
			if in.StopPrice == nil || in.StopPrice.Value != "240" {
				t.Errorf("Expected StopPrice 240, got %v", in.StopPrice)
			}
			// Sell SL: should be LAST_DOWN
			if in.StopCondition != orders.StopCondition_STOP_CONDITION_LAST_DOWN {
				t.Errorf("Expected LAST_DOWN for sell SL, got %v", in.StopCondition)
			}
			if in.ValidBefore != orders.ValidBefore_VALID_BEFORE_GOOD_TILL_CANCEL {
				t.Errorf("Expected GTC, got %v", in.ValidBefore)
			}
			return &orders.OrderState{OrderId: "SL-1"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	id, err := client.PlaceOrder("acc1", "SBER", "Sell", 1, &models.OrderParams{
		OrderType: models.OrderTypeStop,
		StopPrice: 240,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "SL-1" {
		t.Errorf("Expected OrderId SL-1, got %s", id)
	}
}

func TestPlaceOrder_StopLossBuy_UsesLastUp(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.StopCondition != orders.StopCondition_STOP_CONDITION_LAST_UP {
				t.Errorf("Expected LAST_UP for buy SL, got %v", in.StopCondition)
			}
			return &orders.OrderState{OrderId: "SL-2"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	_, err := client.PlaceOrder("acc1", "SBER", "Buy", 1, &models.OrderParams{
		OrderType: models.OrderTypeStop,
		StopPrice: 260,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestPlaceOrder_TakeProfitSell_UsesLastUp(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.Type != orders.OrderType_ORDER_TYPE_STOP {
				t.Errorf("Expected STOP type for TP, got %v", in.Type)
			}
			// Sell TP: should be LAST_UP (sell when price rises)
			if in.StopCondition != orders.StopCondition_STOP_CONDITION_LAST_UP {
				t.Errorf("Expected LAST_UP for sell TP, got %v", in.StopCondition)
			}
			return &orders.OrderState{OrderId: "TP-1"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	_, err := client.PlaceOrder("acc1", "SBER", "Sell", 1, &models.OrderParams{
		OrderType: models.OrderTypeTakeProfit,
		StopPrice: 280,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestPlaceSLTPOrder_Success(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceSLTPOrderFunc: func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.AccountId != "acc1" {
				t.Errorf("Expected AccountId acc1, got %s", in.AccountId)
			}
			if in.Side != tradeapiv1.Side_SIDE_SELL {
				t.Errorf("Expected SELL side, got %v", in.Side)
			}
			if in.SlPrice == nil || in.SlPrice.Value != "230" {
				t.Errorf("Expected SL price 230, got %v", in.SlPrice)
			}
			if in.TpPrice == nil || in.TpPrice.Value != "280" {
				t.Errorf("Expected TP price 280, got %v", in.TpPrice)
			}
			if in.QuantitySl == nil || in.QuantitySl.Value != "10" {
				t.Errorf("Expected SL qty 10, got %v", in.QuantitySl)
			}
			if in.QuantityTp == nil || in.QuantityTp.Value != "10" {
				t.Errorf("Expected TP qty 10, got %v", in.QuantityTp)
			}
			if in.ValidBefore != orders.ValidBefore_VALID_BEFORE_GOOD_TILL_CANCEL {
				t.Errorf("Expected GTC, got %v", in.ValidBefore)
			}
			return &orders.OrderState{OrderId: "SLTP-1"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	id, err := client.PlaceSLTPOrder("acc1", "SBER", "Sell", 10, 230, 10, 280)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "SLTP-1" {
		t.Errorf("Expected OrderId SLTP-1, got %s", id)
	}
}

func TestPlaceSLTPOrder_OnlySL(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceSLTPOrderFunc: func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.SlPrice == nil {
				t.Error("Expected SL price to be set")
			}
			if in.TpPrice != nil {
				t.Error("Expected TP price to be nil when tpPrice=0")
			}
			return &orders.OrderState{OrderId: "SL-ONLY"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)

	id, err := client.PlaceSLTPOrder("acc1", "SBER", "Sell", 5, 230, 5, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "SL-ONLY" {
		t.Errorf("Expected OrderId SL-ONLY, got %s", id)
	}
}

func TestPlaceSLTPOrder_LotMultiplication(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceSLTPOrderFunc: func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
			// 2 lots * 10 lot size = 20 shares
			if in.QuantitySl.Value != "20" {
				t.Errorf("Expected SL qty 20 (2 lots * 10), got %s", in.QuantitySl.Value)
			}
			if in.QuantityTp.Value != "20" {
				t.Errorf("Expected TP qty 20 (2 lots * 10), got %s", in.QuantityTp.Value)
			}
			return &orders.OrderState{OrderId: "SLTP-LOT"}, nil
		},
	}
	client := newTestOrderClient(mockOrders)
	client.assetLotCache["SBER"] = 10 // Override lot size for this test

	_, err := client.PlaceSLTPOrder("acc1", "SBER", "Buy", 2, 230, 2, 280)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestPlaceSLTPOrder_Error(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceSLTPOrderFunc: func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
			return nil, fmt.Errorf("insufficient margin")
		},
	}
	client := newTestOrderClient(mockOrders)

	_, err := client.PlaceSLTPOrder("acc1", "SBER", "Sell", 10, 230, 10, 280)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// --- Phase 6: New unit tests for untested methods ---

func TestGetBars(t *testing.T) {
	mockMD := &mockMarketDataServiceClient{
		BarsFunc: func(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error) {
			return &marketdata.BarsResponse{
				Symbol: in.Symbol,
				Bars: []*marketdata.Bar{
					{
						Timestamp: timestamppb.New(time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)),
						Open:      &decimal.Decimal{Value: "280.50"},
						High:      &decimal.Decimal{Value: "285.00"},
						Low:       &decimal.Decimal{Value: "278.00"},
						Close:     &decimal.Decimal{Value: "283.00"},
						Volume:    &decimal.Decimal{Value: "15000"},
					},
					{
						Timestamp: timestamppb.New(time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)),
						Open:      &decimal.Decimal{Value: "283.00"},
						High:      &decimal.Decimal{Value: "286.00"},
						Low:       &decimal.Decimal{Value: "282.00"},
						Close:     &decimal.Decimal{Value: "284.50"},
						Volume:    &decimal.Decimal{Value: "12000"},
					},
				},
			}, nil
		},
	}

	client := &Client{
		marketDataClient:    mockMD,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 10},
		instrumentNameCache: make(map[string]string),
	}

	bars, err := client.GetBars("acc1", "SBER", marketdata.TimeFrame_TIME_FRAME_H1, time.Now().AddDate(0, 0, -7), time.Now())
	if err != nil {
		t.Fatalf("GetBars error: %v", err)
	}
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].Open != 280.50 {
		t.Errorf("expected Open=280.50, got %v", bars[0].Open)
	}
	if bars[0].High != 285.00 {
		t.Errorf("expected High=285.00, got %v", bars[0].High)
	}
	if bars[0].Volume != 15000 {
		t.Errorf("expected Volume=15000, got %v", bars[0].Volume)
	}
	if bars[1].Close != 284.50 {
		t.Errorf("expected Close=284.50, got %v", bars[1].Close)
	}
}

func TestGetAssetInfo(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return &assets.GetAssetResponse{
				Board:         "TQBR",
				Ticker:        "SBER",
				Mic:           "TQBR",
				Name:          "Сбер Банк",
				Type:          "stock",
				Decimals:      2,
				LotSize:       &decimal.Decimal{Value: "10"},
				QuoteCurrency: "RUB",
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 10},
		instrumentNameCache: make(map[string]string),
	}

	info, err := client.GetAssetInfo("acc1", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetInfo error: %v", err)
	}
	if info.Name != "Сбер Банк" {
		t.Errorf("expected name 'Сбер Банк', got %q", info.Name)
	}
	if info.Board != "TQBR" {
		t.Errorf("expected board TQBR, got %q", info.Board)
	}
	if info.LotSize != "10" {
		t.Errorf("expected lot size '10', got %q", info.LotSize)
	}
	if info.Decimals != 2 {
		t.Errorf("expected decimals 2, got %d", info.Decimals)
	}
}

func TestGetAssetParams(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetParamsFunc: func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
			return &assets.GetAssetParamsResponse{
				Symbol:        in.Symbol,
				Longable:      &assets.Longable{Value: assets.Longable_AVAILABLE},
				Shortable:     &assets.Shortable{Value: assets.Shortable_NOT_AVAILABLE},
				LongRiskRate:  &decimal.Decimal{Value: "0.25"},
				ShortRiskRate: &decimal.Decimal{Value: "0.50"},
			}, nil
		},
		// GetAsset needed for getFullSymbol fallback
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return &assets.GetAssetResponse{Ticker: "SBER", Board: "TQBR", LotSize: &decimal.Decimal{Value: "10"}}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 10},
		instrumentNameCache: make(map[string]string),
	}

	params, err := client.GetAssetParams("acc1", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetParams error: %v", err)
	}
	if params.Longable != "Available" {
		t.Errorf("expected Longable 'Available', got %q", params.Longable)
	}
	if params.Shortable != "Not Available" {
		t.Errorf("expected Shortable 'Not Available', got %q", params.Shortable)
	}
	if params.LongRiskRate != "0.25" {
		t.Errorf("expected LongRiskRate '0.25', got %q", params.LongRiskRate)
	}
}

func TestGetSchedule(t *testing.T) {
	start := time.Date(2026, 4, 7, 7, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 7, 15, 40, 0, 0, time.UTC)

	mockAssets := &mockAssetsServiceClient{
		ScheduleFunc: func(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error) {
			return &assets.ScheduleResponse{
				Symbol: in.Symbol,
				Sessions: []*assets.ScheduleResponse_Sessions{
					{
						Type: "main",
						Interval: &interval.Interval{
							StartTime: timestamppb.New(start),
							EndTime:   timestamppb.New(end),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	sessions, err := client.GetSchedule("SBER@TQBR")
	if err != nil {
		t.Fatalf("GetSchedule error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Type != "main" {
		t.Errorf("expected type 'main', got %q", sessions[0].Type)
	}
	if sessions[0].StartTime.UTC() != start {
		t.Errorf("expected start %v, got %v", start, sessions[0].StartTime.UTC())
	}
	if sessions[0].EndTime.UTC() != end {
		t.Errorf("expected end %v, got %v", end, sessions[0].EndTime.UTC())
	}
}

func TestGetFullSymbol_CacheHit(t *testing.T) {
	client := &Client{
		assetMicCache: map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache: map[string]float64{"SBER": 10, "SBER@TQBR": 10},
		instrumentNameCache: make(map[string]string),
	}

	result := client.getFullSymbol("SBER", "acc1")
	if result != "SBER@TQBR" {
		t.Errorf("expected SBER@TQBR, got %s", result)
	}
}

func TestGetFullSymbol_AlreadyFullSymbol(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return &assets.GetAssetResponse{
				Ticker:  "SBER",
				Board:   "TQBR",
				LotSize: &decimal.Decimal{Value: "10"},
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{},
		assetLotCache:       map[string]float64{},
		instrumentNameCache: make(map[string]string),
	}

	// Already contains @, should return as-is (but may trigger lot size fetch)
	result := client.getFullSymbol("SBER@TQBR", "acc1")
	if result != "SBER@TQBR" {
		t.Errorf("expected SBER@TQBR, got %s", result)
	}
}

func TestGetFullSymbol_CacheMissFallback(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return &assets.GetAssetResponse{
				Ticker:  "YNDX",
				Board:   "TQBR",
				LotSize: &decimal.Decimal{Value: "1"},
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{}, // empty cache
		assetLotCache:       map[string]float64{},
		instrumentNameCache: make(map[string]string),
	}

	result := client.getFullSymbol("YNDX", "acc1")
	if result != "YNDX@TQBR" {
		t.Errorf("expected YNDX@TQBR, got %s", result)
	}

	// Should now be cached
	client.assetMutex.RLock()
	cached := client.assetMicCache["YNDX"]
	lot := client.assetLotCache["YNDX"]
	client.assetMutex.RUnlock()

	if cached != "YNDX@TQBR" {
		t.Errorf("expected cached YNDX@TQBR, got %s", cached)
	}
	if lot != 1 {
		t.Errorf("expected lot size 1, got %v", lot)
	}
}
