package ui

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"finam-terminal/models"
)

// waitFor polls cond until true or the timeout elapses.
func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestLoadProfileAsync_EquityFetchesCalendars(t *testing.T) {
	var divCalls, splitCalls int32
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(string) float64 { return 1 }
	mock.GetAssetInfoFunc = func(_, _ string) (*models.AssetDetails, error) {
		return &models.AssetDetails{Name: "Сбербанк", Type: "Stock"}, nil // equity: no markers
	}
	mock.GetDividendsFunc = func(string) ([]models.Dividend, error) {
		atomic.AddInt32(&divCalls, 1)
		return []models.Dividend{{Date: "2026-03-15", Amount: "15.5", Currency: "RUB", IsFuture: true}}, nil
	}
	mock.GetSplitsFunc = func(string) ([]models.Split, error) {
		atomic.AddInt32(&splitCalls, 1)
		return []models.Split{{Date: "2026-08-01", OldRatio: "2", NewRatio: "1", IsFuture: true}}, nil
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.profileOpen = true
	app.profileSymbol = "SBER@TQBR"
	app.loadProfileAsync("acc1", "SBER@TQBR", 2)

	if !waitFor(func() bool { return atomic.LoadInt32(&divCalls) == 1 && atomic.LoadInt32(&splitCalls) == 1 }) {
		t.Fatalf("expected dividends and splits fetched once for equity, got div=%d split=%d",
			atomic.LoadInt32(&divCalls), atomic.LoadInt32(&splitCalls))
	}
}

func TestLoadProfileAsync_FuturesSkipsCalendars(t *testing.T) {
	var divCalls, splitCalls int32
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(string) float64 { return 1 }
	mock.GetAssetInfoFunc = func(_, _ string) (*models.AssetDetails, error) {
		return &models.AssetDetails{Name: "Si-3.26", Type: "Futures", ContractSize: "1000"}, nil
	}
	mock.GetDividendsFunc = func(string) ([]models.Dividend, error) {
		atomic.AddInt32(&divCalls, 1)
		return nil, nil
	}
	mock.GetSplitsFunc = func(string) ([]models.Split, error) {
		atomic.AddInt32(&splitCalls, 1)
		return nil, nil
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.profileOpen = true
	app.profileSymbol = "SiH6@RTSX"
	app.loadProfileAsync("acc1", "SiH6@RTSX", 2)

	// Give the goroutine time; assert calendars were never fetched for a future.
	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt32(&divCalls) != 0 || atomic.LoadInt32(&splitCalls) != 0 {
		t.Errorf("futures must not fetch calendars, got div=%d split=%d",
			atomic.LoadInt32(&divCalls), atomic.LoadInt32(&splitCalls))
	}
}

func TestLoadProfileAsync_DividendFailureNonFatal(t *testing.T) {
	var splitCalls int32
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(string) float64 { return 1 }
	mock.GetAssetInfoFunc = func(_, _ string) (*models.AssetDetails, error) {
		return &models.AssetDetails{Name: "Сбербанк", Type: "Stock"}, nil
	}
	mock.GetDividendsFunc = func(string) ([]models.Dividend, error) {
		return nil, errors.New("boom")
	}
	mock.GetSplitsFunc = func(string) ([]models.Split, error) {
		atomic.AddInt32(&splitCalls, 1)
		return []models.Split{{Date: "2026-08-01", OldRatio: "2", NewRatio: "1", IsFuture: true}}, nil
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.profileOpen = true
	app.profileSymbol = "SBER@TQBR"

	// A failing dividends fetch must not prevent the splits fetch or panic.
	app.loadProfileAsync("acc1", "SBER@TQBR", 2)
	if !waitFor(func() bool { return atomic.LoadInt32(&splitCalls) == 1 }) {
		t.Error("splits should still be fetched when dividends fail")
	}
}
