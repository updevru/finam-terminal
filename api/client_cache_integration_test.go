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

	// Lookup by ticker
	lot := client.GetLotSize("SBER")
	if lot == 0 {
		// Try full symbol
		lot = client.GetLotSize("SBER@TQBR")
	}
	if lot != 10 {
		t.Errorf("expected lot size 10, got %v", lot)
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
