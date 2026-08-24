package api

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetAssetParams_MapsTradeLotSize verifies that trade_lot_size (API 2.18.1)
// is surfaced on models.AssetParams.
func TestGetAssetParams_MapsTradeLotSize(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetParamsFunc: func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
			return &assets.GetAssetParamsResponse{
				Symbol:       in.Symbol,
				TradeLotSize: 5,
			}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 10, "SBER@TQBR": 10},
		tradeLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	params, err := client.GetAssetParams("acc1", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetParams error: %v", err)
	}
	if params.TradeLotSize != 5 {
		t.Errorf("expected TradeLotSize 5, got %d", params.TradeLotSize)
	}
}

// TestGetAssetParams_TradeLotSizeAbsent verifies that a response without
// trade_lot_size maps to 0 ("value is absent" per the API docs).
func TestGetAssetParams_TradeLotSizeAbsent(t *testing.T) {
	mockAssets := &mockAssetsServiceClient{
		GetAssetParamsFunc: func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
			return &assets.GetAssetParamsResponse{Symbol: in.Symbol}, nil
		},
	}

	client := &Client{
		assetsClient:        mockAssets,
		assetMicCache:       map[string]string{"SBER": "SBER@TQBR"},
		assetLotCache:       map[string]float64{"SBER": 10, "SBER@TQBR": 10},
		tradeLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}

	params, err := client.GetAssetParams("acc1", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetParams error: %v", err)
	}
	if params.TradeLotSize != 0 {
		t.Errorf("expected TradeLotSize 0 when absent, got %d", params.TradeLotSize)
	}
}

// TestGetLotSize_PrefersTradeLot verifies the two-tier lot cache: the trade lot
// from GetAssetParams wins over the asset lot from GetAsset, both by ticker and
// by full symbol.
func TestGetLotSize_PrefersTradeLot(t *testing.T) {
	tests := []struct {
		name     string
		tradeLot map[string]float64
		want     float64
	}{
		{
			name:     "by ticker",
			tradeLot: map[string]float64{"SBER": 5, "SBER@TQBR": 5},
			want:     5,
		},
		{
			name:     "by full symbol only",
			tradeLot: map[string]float64{"SBER@TQBR": 5},
			want:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				assetMicCache: map[string]string{"SBER": "SBER@TQBR"},
				assetLotCache: map[string]float64{"SBER": 10, "SBER@TQBR": 10},
				tradeLotCache: tt.tradeLot,
			}

			if got := client.GetLotSize("SBER"); got != tt.want {
				t.Errorf("GetLotSize(SBER) = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetLotSize_FallsBackToAssetLot verifies that a zero trade lot ("API has no
// value") does not shadow the asset lot size.
func TestGetLotSize_FallsBackToAssetLot(t *testing.T) {
	tests := []struct {
		name     string
		tradeLot map[string]float64
		want     float64
	}{
		{
			name:     "trade lot is zero",
			tradeLot: map[string]float64{"SBER": 0, "SBER@TQBR": 0},
			want:     10,
		},
		{
			name:     "trade lot not cached",
			tradeLot: map[string]float64{},
			want:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				assetMicCache: map[string]string{"SBER": "SBER@TQBR"},
				assetLotCache: map[string]float64{"SBER": 10, "SBER@TQBR": 10},
				tradeLotCache: tt.tradeLot,
			}

			if got := client.GetLotSize("SBER"); got != tt.want {
				t.Errorf("GetLotSize(SBER) = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetLotSize_UnknownTicker verifies that an entirely unknown ticker still
// reports 0 (callers treat that as "no lot information").
func TestGetLotSize_UnknownTicker(t *testing.T) {
	client := &Client{
		assetMicCache: map[string]string{},
		assetLotCache: map[string]float64{},
		tradeLotCache: map[string]float64{},
	}

	if got := client.GetLotSize("UNKNOWN"); got != 0 {
		t.Errorf("GetLotSize(UNKNOWN) = %v, want 0", got)
	}
}

// tradeLotMocks builds an assets mock that serves an asset lot of 10 and a
// trade lot of tradeLot, counting the calls of both RPCs.
type tradeLotMocks struct {
	assets          *mockAssetsServiceClient
	assetCalls      atomic.Int64
	paramCalls      atomic.Int64
	paramsErr       error
	tradeLotToServe int64
}

func newTradeLotMocks(tradeLot int64) *tradeLotMocks {
	m := &tradeLotMocks{tradeLotToServe: tradeLot}
	m.assets = &mockAssetsServiceClient{
		GetAssetFunc: func(ctx context.Context, in *assets.GetAssetRequest, opts ...grpc.CallOption) (*assets.GetAssetResponse, error) {
			m.assetCalls.Add(1)
			return &assets.GetAssetResponse{
				Ticker:  "SBER",
				Board:   "TQBR",
				LotSize: &decimal.Decimal{Value: "10"},
			}, nil
		},
		GetAssetParamsFunc: func(ctx context.Context, in *assets.GetAssetParamsRequest, opts ...grpc.CallOption) (*assets.GetAssetParamsResponse, error) {
			m.paramCalls.Add(1)
			if m.paramsErr != nil {
				return nil, m.paramsErr
			}
			return &assets.GetAssetParamsResponse{
				Symbol:       in.Symbol,
				TradeLotSize: m.tradeLotToServe,
			}, nil
		},
	}
	return m
}

func newTradeLotClient(m *tradeLotMocks) *Client {
	return &Client{
		assetsClient:        m.assets,
		assetMicCache:       make(map[string]string),
		assetLotCache:       make(map[string]float64),
		tradeLotCache:       make(map[string]float64),
		instrumentNameCache: make(map[string]string),
	}
}

// TestFetchLotSize_PopulatesBothTiers verifies that resolving a cold full symbol
// fills the asset lot cache (GetAsset) and the trade lot cache (GetAssetParams).
func TestFetchLotSize_PopulatesBothTiers(t *testing.T) {
	m := newTradeLotMocks(5)
	client := newTradeLotClient(m)

	client.getFullSymbol("SBER@TQBR", "acc1")

	client.assetMutex.RLock()
	assetLot := client.assetLotCache["SBER@TQBR"]
	tradeLot, hasTrade := client.tradeLotCache["SBER@TQBR"]
	client.assetMutex.RUnlock()

	if assetLot != 10 {
		t.Errorf("assetLotCache[SBER@TQBR] = %v, want 10", assetLot)
	}
	if !hasTrade || tradeLot != 5 {
		t.Errorf("tradeLotCache[SBER@TQBR] = %v (present=%v), want 5", tradeLot, hasTrade)
	}
	if got := client.GetLotSize("SBER@TQBR"); got != 5 {
		t.Errorf("GetLotSize(SBER@TQBR) = %v, want 5 (trade lot)", got)
	}
}

// TestFetchLotSize_ColdTickerPopulatesBothTiers verifies the same for a bare
// ticker, which resolves through the MIC cache branch of getFullSymbol.
func TestFetchLotSize_ColdTickerPopulatesBothTiers(t *testing.T) {
	m := newTradeLotMocks(5)
	client := newTradeLotClient(m)

	if full := client.getFullSymbol("SBER", "acc1"); full != "SBER@TQBR" {
		t.Fatalf("getFullSymbol(SBER) = %q, want SBER@TQBR", full)
	}

	if got := client.GetLotSize("SBER"); got != 5 {
		t.Errorf("GetLotSize(SBER) = %v, want 5 (trade lot)", got)
	}
}

// TestFetchLotSize_NegativeCacheStopsRefetching verifies that a trade lot of 0
// ("API has no value") is cached, so warm lookups issue no further RPC.
func TestFetchLotSize_NegativeCacheStopsRefetching(t *testing.T) {
	m := newTradeLotMocks(0)
	client := newTradeLotClient(m)

	client.getFullSymbol("SBER@TQBR", "acc1")
	assetCalls, paramCalls := m.assetCalls.Load(), m.paramCalls.Load()
	if assetCalls != 1 || paramCalls != 1 {
		t.Fatalf("cold resolve: GetAsset=%d GetAssetParams=%d, want 1 and 1", assetCalls, paramCalls)
	}

	// Warm path: nothing may hit the network again.
	client.getFullSymbol("SBER@TQBR", "acc1")
	if got := m.assetCalls.Load(); got != assetCalls {
		t.Errorf("GetAsset called %d times on warm path, want %d", got, assetCalls)
	}
	if got := m.paramCalls.Load(); got != paramCalls {
		t.Errorf("GetAssetParams called %d times on warm path, want %d", got, paramCalls)
	}

	if got := client.GetLotSize("SBER@TQBR"); got != 10 {
		t.Errorf("GetLotSize(SBER@TQBR) = %v, want 10 (asset lot fallback)", got)
	}
}

// TestFetchLotSize_ParamsErrorKeepsAssetLotAndRetries verifies that a failing
// GetAssetParams neither blocks the asset lot nor poisons the cache: the next
// miss tries again.
func TestFetchLotSize_ParamsErrorKeepsAssetLotAndRetries(t *testing.T) {
	m := newTradeLotMocks(5)
	m.paramsErr = status.Error(codes.Unavailable, "params unavailable")
	client := newTradeLotClient(m)

	client.getFullSymbol("SBER@TQBR", "acc1")

	if got := client.GetLotSize("SBER@TQBR"); got != 10 {
		t.Errorf("GetLotSize(SBER@TQBR) = %v, want 10 (asset lot survives params error)", got)
	}
	if got := m.paramCalls.Load(); got != 1 {
		t.Fatalf("GetAssetParams called %d times, want 1", got)
	}

	// The error must not be cached: the next resolve retries and now succeeds.
	m.paramsErr = nil
	client.getFullSymbol("SBER@TQBR", "acc1")
	if got := m.paramCalls.Load(); got != 2 {
		t.Errorf("GetAssetParams called %d times after retry, want 2", got)
	}
	if got := client.GetLotSize("SBER@TQBR"); got != 5 {
		t.Errorf("GetLotSize(SBER@TQBR) = %v, want 5 after successful retry", got)
	}
}

// TestPlaceOrder_UsesTradeLotSize verifies that an order for a cold symbol is
// sized by the trade lot (2 lots x 5 = 10 shares), not the asset lot.
func TestPlaceOrder_UsesTradeLotSize(t *testing.T) {
	m := newTradeLotMocks(5)

	var gotQuantity string
	mockOrders := &mockOrdersServiceClient{
		PlaceOrderFunc: func(ctx context.Context, in *orders.Order, opts ...grpc.CallOption) (*orders.OrderState, error) {
			gotQuantity = in.Quantity.GetValue()
			return &orders.OrderState{OrderId: "TL-001"}, nil
		},
	}

	client := newTradeLotClient(m)
	client.ordersClient = mockOrders

	if _, err := client.PlaceOrder("acc1", "SBER", "Buy", 2, nil); err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if gotQuantity != "10" {
		t.Errorf("PlaceOrder quantity = %q, want \"10\" (2 lots * trade lot 5)", gotQuantity)
	}
}

// TestPlaceSLTPOrder_UsesTradeLotSize verifies the same sizing for linked SL/TP
// orders.
func TestPlaceSLTPOrder_UsesTradeLotSize(t *testing.T) {
	m := newTradeLotMocks(5)

	var gotSL, gotTP string
	mockOrders := &mockOrdersServiceClient{
		PlaceSLTPOrderFunc: func(ctx context.Context, in *orders.SLTPOrder, opts ...grpc.CallOption) (*orders.OrderState, error) {
			gotSL = in.QuantitySl.GetValue()
			gotTP = in.QuantityTp.GetValue()
			return &orders.OrderState{OrderId: "TL-002"}, nil
		},
	}

	client := newTradeLotClient(m)
	client.ordersClient = mockOrders

	if _, err := client.PlaceSLTPOrder("acc1", "SBER", "Sell", 2, 100, 3, 200); err != nil {
		t.Fatalf("PlaceSLTPOrder error: %v", err)
	}
	if gotSL != "10" {
		t.Errorf("SL quantity = %q, want \"10\" (2 lots * trade lot 5)", gotSL)
	}
	if gotTP != "15" {
		t.Errorf("TP quantity = %q, want \"15\" (3 lots * trade lot 5)", gotTP)
	}
}
