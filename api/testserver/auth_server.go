package testserver

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockAuthServer implements auth.AuthServiceServer for testing.
type MockAuthServer struct {
	auth.UnimplementedAuthServiceServer

	// ValidTokens maps API tokens to whether they are valid.
	ValidTokens map[string]bool

	// AccountIDs returned by TokenDetails.
	AccountIDs []string

	// TokenExpiry controls the JWT expiry time. Defaults to 1 hour from now.
	TokenExpiry time.Duration

	// AuthCallCount tracks the number of Auth calls (for refresh tests).
	AuthCallCount atomic.Int64

	// AuthCalled is sent to (non-blocking) on every Auth call, for synchronization in tests.
	AuthCalled chan struct{}

	// AuthOverride, if set, is called instead of default Auth behavior.
	// Allows dynamic per-call error injection (e.g., fail first then succeed).
	AuthOverride func(ctx context.Context, req *auth.AuthRequest) (*auth.AuthResponse, error)

	// AuthError, if set, is returned by Auth instead of the normal response.
	// Ignored when AuthOverride is set.
	AuthError error
}

// NewMockAuthServer creates a MockAuthServer with sensible defaults.
func NewMockAuthServer() *MockAuthServer {
	return &MockAuthServer{
		ValidTokens: map[string]bool{"test-api-token": true},
		AccountIDs:  []string{"ACC001", "ACC002"},
		TokenExpiry: 1 * time.Hour,
		AuthCalled:  make(chan struct{}, 100),
	}
}

// Auth validates the secret and returns a JWT.
func (m *MockAuthServer) Auth(ctx context.Context, req *auth.AuthRequest) (*auth.AuthResponse, error) {
	m.AuthCallCount.Add(1)

	// Non-blocking notification
	select {
	case m.AuthCalled <- struct{}{}:
	default:
	}

	if m.AuthOverride != nil {
		return m.AuthOverride(ctx, req)
	}

	if m.AuthError != nil {
		return nil, m.AuthError
	}

	if !m.ValidTokens[req.Secret] {
		return nil, status.Errorf(codes.Unauthenticated, "invalid API token")
	}

	jwt := MakeJWT(time.Now().Add(m.TokenExpiry))
	return &auth.AuthResponse{Token: jwt}, nil
}

// TokenDetails returns the configured account IDs.
func (m *MockAuthServer) TokenDetails(_ context.Context, _ *auth.TokenDetailsRequest) (*auth.TokenDetailsResponse, error) {
	return &auth.TokenDetailsResponse{
		AccountIds: m.AccountIDs,
	}, nil
}
