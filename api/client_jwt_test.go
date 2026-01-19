package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/grpc"
)

func TestGetExpiryFromToken(t *testing.T) {
	// 1. Create a dummy valid token
	// Header: {"alg":"HS256","typ":"JWT"} -> eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
	header := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	
	// Payload: {"exp": <future_timestamp>}
	expTime := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	expUnix := expTime.Unix()
	payloadJson := fmt.Sprintf(`{"exp":%d}`, expUnix)
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJson))
	
	signature := "dummy_sig"
	token := fmt.Sprintf("%s.%s.%s", header, payload, signature)

	c := &Client{}
	
	// method to be implemented: getExpiryFromToken
	gotTime, err := c.getExpiryFromToken(token)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Compare timestamps (allow small delta due to float conversions if any, though Unix int is precise)
	if !gotTime.Equal(expTime) {
		t.Errorf("Expected expiry %v, got %v", expTime, gotTime)
	}

	// 2. Test invalid token (no parts)
	_, err = c.getExpiryFromToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// 3. Test malformed payload
	badPayloadToken := fmt.Sprintf("%s.%s.%s", header, "bad-base64", signature)
	_, err = c.getExpiryFromToken(badPayloadToken)
	if err == nil {
		// We expect an error here
		t.Error("Expected error for bad payload, got nil")
	}
}

func TestTokenRefreshLoop(t *testing.T) {
	// Mock auth client to track calls
	authCalls := 0
	mockAuth := &mockAuthServiceClient{
		AuthFunc: func(ctx context.Context, in *auth.AuthRequest, opts ...grpc.CallOption) (*auth.AuthResponse, error) {
			authCalls++
			// Create a token that expires very soon (e.g., in 2 seconds)
			expTime := time.Now().Add(2 * time.Second).Unix()
			payloadJson := fmt.Sprintf(`{"exp":%d}`, expTime)
			payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJson))
			token := fmt.Sprintf("header.%s.sig", payload)
			
			return &auth.AuthResponse{Token: token}, nil
		},
	}

	client := &Client{
		authClient: mockAuth,
		apiToken:   "test-secret",
	}

	ctx, cancel := context.WithCancel(context.Background())
	client.refreshCancel = cancel

	// Start refresh loop in background
	// We'll modify startTokenRefresh to use a shorter lead time for testing if possible, 
	// or just wait. But for now, let's just test that it calls authenticate.
	
	// Implementation will happen in client.go
	go client.startTokenRefresh(ctx)

	// Wait for the loop to trigger at least one refresh
	// Since it refreshes 2 minutes before expiry, and our token expires in 2 seconds,
	// it should trigger immediately or very soon.
	
	time.Sleep(1100 * time.Millisecond)

	if authCalls == 0 {
		t.Error("Expected at least one call to authenticate in the refresh loop")
	}

	client.Close()
}
