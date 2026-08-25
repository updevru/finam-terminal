package api

import (
	"testing"
	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func dec(v string) *decimal.Decimal { return &decimal.Decimal{Value: v} }

// TestMergeQuote_SnapshotReplacesState verifies that a snapshot is authoritative:
// fields it omits must not survive from the previous state.
func TestMergeQuote_SnapshotReplacesState(t *testing.T) {
	prev := &marketdata.Quote{
		Symbol: "SBER@TQBR",
		Last:   dec("280"),
		Bid:    dec("279.90"),
		Ask:    dec("280.10"),
	}
	snapshot := &marketdata.Quote{
		Symbol:         "SBER@TQBR",
		Last:           dec("290"),
		IsDataSnapshot: true,
	}

	merged := mergeQuote(prev, snapshot)

	if merged.Last.GetValue() != "290" {
		t.Errorf("Last = %q, want 290", merged.Last.GetValue())
	}
	if merged.Bid != nil {
		t.Errorf("Bid = %q, want nil (snapshot omitted it)", merged.Bid.GetValue())
	}
	if merged.Ask != nil {
		t.Errorf("Ask = %q, want nil (snapshot omitted it)", merged.Ask.GetValue())
	}
}

// TestMergeQuote_IncrementKeepsPreviousFields verifies that an incremental
// update only overwrites the fields it carries.
func TestMergeQuote_IncrementKeepsPreviousFields(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 8, 13, 10, 30, 0, 0, time.UTC))
	prev := &marketdata.Quote{
		Symbol:    "SBER@TQBR",
		Last:      dec("280"),
		Bid:       dec("279.90"),
		Ask:       dec("280.10"),
		Volume:    dec("1000"),
		Timestamp: timestamppb.New(time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)),
	}
	increment := &marketdata.Quote{
		Symbol:    "SBER@TQBR",
		Last:      dec("291"),
		Timestamp: ts,
	}

	merged := mergeQuote(prev, increment)

	if merged.Last.GetValue() != "291" {
		t.Errorf("Last = %q, want 291", merged.Last.GetValue())
	}
	if merged.Bid.GetValue() != "279.90" {
		t.Errorf("Bid = %q, want 279.90 (kept from previous state)", merged.Bid.GetValue())
	}
	if merged.Ask.GetValue() != "280.10" {
		t.Errorf("Ask = %q, want 280.10 (kept)", merged.Ask.GetValue())
	}
	if merged.Volume.GetValue() != "1000" {
		t.Errorf("Volume = %q, want 1000 (kept)", merged.Volume.GetValue())
	}
	if !merged.Timestamp.AsTime().Equal(ts.AsTime()) {
		t.Errorf("Timestamp = %v, want %v (from the increment)", merged.Timestamp.AsTime(), ts.AsTime())
	}

	// The merge must not mutate the previous state in place.
	if prev.Last.GetValue() != "280" {
		t.Errorf("previous state mutated: Last = %q, want 280", prev.Last.GetValue())
	}
}

// TestMergeQuote_FirstMessageWithoutFlag verifies that the very first message is
// taken as the full state even when it does not set IsDataSnapshot.
func TestMergeQuote_FirstMessageWithoutFlag(t *testing.T) {
	next := &marketdata.Quote{Symbol: "SBER@TQBR", Last: dec("290")}

	merged := mergeQuote(nil, next)

	if merged != next {
		t.Error("merging into an empty state should return the incoming quote as-is")
	}
}

// TestQuoteToModel_FormatsAndFallsBack verifies the model mapping, including
// "N/A" for fields the stream did not send.
func TestQuoteToModel_FormatsAndFallsBack(t *testing.T) {
	ts := time.Date(2026, 8, 13, 10, 30, 0, 0, time.UTC)
	q := &marketdata.Quote{
		Symbol:    "SBER@TQBR",
		Last:      dec("290"),
		Bid:       dec("289.90"),
		Timestamp: timestamppb.New(ts),
	}

	m := quoteToModel("SBER@TQBR", q)

	if m.Symbol != "SBER@TQBR" {
		t.Errorf("Symbol = %q, want SBER@TQBR", m.Symbol)
	}
	if m.Last != "290" {
		t.Errorf("Last = %q, want 290", m.Last)
	}
	if m.Bid != "289.90" {
		t.Errorf("Bid = %q, want 289.90", m.Bid)
	}
	if m.Ask != "N/A" {
		t.Errorf("Ask = %q, want N/A for an absent field", m.Ask)
	}
	if m.OpenInterest != "N/A" {
		t.Errorf("OpenInterest = %q, want N/A for an absent field", m.OpenInterest)
	}
	if !m.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", m.Timestamp, ts)
	}
	if m.Timestamp.Location() != time.Local {
		t.Errorf("Timestamp location = %v, want local", m.Timestamp.Location())
	}
}
