package api

import (
	"context"
	"testing"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"google.golang.org/grpc"
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
