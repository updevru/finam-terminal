//go:build integration

package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/api/testserver"
	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
)

// setupTestServer creates a TestServer + Client pair for integration tests.
func setupTestServer(t *testing.T) (*Client, *testserver.TestServer) {
	t.Helper()

	ts := testserver.NewTestServer()
	ts.Start()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}

	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client from conn: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		ts.Stop()
	})

	return client, ts
}

// --- Task 3.1: Client lifecycle tests ---

func TestIntegration_ClientLifecycle(t *testing.T) {
	client, _ := setupTestServer(t)

	// Verify auth happened (token should be set)
	client.tokenMutex.RLock()
	token := client.token
	expiry := client.tokenExpiry
	client.tokenMutex.RUnlock()

	if token == "" {
		t.Error("expected token to be set after init")
	}
	if expiry.IsZero() {
		t.Error("expected token expiry to be set")
	}

	// Verify cache was populated
	client.assetMutex.RLock()
	cacheLen := len(client.securityCache)
	micCacheLen := len(client.assetMicCache)
	nameCacheLen := len(client.instrumentNameCache)
	client.assetMutex.RUnlock()

	if cacheLen == 0 {
		t.Error("expected security cache to be populated")
	}
	if micCacheLen == 0 {
		t.Error("expected MIC cache to be populated")
	}
	if nameCacheLen == 0 {
		t.Error("expected instrument name cache to be populated")
	}

	// Verify close works cleanly
	if err := client.Close(); err != nil {
		t.Errorf("close error: %v", err)
	}
}

func TestIntegration_Auth_InvalidToken(t *testing.T) {
	ts := testserver.NewTestServer()
	ts.Start()
	defer ts.Stop()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	_, err = newClientFromConn(conn, "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestIntegration_Auth_JWTParsing(t *testing.T) {
	client, _ := setupTestServer(t)

	client.tokenMutex.RLock()
	expiry := client.tokenExpiry
	client.tokenMutex.RUnlock()

	if expiry.IsZero() {
		t.Fatal("expected non-zero expiry from JWT parsing")
	}

	// The mock JWT has 1h expiry, so expiry should be in the future
	if expiry.Before(time.Now()) {
		t.Error("expected expiry to be in the future")
	}
}

// --- Task 3.2: Account and position tests ---

func TestIntegration_GetAccounts(t *testing.T) {
	client, _ := setupTestServer(t)

	accounts, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("GetAccounts error: %v", err)
	}

	if len(accounts) != 2 { // Mock returns ACC001, ACC002
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	if accounts[0].ID != "ACC001" {
		t.Errorf("expected first account ACC001, got %s", accounts[0].ID)
	}
	if accounts[1].ID != "ACC002" {
		t.Errorf("expected second account ACC002, got %s", accounts[1].ID)
	}
}

func TestIntegration_GetAccountDetails(t *testing.T) {
	client, _ := setupTestServer(t)

	account, positions, err := client.GetAccountDetails("ACC001")
	if err != nil {
		t.Fatalf("GetAccountDetails error: %v", err)
	}

	if account.ID != "ACC001" {
		t.Errorf("expected account ID ACC001, got %s", account.ID)
	}
	if account.Equity == "" || account.Equity == "N/A" {
		t.Error("expected equity to be populated")
	}

	// Mock has 3 positions but one is zero-qty, should be filtered
	if len(positions) != 2 {
		t.Fatalf("expected 2 non-zero positions, got %d", len(positions))
	}

	// Check that MIC is resolved
	for _, pos := range positions {
		if pos.MIC == "" {
			t.Errorf("expected MIC to be set for %s", pos.Symbol)
		}
		if pos.Name == "" {
			t.Errorf("expected Name to be set for %s", pos.Symbol)
		}
	}
}

// --- Task 3.3: Market data tests ---

func TestIntegration_GetQuotes(t *testing.T) {
	client, _ := setupTestServer(t)

	quotes, err := client.GetQuotes("ACC001", []string{"SBER", "GAZP"})
	if err != nil {
		t.Fatalf("GetQuotes error: %v", err)
	}

	if len(quotes) < 2 {
		t.Fatalf("expected at least 2 quotes, got %d", len(quotes))
	}

	sberQuote := quotes["SBER@TQBR"]
	if sberQuote == nil {
		t.Fatal("expected SBER@TQBR quote")
	}
	if sberQuote.Last == "" || sberQuote.Last == "N/A" {
		t.Error("expected Last price for SBER")
	}
}

func TestIntegration_GetSnapshots(t *testing.T) {
	client, _ := setupTestServer(t)

	snapshots, err := client.GetSnapshots("ACC001", []string{"SBER", "GAZP"})
	if err != nil {
		t.Fatalf("GetSnapshots error: %v", err)
	}

	// Snapshots are keyed by ticker
	if _, ok := snapshots["SBER"]; !ok {
		t.Error("expected snapshot for SBER")
	}
	if _, ok := snapshots["GAZP"]; !ok {
		t.Error("expected snapshot for GAZP")
	}
}

func TestIntegration_GetBars(t *testing.T) {
	client, _ := setupTestServer(t)

	bars, err := client.GetBars("ACC001", "SBER", marketdata.TimeFrame_TIME_FRAME_D, time.Now().AddDate(0, 0, -7), time.Now())
	if err != nil {
		t.Fatalf("GetBars error: %v", err)
	}

	if len(bars) == 0 {
		t.Fatal("expected bars data")
	}

	bar := bars[0]
	if bar.Open == 0 {
		t.Error("expected non-zero Open")
	}
	if bar.High == 0 {
		t.Error("expected non-zero High")
	}
}

// --- Task 3.4: Search and asset info tests ---

func TestIntegration_SearchSecurities(t *testing.T) {
	client, _ := setupTestServer(t)

	// Search by ticker
	results, err := client.SearchSecurities("SBER")
	if err != nil {
		t.Fatalf("SearchSecurities error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for SBER")
	}

	// Search by name (partial, case-insensitive)
	results, err = client.SearchSecurities("газпром")
	if err != nil {
		t.Fatalf("SearchSecurities error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for газпром")
	}
}

func TestIntegration_GetAssetInfo(t *testing.T) {
	client, _ := setupTestServer(t)

	info, err := client.GetAssetInfo("ACC001", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetInfo error: %v", err)
	}

	if info == nil {
		t.Fatal("expected non-nil asset info")
	}
	if info.Name == "" {
		t.Error("expected asset name")
	}
}

func TestIntegration_GetAssetParams(t *testing.T) {
	client, _ := setupTestServer(t)

	params, err := client.GetAssetParams("ACC001", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetParams error: %v", err)
	}

	if params == nil {
		t.Fatal("expected non-nil asset params")
	}
}

func TestIntegration_GetSchedule(t *testing.T) {
	client, _ := setupTestServer(t)

	sessions, err := client.GetSchedule("SBER@TQBR")
	if err != nil {
		t.Fatalf("GetSchedule error: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("expected at least one trading session")
	}
}

// --- Task 3.5: Trade history and order management tests ---

func TestIntegration_GetTradeHistory(t *testing.T) {
	client, _ := setupTestServer(t)

	trades, err := client.GetTradeHistory("ACC001")
	if err != nil {
		t.Fatalf("GetTradeHistory error: %v", err)
	}

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	// Check side mapping
	if trades[0].Side != "Buy" {
		t.Errorf("expected Buy, got %s", trades[0].Side)
	}
	if trades[1].Side != "Sell" {
		t.Errorf("expected Sell, got %s", trades[1].Side)
	}
}

func TestIntegration_GetActiveOrders(t *testing.T) {
	client, _ := setupTestServer(t)

	activeOrders, err := client.GetActiveOrders("ACC001")
	if err != nil {
		t.Fatalf("GetActiveOrders error: %v", err)
	}

	if len(activeOrders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(activeOrders))
	}

	// Check status mapping (NEW -> Active)
	if activeOrders[0].Status != "Active" {
		t.Errorf("expected Active status, got %s", activeOrders[0].Status)
	}
}

func TestIntegration_PlaceOrder_Market(t *testing.T) {
	client, ts := setupTestServer(t)

	orderID, err := client.PlaceOrder("ACC001", "SBER", "buy", 5, nil)
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}

	if orderID == "" {
		t.Error("expected non-empty order ID")
	}

	// Check recorded request
	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()
	if len(ts.Orders.RecordedOrders) == 0 {
		t.Fatal("expected recorded order")
	}
	req := ts.Orders.RecordedOrders[len(ts.Orders.RecordedOrders)-1]
	if req.Symbol != "SBER@TQBR" {
		t.Errorf("expected SBER@TQBR, got %s", req.Symbol)
	}
}

func TestIntegration_PlaceOrder_Limit(t *testing.T) {
	client, ts := setupTestServer(t)

	orderID, err := client.PlaceOrder("ACC001", "SBER", "buy", 3, &models.OrderParams{
		OrderType:  models.OrderTypeLimit,
		LimitPrice: 280.50,
	})
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if orderID == "" {
		t.Error("expected non-empty order ID")
	}

	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()
	req := ts.Orders.RecordedOrders[len(ts.Orders.RecordedOrders)-1]
	if req.LimitPrice == nil || req.LimitPrice.Value == "" {
		t.Error("expected limit price in request")
	}
}

func TestIntegration_PlaceOrder_Stop(t *testing.T) {
	client, ts := setupTestServer(t)

	_, err := client.PlaceOrder("ACC001", "SBER", "sell", 2, &models.OrderParams{
		OrderType: models.OrderTypeStop,
		StopPrice: 275.00,
	})
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}

	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()
	req := ts.Orders.RecordedOrders[len(ts.Orders.RecordedOrders)-1]
	if req.StopPrice == nil || req.StopPrice.Value == "" {
		t.Error("expected stop price in request")
	}
	// Sell stop should be LAST_DOWN
	if req.StopCondition.String() != "STOP_CONDITION_LAST_DOWN" {
		t.Errorf("expected STOP_CONDITION_LAST_DOWN, got %s", req.StopCondition.String())
	}
}

func TestIntegration_PlaceSLTPOrder(t *testing.T) {
	client, ts := setupTestServer(t)

	orderID, err := client.PlaceSLTPOrder("ACC001", "SBER", "sell", 5, 270.0, 5, 300.0)
	if err != nil {
		t.Fatalf("PlaceSLTPOrder error: %v", err)
	}
	if orderID == "" {
		t.Error("expected non-empty order ID")
	}

	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()
	if len(ts.Orders.RecordedSLTPOrders) == 0 {
		t.Fatal("expected recorded SLTP order")
	}
	req := ts.Orders.RecordedSLTPOrders[len(ts.Orders.RecordedSLTPOrders)-1]
	if req.SlPrice == nil || req.SlPrice.Value == "" {
		t.Error("expected SL price in request")
	}
	if req.TpPrice == nil || req.TpPrice.Value == "" {
		t.Error("expected TP price in request")
	}
}

func TestIntegration_CancelOrder(t *testing.T) {
	client, ts := setupTestServer(t)

	err := client.CancelOrder("ACC001", "ORD001")
	if err != nil {
		t.Fatalf("CancelOrder error: %v", err)
	}

	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()
	if len(ts.Orders.RecordedCancellations) == 0 {
		t.Fatal("expected recorded cancellation")
	}
	if ts.Orders.RecordedCancellations[0].OrderId != "ORD001" {
		t.Error("expected ORD001 in cancellation")
	}
}

func TestIntegration_ClosePosition(t *testing.T) {
	client, _ := setupTestServer(t)

	// Positive quantity means long position -> close by selling
	orderID, err := client.ClosePosition("ACC001", "SBER", "100", 5)
	if err != nil {
		t.Fatalf("ClosePosition error: %v", err)
	}
	if orderID == "" {
		t.Error("expected order ID")
	}
}

