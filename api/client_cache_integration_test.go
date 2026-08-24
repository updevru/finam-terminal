//go:build integration

package api

import (
	"testing"
)

func TestIntegration_AssetCache_PopulatedOnInit(t *testing.T) {
	client, _ := setupTestServer(t)

	client.assetMutex.RLock()
	defer client.assetMutex.RUnlock()

	// 5 assets from DefaultAssets()
	if len(client.securityCache) != 5 {
		t.Errorf("expected 5 securities in cache, got %d", len(client.securityCache))
	}

	// MIC cache should map ticker -> symbol@mic
	if sym, ok := client.assetMicCache["SBER"]; !ok || sym != "SBER@TQBR" {
		t.Errorf("expected assetMicCache[SBER]=SBER@TQBR, got %q (exists=%v)", sym, ok)
	}

	// Instrument name cache should have entries by ticker and full symbol
	if name := client.instrumentNameCache["SBER"]; name != "Сбер Банк" {
		t.Errorf("expected name 'Сбер Банк' for SBER, got %q", name)
	}
	if name := client.instrumentNameCache["SBER@TQBR"]; name != "Сбер Банк" {
		t.Errorf("expected name 'Сбер Банк' for SBER@TQBR, got %q", name)
	}
}

func TestIntegration_AssetCache_LotSizeFetchOnDemand(t *testing.T) {
	client, _ := setupTestServer(t)

	// Lot size is NOT populated by loadAssetCache (Assets endpoint doesn't return it in the Asset proto).
	// It should be fetched on demand via GetAsset when getFullSymbol encounters a cache miss.
	client.assetMutex.RLock()
	lotBefore := client.assetLotCache["SBER"]
	client.assetMutex.RUnlock()

	if lotBefore != 0 {
		t.Fatalf("expected lot size to be 0 before demand fetch, got %v", lotBefore)
	}

	// Trigger a method that calls getFullSymbol -> fetchLotSize
	_, _ = client.GetQuotes("ACC001", []string{"SBER"})

	client.assetMutex.RLock()
	lotAfter := client.assetLotCache["SBER"]
	lotFull := client.assetLotCache["SBER@TQBR"]
	client.assetMutex.RUnlock()

	// After the fetch, lot size should be cached by both ticker and full symbol
	if lotAfter == 0 && lotFull == 0 {
		t.Error("expected lot size to be cached after demand fetch")
	}
}

func TestIntegration_GetLotSize_CacheLookup(t *testing.T) {
	client, _ := setupTestServer(t)

	// Trigger lot size fetch
	_, _ = client.GetQuotes("ACC001", []string{"SBER"})

	// Lookup by ticker. The fixtures serve an asset lot of 10 and a trade lot
	// of 5, and the trade lot is what the broker sizes orders by.
	lot := client.GetLotSize("SBER")
	if lot == 0 {
		// Try full symbol
		lot = client.GetLotSize("SBER@TQBR")
	}
	if lot != 5 {
		t.Errorf("expected trade lot size 5, got %v", lot)
	}

	// The asset lot tier is still populated underneath.
	client.assetMutex.RLock()
	assetLot := client.assetLotCache["SBER"]
	client.assetMutex.RUnlock()
	if assetLot != 10 {
		t.Errorf("expected asset lot size 10, got %v", assetLot)
	}
}

func TestIntegration_GetInstrumentName_CacheLookup(t *testing.T) {
	client, _ := setupTestServer(t)

	// By ticker
	name := client.GetInstrumentName("SBER")
	if name != "Сбер Банк" {
		t.Errorf("expected 'Сбер Банк' by ticker, got %q", name)
	}

	// By full symbol
	name = client.GetInstrumentName("GAZP@TQBR")
	if name != "Газпром" {
		t.Errorf("expected 'Газпром' by full symbol, got %q", name)
	}

	// Unknown
	name = client.GetInstrumentName("UNKNOWN")
	if name != "" {
		t.Errorf("expected empty for unknown, got %q", name)
	}
}

func TestIntegration_UpdateInstrumentCache(t *testing.T) {
	client, _ := setupTestServer(t)

	client.UpdateInstrumentCache("TEST", "TEST@XXYY", "Test Instrument")

	if name := client.GetInstrumentName("TEST"); name != "Test Instrument" {
		t.Errorf("expected 'Test Instrument' by ticker, got %q", name)
	}
	if name := client.GetInstrumentName("TEST@XXYY"); name != "Test Instrument" {
		t.Errorf("expected 'Test Instrument' by full symbol, got %q", name)
	}
}

func TestIntegration_EnsureLotSize_WarmsColdSymbol(t *testing.T) {
	client, _ := setupTestServer(t)

	// Nothing has touched SBER yet: both tiers are cold.
	client.assetMutex.RLock()
	_, hasTradeLot := client.tradeLotCache["SBER"]
	client.assetMutex.RUnlock()
	if hasTradeLot {
		t.Fatal("expected the trade lot cache to be cold before EnsureLotSize")
	}

	if lot := client.EnsureLotSize("ACC001", "SBER"); lot != 5 {
		t.Errorf("EnsureLotSize(SBER) = %v, want 5 (trade lot)", lot)
	}

	client.assetMutex.RLock()
	tradeLot := client.tradeLotCache["SBER"]
	assetLot := client.assetLotCache["SBER"]
	client.assetMutex.RUnlock()

	if tradeLot != 5 {
		t.Errorf("tradeLotCache[SBER] = %v, want 5", tradeLot)
	}
	if assetLot != 10 {
		t.Errorf("assetLotCache[SBER] = %v, want 10", assetLot)
	}
}

func TestIntegration_PositionsWarmTradeLot(t *testing.T) {
	client, _ := setupTestServer(t)

	_, positions, err := client.GetAccountDetails("ACC001")
	if err != nil {
		t.Fatalf("GetAccountDetails error: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("expected positions for ACC001")
	}

	for _, pos := range positions {
		if pos.LotSize != 5 {
			t.Errorf("position %s: LotSize = %v, want 5 (trade lot)", pos.Symbol, pos.LotSize)
		}
	}
}

func TestIntegration_PlaceOrder_UsesTradeLotSize(t *testing.T) {
	client, ts := setupTestServer(t)

	if _, err := client.PlaceOrder("ACC001", "SBER", "Buy", 2, nil); err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}

	ts.Orders.Mu.Lock()
	defer ts.Orders.Mu.Unlock()

	if len(ts.Orders.RecordedOrders) != 1 {
		t.Fatalf("expected 1 recorded order, got %d", len(ts.Orders.RecordedOrders))
	}
	// 2 lots * trade lot 5 = 10 shares (the asset lot of 10 would give 20).
	if got := ts.Orders.RecordedOrders[0].Quantity.GetValue(); got != "10" {
		t.Errorf("recorded order quantity = %q, want \"10\"", got)
	}
}

func TestIntegration_GetAssetParams_WarmsTradeLotCache(t *testing.T) {
	client, _ := setupTestServer(t)

	params, err := client.GetAssetParams("ACC001", "SBER@TQBR")
	if err != nil {
		t.Fatalf("GetAssetParams error: %v", err)
	}
	if params.TradeLotSize != 5 {
		t.Errorf("params.TradeLotSize = %d, want 5", params.TradeLotSize)
	}

	client.assetMutex.RLock()
	byFull := client.tradeLotCache["SBER@TQBR"]
	byTicker := client.tradeLotCache["SBER"]
	client.assetMutex.RUnlock()

	if byFull != 5 || byTicker != 5 {
		t.Errorf("trade lot cache after profile load: full=%v ticker=%v, want 5 and 5", byFull, byTicker)
	}
}
