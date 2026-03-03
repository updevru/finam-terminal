package api

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

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
