package testserver

import (
	"context"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockAssetsServer implements assets.AssetsServiceServer for testing.
type MockAssetsServer struct {
	assets.UnimplementedAssetsServiceServer

	// GetAssetError, if set, is returned by GetAsset.
	GetAssetError error

	// GetAssetParamsError, if set, is returned by GetAssetParams.
	GetAssetParamsError error

	// ScheduleError, if set, is returned by Schedule.
	ScheduleError error
}

// NewMockAssetsServer creates a MockAssetsServer with defaults.
func NewMockAssetsServer() *MockAssetsServer {
	return &MockAssetsServer{}
}

// Assets returns the bulk asset list.
func (m *MockAssetsServer) Assets(_ context.Context, _ *assets.AssetsRequest) (*assets.AssetsResponse, error) {
	return &assets.AssetsResponse{
		Assets: DefaultAssets(),
	}, nil
}

// GetAsset returns details for a specific instrument.
func (m *MockAssetsServer) GetAsset(_ context.Context, req *assets.GetAssetRequest) (*assets.GetAssetResponse, error) {
	if m.GetAssetError != nil {
		return nil, m.GetAssetError
	}
	return DefaultAssetInfo(req.Symbol), nil
}

// GetAssetParams returns trading parameters for an instrument.
func (m *MockAssetsServer) GetAssetParams(_ context.Context, req *assets.GetAssetParamsRequest) (*assets.GetAssetParamsResponse, error) {
	if m.GetAssetParamsError != nil {
		return nil, m.GetAssetParamsError
	}

	resp := DefaultAssetParams(req.Symbol)
	if resp == nil {
		return nil, status.Errorf(codes.NotFound, "params not found for %s", req.Symbol)
	}
	return resp, nil
}

// Schedule returns trading sessions for an instrument.
func (m *MockAssetsServer) Schedule(_ context.Context, _ *assets.ScheduleRequest) (*assets.ScheduleResponse, error) {
	if m.ScheduleError != nil {
		return nil, m.ScheduleError
	}
	return DefaultSchedule(), nil
}
