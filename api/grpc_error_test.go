package api

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestLogGRPCError_BasicFormat(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{
		conn: nil, // conn.Target() will be tested separately
	}

	err := grpcstatus.Error(codes.NotFound, "instrument not found")
	client.logGRPCError("MarketDataService", "LastQuote", err, "Symbol: SBER@TQBR")

	output := buf.String()

	// Verify log contains all required parts
	checks := []string{
		"[ERROR]",
		"MarketDataService.LastQuote failed",
		"Symbol: SBER@TQBR",
		"gRPC code: NotFound",
		"Message: instrument not found",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestLogGRPCError_MultipleParams(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{}

	err := grpcstatus.Error(codes.PermissionDenied, "access denied")
	client.logGRPCError("AccountsService", "Trades", err, "AccountId: acc123", "Interval: 2025-01-01/2025-01-31")

	output := buf.String()

	checks := []string{
		"AccountsService.Trades failed",
		"AccountId: acc123",
		"Interval: 2025-01-01/2025-01-31",
		"gRPC code: PermissionDenied",
		"Message: access denied",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestLogGRPCError_NoParams(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{}

	err := grpcstatus.Error(codes.Unauthenticated, "invalid token")
	client.logGRPCError("AuthService", "Auth", err)

	output := buf.String()

	if !strings.Contains(output, "AuthService.Auth failed") {
		t.Errorf("Log output missing service.method.\nGot: %s", output)
	}
	if !strings.Contains(output, "gRPC code: Unauthenticated") {
		t.Errorf("Log output missing gRPC code.\nGot: %s", output)
	}
	if !strings.Contains(output, "Message: invalid token") {
		t.Errorf("Log output missing message.\nGot: %s", output)
	}
}

func TestLogGRPCError_NonGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{}

	err := os.ErrNotExist // A non-gRPC error
	client.logGRPCError("AssetsService", "GetAsset", err, "Symbol: GAZP")

	output := buf.String()

	if !strings.Contains(output, "AssetsService.GetAsset failed") {
		t.Errorf("Log output missing service.method.\nGot: %s", output)
	}
	// For non-gRPC errors, status.FromError returns codes.Unknown
	if !strings.Contains(output, "gRPC code: Unknown") {
		t.Errorf("Log output missing gRPC code for non-gRPC error.\nGot: %s", output)
	}
}

func TestAuthenticate_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAuth := &mockAuthServiceClient{
		AuthFunc: func(ctx context.Context, in *auth.AuthRequest, opts ...grpc.CallOption) (*auth.AuthResponse, error) {
			return nil, grpcstatus.Error(codes.Unauthenticated, "bad token")
		},
	}

	client := &Client{authClient: mockAuth}
	err := client.authenticate("secret")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AuthService.Auth failed",
		"gRPC code: Unauthenticated",
		"Message: bad token",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
	// Token must NOT appear in log
	if strings.Contains(output, "secret") {
		t.Errorf("Log must not contain the secret token.\nGot: %s", output)
	}
}

func TestLoadAssetCache_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAssets := &mockAssetsServiceClient{
		AssetsFunc: func(ctx context.Context, in *assets.AssetsRequest, opts ...grpc.CallOption) (*assets.AssetsResponse, error) {
			return nil, grpcstatus.Error(codes.Internal, "server error")
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
		securityCache:       nil,
	}
	err := client.loadAssetCache()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AssetsService.Assets failed",
		"gRPC code: Internal",
		"Message: server error",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetAccounts_TokenDetails_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAuth := &mockAuthServiceClient{
		TokenDetailsFunc: func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
			return nil, grpcstatus.Error(codes.PermissionDenied, "expired")
		},
	}

	client := &Client{authClient: mockAuth}
	_, err := client.GetAccounts()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AuthService.TokenDetails failed",
		"gRPC code: PermissionDenied",
		"Message: expired",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
	// Token must NOT appear in log
	if strings.Contains(output, "token") && strings.Contains(output, "Token:") {
		t.Errorf("Log must not contain the token value")
	}
}

func TestGetAccounts_GetAccountLoop_UsesHelper(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAuth := &mockAuthServiceClient{
		TokenDetailsFunc: func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
			return &auth.TokenDetailsResponse{AccountIds: []string{"acc-fail"}}, nil
		},
	}
	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return nil, grpcstatus.Error(codes.NotFound, "account not found")
		},
	}

	client := &Client{authClient: mockAuth, accountsClient: mockAccounts}
	accs, err := client.GetAccounts()

	// Should not return error — adds account with LoadError instead
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(accs) != 1 || accs[0].LoadError == "" {
		t.Errorf("Expected account with LoadError set")
	}

	output := buf.String()
	checks := []string{
		"AccountsService.GetAccount failed",
		"AccountId: acc-fail",
		"gRPC code: NotFound",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetAccountDetails_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return nil, grpcstatus.Error(codes.Internal, "db error")
		},
	}

	client := &Client{accountsClient: mockAccounts}
	_, _, err := client.GetAccountDetails("acc-xyz")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AccountsService.GetAccount failed",
		"AccountId: acc-xyz",
		"gRPC code: Internal",
		"Message: db error",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetTradeHistory_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAccounts := &mockAccountsServiceClient{
		TradesFunc: func(ctx context.Context, in *accounts.TradesRequest, opts ...grpc.CallOption) (*accounts.TradesResponse, error) {
			return nil, grpcstatus.Error(codes.Unavailable, "service down")
		},
	}

	client := &Client{accountsClient: mockAccounts}
	_, err := client.GetTradeHistory("acc-hist")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AccountsService.Trades failed",
		"AccountId: acc-hist",
		"gRPC code: Unavailable",
		"Message: service down",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetActiveOrders_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockOrders := &mockOrdersServiceClient{
		GetOrdersFunc: func(ctx context.Context, in *orders.OrdersRequest, opts ...grpc.CallOption) (*orders.OrdersResponse, error) {
			return nil, grpcstatus.Error(codes.DeadlineExceeded, "timeout")
		},
	}

	client := &Client{ordersClient: mockOrders}
	_, err := client.GetActiveOrders("acc-orders")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"OrdersService.GetOrders failed",
		"AccountId: acc-orders",
		"gRPC code: DeadlineExceeded",
		"Message: timeout",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetQuotes_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockMarketData := &mockMarketDataServiceClient{
		LastQuoteFunc: func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
			return nil, grpcstatus.Error(codes.NotFound, "no data")
		},
	}

	client := &Client{
		marketDataClient: mockMarketData,
		assetMicCache:    map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:    map[string]float64{"SBER": 1},
	}

	_, _ = client.GetQuotes("acc1", []string{"SBER"})

	output := buf.String()
	checks := []string{
		"MarketDataService.LastQuote failed",
		"Symbol: SBER@TQBR",
		"gRPC code: NotFound",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetSnapshots_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockMarketData := &mockMarketDataServiceClient{
		LastQuoteFunc: func(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
			return nil, grpcstatus.Error(codes.Unavailable, "service down")
		},
	}

	client := &Client{
		marketDataClient: mockMarketData,
		assetMicCache:    map[string]string{"GAZP": "GAZP@TQBR"},
		assetLotCache:    map[string]float64{"GAZP": 1},
	}

	_, _ = client.GetSnapshots("acc1", []string{"GAZP"})

	output := buf.String()
	checks := []string{
		"MarketDataService.LastQuote failed",
		"Symbol: GAZP@TQBR",
		"gRPC code: Unavailable",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetBars_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Need a mock for Bars method
	client := &Client{
		marketDataClient: &mockMarketDataBarsClient{
			BarsFunc: func(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error) {
				return nil, grpcstatus.Error(codes.InvalidArgument, "bad timeframe")
			},
		},
		assetMicCache: map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache: map[string]float64{"SBER": 1},
	}

	now := time.Now()
	_, err := client.GetBars("acc1", "SBER", marketdata.TimeFrame_TIME_FRAME_D, now.AddDate(0, 0, -7), now)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"MarketDataService.Bars failed",
		"Symbol: SBER@TQBR",
		"gRPC code: InvalidArgument",
		"Message: bad timeframe",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

// mockMarketDataBarsClient extends mock to include Bars method
type mockMarketDataBarsClient struct {
	marketdata.MarketDataServiceClient
	BarsFunc func(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error)
}

func (m *mockMarketDataBarsClient) Bars(ctx context.Context, in *marketdata.BarsRequest, opts ...grpc.CallOption) (*marketdata.BarsResponse, error) {
	return m.BarsFunc(ctx, in, opts...)
}

func (m *mockMarketDataBarsClient) LastQuote(ctx context.Context, in *marketdata.QuoteRequest, opts ...grpc.CallOption) (*marketdata.QuoteResponse, error) {
	return nil, nil // Not used in bars tests
}

func TestGetFullSymbol_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return nil, grpcstatus.Error(codes.NotFound, "asset not found")
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	_ = client.getFullSymbol("UNKNOWN", "acc1")

	output := buf.String()
	checks := []string{
		"AssetsService.GetAsset failed",
		"Symbol: UNKNOWN",
		"AccountId: acc1",
		"gRPC code: NotFound",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestFetchLotSize_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return nil, grpcstatus.Error(codes.Internal, "server error")
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	client.fetchLotSize("SBER@TQBR", "acc2")

	output := buf.String()
	checks := []string{
		"AssetsService.GetAsset failed",
		"Symbol: SBER@TQBR",
		"AccountId: acc2",
		"gRPC code: Internal",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestGetAssetInfo_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockAssets := &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			return nil, grpcstatus.Error(codes.Unavailable, "service unavailable")
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 1},
		instrumentNameCache: make(map[string]string),
	}

	_, err := client.GetAssetInfo("acc1", "SBER")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AssetsService.GetAsset failed",
		"Symbol: SBER@TQBR",
		"AccountId: acc1",
		"gRPC code: Unavailable",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

// mockAssetsGetAssetParamsClient extends mockAssetsServiceClient with GetAssetParams
type mockAssetsGetAssetParamsClient struct {
	assets.AssetsServiceClient
	GetAssetFunc       func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error)
	GetAssetParamsFunc func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error)
}

func (m *mockAssetsGetAssetParamsClient) GetAsset(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
	return m.GetAssetFunc(ctx, in, opts...)
}

func (m *mockAssetsGetAssetParamsClient) GetAssetParams(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
	return m.GetAssetParamsFunc(ctx, in, opts...)
}

func TestGetAssetParams_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{
		assetsClient: &mockAssetsGetAssetParamsClient{
			GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
				return &assets.GetAssetResponse{Ticker: "SBER", Board: "TQBR"}, nil
			},
			GetAssetParamsFunc: func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
				return nil, grpcstatus.Error(codes.PermissionDenied, "no access")
			},
		},
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 1},
		instrumentNameCache: make(map[string]string),
	}

	_, err := client.GetAssetParams("acc1", "SBER")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AssetsService.GetAssetParams failed",
		"Symbol: SBER@TQBR",
		"AccountId: acc1",
		"gRPC code: PermissionDenied",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

// mockAssetsScheduleClient extends mock with Schedule
type mockAssetsScheduleClient struct {
	assets.AssetsServiceClient
	ScheduleFunc func(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error)
	GetAssetFunc func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error)
}

func (m *mockAssetsScheduleClient) Schedule(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error) {
	return m.ScheduleFunc(ctx, in, opts...)
}

func (m *mockAssetsScheduleClient) GetAsset(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
	if m.GetAssetFunc != nil {
		return m.GetAssetFunc(ctx, in, opts...)
	}
	return nil, grpcstatus.Error(codes.Unimplemented, "not mocked")
}

func TestGetSchedule_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	client := &Client{
		assetsClient: &mockAssetsScheduleClient{
			ScheduleFunc: func(ctx context.Context, in *assets.ScheduleRequest, opts ...grpc.CallOption) (*assets.ScheduleResponse, error) {
				return nil, grpcstatus.Error(codes.NotFound, "no schedule")
			},
		},
	}

	_, err := client.GetSchedule("SBER@TQBR")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"AssetsService.Schedule failed",
		"Symbol: SBER@TQBR",
		"gRPC code: NotFound",
		"Message: no schedule",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}

func TestPlaceOrder_LogsGRPCError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			return nil, grpcstatus.Error(codes.InvalidArgument, "bad order")
		},
	}

	client := &Client{
		ordersClient:  mockOrders,
		assetMicCache: map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache: map[string]float64{"SBER": 10, "SBER@TQBR": 10},
	}

	_, err := client.PlaceOrder("acc1", "SBER", "Buy", 5, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	output := buf.String()
	checks := []string{
		"OrdersService.PlaceOrder failed",
		"AccountId: acc1",
		"Symbol: SBER@TQBR",
		"Side: Buy",
		"Quantity: 5",
		"gRPC code: InvalidArgument",
		"Message: bad order",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Log output missing %q.\nGot: %s", check, output)
		}
	}
}
