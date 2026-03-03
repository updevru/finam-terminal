package api

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
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
