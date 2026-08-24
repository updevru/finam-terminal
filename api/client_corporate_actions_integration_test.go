//go:build integration

package api

import (
	"testing"
)

func TestIntegration_GetDividends(t *testing.T) {
	client, _ := setupTestServer(t)

	divs, err := client.GetDividends("SBER@TQBR")
	if err != nil {
		t.Fatalf("GetDividends error: %v", err)
	}

	// past (1) + future (1), merged and sorted ascending by date
	if len(divs) != 2 {
		t.Fatalf("expected 2 dividends, got %d", len(divs))
	}

	// [0] past
	if divs[0].Date != "2026-03-15" {
		t.Errorf("div[0].Date: expected 2026-03-15, got %q", divs[0].Date)
	}
	if divs[0].Amount != "15.5" {
		t.Errorf("div[0].Amount: expected 15.5, got %q", divs[0].Amount)
	}
	if divs[0].Currency != "RUB" {
		t.Errorf("div[0].Currency: expected RUB, got %q", divs[0].Currency)
	}
	if divs[0].IsFuture {
		t.Errorf("div[0].IsFuture: expected false")
	}

	// [1] future
	if divs[1].Date != "2026-09-15" {
		t.Errorf("div[1].Date: expected 2026-09-15, got %q", divs[1].Date)
	}
	if !divs[1].IsFuture {
		t.Errorf("div[1].IsFuture: expected true")
	}
}

func TestIntegration_GetSplits(t *testing.T) {
	client, _ := setupTestServer(t)

	splits, err := client.GetSplits("SBER@TQBR")
	if err != nil {
		t.Fatalf("GetSplits error: %v", err)
	}

	if len(splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(splits))
	}

	// [0] past — has NewLot
	if splits[0].Date != "2025-06-01" {
		t.Errorf("split[0].Date: expected 2025-06-01, got %q", splits[0].Date)
	}
	if splits[0].OldRatio != "1" || splits[0].NewRatio != "10" {
		t.Errorf("split[0] ratio: expected 1/10, got %s/%s", splits[0].OldRatio, splits[0].NewRatio)
	}
	if splits[0].NewLot != "1" {
		t.Errorf("split[0].NewLot: expected 1, got %q", splits[0].NewLot)
	}
	if splits[0].ConvType == "" {
		t.Errorf("split[0].ConvType: expected non-empty")
	}
	if splits[0].IsFuture {
		t.Errorf("split[0].IsFuture: expected false")
	}

	// [1] future — nil NewLot wrapper must yield empty string, not panic
	if splits[1].Date != "2026-08-01" {
		t.Errorf("split[1].Date: expected 2026-08-01, got %q", splits[1].Date)
	}
	if splits[1].NewLot != "" {
		t.Errorf("split[1].NewLot: expected empty (nil wrapper), got %q", splits[1].NewLot)
	}
	if !splits[1].IsFuture {
		t.Errorf("split[1].IsFuture: expected true")
	}
}

func TestIntegration_GetBondEvents(t *testing.T) {
	client, _ := setupTestServer(t)

	events, err := client.GetBondEvents("SU26238@TQOB")
	if err != nil {
		t.Fatalf("GetBondEvents error: %v", err)
	}

	// past coupon (1) + future amortization + future offer, sorted ascending
	if len(events) != 3 {
		t.Fatalf("expected 3 bond events, got %d", len(events))
	}

	// [0] coupon
	c := events[0]
	if c.Kind != "Coupon" {
		t.Errorf("event[0].Kind: expected Coupon, got %q", c.Kind)
	}
	if c.Date != "2026-01-20" {
		t.Errorf("coupon Date: expected 2026-01-20, got %q", c.Date)
	}
	if c.Value != "34.9" || c.Currency != "RUB" {
		t.Errorf("coupon Value/Currency: expected 34.9/RUB, got %s/%s", c.Value, c.Currency)
	}
	if c.RecordDate != "2026-01-18" {
		t.Errorf("coupon RecordDate: expected 2026-01-18, got %q", c.RecordDate)
	}
	if c.StartDate != "2025-07-20" {
		t.Errorf("coupon StartDate: expected 2025-07-20, got %q", c.StartDate)
	}
	if c.FaceValue != "1000" {
		t.Errorf("coupon FaceValue: expected 1000, got %q", c.FaceValue)
	}
	if c.Percent != "6.98" {
		t.Errorf("coupon Percent: expected 6.98, got %q", c.Percent)
	}
	if c.IsFuture {
		t.Errorf("coupon IsFuture: expected false")
	}

	// [1] amortization
	a := events[1]
	if a.Kind != "Amortization" {
		t.Errorf("event[1].Kind: expected Amortization, got %q", a.Kind)
	}
	if a.NewFaceValue != "800" || a.InitialFaceValue != "1000" {
		t.Errorf("amort face values: expected 800/1000, got %s/%s", a.NewFaceValue, a.InitialFaceValue)
	}
	if a.Percent != "20" {
		t.Errorf("amort Percent: expected 20, got %q", a.Percent)
	}
	if !a.IsFuture {
		t.Errorf("amort IsFuture: expected true")
	}

	// [2] offer — nil Value and nil Currency wrappers must be empty, not panic
	o := events[2]
	if o.Kind != "Offer" {
		t.Errorf("event[2].Kind: expected Offer, got %q", o.Kind)
	}
	if o.Value != "" {
		t.Errorf("offer Value: expected empty (nil), got %q", o.Value)
	}
	if o.Currency != "" {
		t.Errorf("offer Currency: expected empty (nil), got %q", o.Currency)
	}
	if o.Type != "PUT" {
		t.Errorf("offer Type: expected PUT, got %q", o.Type)
	}
	if o.Price != "100" {
		t.Errorf("offer Price: expected 100, got %q", o.Price)
	}
	if o.Start != "2026-11-10" || o.End != "2026-11-14" {
		t.Errorf("offer window: expected 2026-11-10/2026-11-14, got %s/%s", o.Start, o.End)
	}
	if o.Agent != "Sberbank CIB" {
		t.Errorf("offer Agent: expected 'Sberbank CIB', got %q", o.Agent)
	}
	if !o.IsFuture {
		t.Errorf("offer IsFuture: expected true")
	}
}
