package api

import (
	"context"
	"testing"

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
}

func (m *mockAccountsServiceClient) GetAccount(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
	return m.GetAccountFunc(ctx, in, opts...)
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
}

func (m *mockOrdersServiceClient) PlaceOrder(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
	return m.PlaceOrderFunc(ctx, in, opts...)
}

func TestPlaceOrder_Success(t *testing.T) {
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			if in.AccountId != "test-acc" {
				return nil, grpc.ErrClientConnClosing // Just a dummy error
			}
			if in.Symbol != "SBER@TQBR" {
				return nil, grpc.ErrClientConnClosing
			}
			if in.Side != tradeapiv1.Side_SIDE_BUY {
				return nil, grpc.ErrClientConnClosing
			}
			if in.Quantity == nil || in.Quantity.Value != "10" {
				return nil, grpc.ErrClientConnClosing
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
			return nil, grpc.ErrClientConnClosing
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
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
				return nil, grpc.ErrClientConnClosing
			}
			if in.Quantity.Value != "5" {
				return nil, grpc.ErrClientConnClosing
			}
			return &orders.OrderState{OrderId: "999"}, nil
		},
	}

	client := &Client{
		ordersClient: mockOrders,
		assetMicCache: map[string]string{
			"SBER": "SBER@TQBR",
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
				assetsClient:  mockAssets,
				assetMicCache: make(map[string]string),
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
						return nil, grpc.ErrClientConnClosing
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
			}
		
			// Test GetSnapshots
			quotes, err := client.GetSnapshots([]string{"SBER"})
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
		