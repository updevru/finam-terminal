package api

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
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
		t.Error("Expected error for bad payload, got nil")
	}
}
