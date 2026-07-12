package testserver

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// JwtRenewalItem is one item fed into MockAuthServer.JwtRenewalQueue: either a
// token to send over the SubscribeJwtRenewal stream, or an error that ends it
// (simulating a dropped connection).
type JwtRenewalItem struct {
	Token string
	Err   error
}

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

	// LastAuthSourceAppID stores the source_app_id observed on the most recent Auth call.
	LastAuthSourceAppID atomic.Value

	// JwtRenewalCallCount tracks the number of SubscribeJwtRenewal calls (each
	// call is one stream open, i.e. an initial subscribe or a reconnect).
	JwtRenewalCallCount atomic.Int64

	// JwtRenewalCalled is sent to (non-blocking) on every SubscribeJwtRenewal call.
	JwtRenewalCalled chan struct{}

	// JwtRenewalQueue feeds the active SubscribeJwtRenewal stream: each item is
	// either sent as a new token or, if Err is set, ends the stream with that
	// error (simulating a dropped connection). The stream also ends cleanly
	// (nil) when its context is cancelled.
	JwtRenewalQueue chan JwtRenewalItem

	// LastJwtRenewalSourceAppID stores the source_app_id observed on the most
	// recent SubscribeJwtRenewal call.
	LastJwtRenewalSourceAppID atomic.Value

	// JwtRenewalOverride, if set, is called instead of default stream behavior.
	JwtRenewalOverride func(req *auth.SubscribeJwtRenewalRequest, stream auth.AuthService_SubscribeJwtRenewalServer) error
}

// NewMockAuthServer creates a MockAuthServer with sensible defaults.
func NewMockAuthServer() *MockAuthServer {
	return &MockAuthServer{
		ValidTokens:      map[string]bool{"test-api-token": true},
		AccountIDs:       []string{"ACC001", "ACC002"},
		TokenExpiry:      1 * time.Hour,
		AuthCalled:       make(chan struct{}, 100),
		JwtRenewalCalled: make(chan struct{}, 100),
		JwtRenewalQueue:  make(chan JwtRenewalItem, 100),
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

	m.LastAuthSourceAppID.Store(req.SourceAppId)

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

// SubscribeJwtRenewal streams new JWTs from JwtRenewalQueue until the item's
// Err ends the stream (simulating a dropped connection) or the stream's
// context is cancelled (clean stop).
func (m *MockAuthServer) SubscribeJwtRenewal(req *auth.SubscribeJwtRenewalRequest, stream auth.AuthService_SubscribeJwtRenewalServer) error {
	m.JwtRenewalCallCount.Add(1)

	// Non-blocking notification
	select {
	case m.JwtRenewalCalled <- struct{}{}:
	default:
	}

	if m.JwtRenewalOverride != nil {
		return m.JwtRenewalOverride(req, stream)
	}

	if !m.ValidTokens[req.Secret] {
		return status.Errorf(codes.Unauthenticated, "invalid API token")
	}

	m.LastJwtRenewalSourceAppID.Store(req.SourceAppId)

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case item, ok := <-m.JwtRenewalQueue:
			if !ok {
				return nil
			}
			if item.Err != nil {
				return item.Err
			}
			if err := stream.Send(&auth.SubscribeJwtRenewalResponse{Token: item.Token}); err != nil {
				return err
			}
		}
	}
}

// TokenDetails returns the configured account IDs.
func (m *MockAuthServer) TokenDetails(_ context.Context, _ *auth.TokenDetailsRequest) (*auth.TokenDetailsResponse, error) {
	return &auth.TokenDetailsResponse{
		AccountIds: m.AccountIDs,
	}, nil
}
