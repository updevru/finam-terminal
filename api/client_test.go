package api

import (
	"context"
	"fmt"
	"testing"

	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	tradeapiv1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockMarketDataServiceClient is a manual mock for marketdata.MarketDataServiceClient
type mockMarketDataServiceClient struct {
	marketdata.MarketDataServiceClient
	LastQuoteFunc func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error)
}

func (m *mockMarketDataServiceClient) LastQuote(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
	return m.LastQuoteFunc(ctx, in, opts...)
}

// mockAssetsServiceClient is a manual mock for assets.AssetsServiceClient
type mockAssetsServiceClient struct {
	assets.AssetsServiceClient
	AssetsFunc   func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error)
	GetAssetFunc func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error)
}

func (m *mockAssetsServiceClient) Assets(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
	return m.AssetsFunc(ctx, in, opts...)
}

func (m *mockAssetsServiceClient) GetAsset(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
	return m.GetAssetFunc(ctx, in, opts...)
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
	PlaceOrderFunc func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error)
	GetOrdersFunc  func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error)
}

func (m *mockOrdersServiceClient) PlaceOrder(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
	return m.PlaceOrderFunc(ctx, in, opts...)
}

func (m *mockOrdersServiceClient) GetOrders(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
	return m.GetOrdersFunc(ctx, in, opts...)
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

	txID, err := client.PlaceOrder("test-acc", "SBER", "Buy", 10)
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

	_, err := client.PlaceOrder("test-acc", "SBER", "Buy", 10)
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
						Symbol:       "GAZP",
						Quantity:     &decimal.Decimal{Value: "10"},
						AveragePrice: &decimal.Decimal{Value: "150"},
						CurrentPrice: &decimal.Decimal{Value: "160"},
						DailyPnl:     &decimal.Decimal{Value: "100"},
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
										}, nil			},
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
								Symbol:   "SBER@TQBR",
								Last:     &decimal.Decimal{Value: "250.50"},
								LastSize: &decimal.Decimal{Value: "10"},
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
						TradeId: "T1",
						Symbol:  "SBER",
						Price:   &decimal.Decimal{Value: "250.00"},
						Size:    &decimal.Decimal{Value: "10"},
						Side:    tradeapiv1.Side_SIDE_BUY,
						Timestamp: timestamppb.Now(),
					},
				},
			}, nil
		},
	}

	client := &Client{
		accountsClient: mockAccounts,
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
	if activeOrders[0].Status != "New" {
		t.Errorf("Expected Status New, got %s", activeOrders[0].Status)
	}
	if activeOrders[0].Side != "Sell" {
		t.Errorf("Expected Side Sell, got %s", activeOrders[0].Side)
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
					Ticker: "SBER",
					Board:  "TQBR",
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
		accountsClient: mockAccounts,
		assetsClient:   mockAssets,
		assetMicCache:  make(map[string]string),
		assetLotCache:  make(map[string]float64),
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
	txID, err := client.PlaceOrder("test-acc", "SBER", "Buy", 1)
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

	txID, err := client.PlaceOrder("test-acc", "GAZP", "Sell", 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if txID != "LOT-002" {
		t.Errorf("Expected OrderId LOT-002, got %s", txID)
	}
}
		