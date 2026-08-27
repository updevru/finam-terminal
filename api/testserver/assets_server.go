package testserver

import (
	"context"
	"sync/atomic"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockAssetsServer implements assets.AssetsServiceServer for testing.
type MockAssetsServer struct {
	assets.UnimplementedAssetsServiceServer

	// GetAssetError, if set, is returned by GetAsset.
	GetAssetError error

	// GetAssetCallCount and GetAssetParamsCallCount count the lot-resolution
	// calls, so a test can prove a quote request does not drag them along.
	GetAssetCallCount       atomic.Int64
	GetAssetParamsCallCount atomic.Int64

	// GetAssetParamsError, if set, is returned by GetAssetParams.
	GetAssetParamsError error

	// ScheduleError, if set, is returned by Schedule.
	ScheduleError error

	// GetConstituentsError, if set, is returned by GetConstituents.
	GetConstituentsError error

	// GetConstituentsCallCount counts GetConstituents calls, so a test can
	// prove the client's composition cache prevents repeated RPCs. One full
	// fetch of DefaultConstituents costs two calls (two pages).
	GetConstituentsCallCount atomic.Int64

	// EmptyConstituents makes GetConstituents answer with no components at all,
	// which the client must treat as a failed load rather than a valid empty
	// index.
	EmptyConstituents bool

	// EndlessConstituents makes every page advertise another page after it,
	// simulating a server whose cursor never terminates, so the client's page
	// guard can be observed.
	EndlessConstituents bool
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
	m.GetAssetCallCount.Add(1)
	if m.GetAssetError != nil {
		return nil, m.GetAssetError
	}
	return DefaultAssetInfo(req.Symbol), nil
}

// GetAssetParams returns trading parameters for an instrument.
func (m *MockAssetsServer) GetAssetParams(_ context.Context, req *assets.GetAssetParamsRequest) (*assets.GetAssetParamsResponse, error) {
	m.GetAssetParamsCallCount.Add(1)
	if m.GetAssetParamsError != nil {
		return nil, m.GetAssetParamsError
	}

	resp := DefaultAssetParams(req.Symbol)
	if resp == nil {
		return nil, status.Errorf(codes.NotFound, "params not found for %s", req.Symbol)
	}
	return resp, nil
}

// GetConstituents returns one page of the index composition, driven by the
// request cursor, so the client's pagination loop is exercised end to end.
func (m *MockAssetsServer) GetConstituents(_ context.Context, req *assets.GetConstituentsRequest) (*assets.GetConstituentsResponse, error) {
	m.GetConstituentsCallCount.Add(1)

	if m.GetConstituentsError != nil {
		return nil, m.GetConstituentsError
	}
	if m.EmptyConstituents {
		return &assets.GetConstituentsResponse{}, nil
	}

	resp := DefaultConstituents(req.Cursor)
	if m.EndlessConstituents {
		resp.NextCursor = req.Cursor + 1
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
