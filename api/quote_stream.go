package api

import (
	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/protobuf/proto"
)

// quoteToModel converts an SDK quote into the model the UI renders. Absent
// fields become "N/A" via formatDecimal.
func quoteToModel(symbol string, q *marketdata.Quote) *models.Quote {
	return &models.Quote{
		Symbol:       symbol,
		Bid:          formatDecimal(q.Bid),
		BidSize:      formatDecimal(q.BidSize),
		Ask:          formatDecimal(q.Ask),
		AskSize:      formatDecimal(q.AskSize),
		Last:         formatDecimal(q.Last),
		LastSize:     formatDecimal(q.LastSize),
		Volume:       formatDecimal(q.Volume),
		Open:         formatDecimal(q.Open),
		High:         formatDecimal(q.High),
		Low:          formatDecimal(q.Low),
		Close:        formatDecimal(q.Close),
		OpenInterest: formatDecimal(q.OpenInterest),
		Timestamp:    q.Timestamp.AsTime().Local(),
	}
}

// mergeQuote folds a streamed quote into the previously known state.
//
// A snapshot (Trade API 2.19.0 is_data_snapshot) is authoritative: it replaces
// the state wholesale, so a field it omits is genuinely absent. An incremental
// update carries only what changed, so every non-nil field overwrites the
// previous one and everything else is kept. The first message is treated as a
// full state even when the flag is not set — there is nothing to merge into.
//
// prev is never mutated: the merge works on a copy, so a consumer holding the
// previous quote keeps seeing consistent data.
func mergeQuote(prev, next *marketdata.Quote) *marketdata.Quote {
	if prev == nil || next == nil || next.IsDataSnapshot {
		return next
	}

	// proto.Clone rather than a struct copy: protobuf messages carry internal
	// state that must not be copied by value.
	merged := proto.Clone(prev).(*marketdata.Quote)
	merged.Symbol = next.Symbol
	merged.IsDataSnapshot = next.IsDataSnapshot

	if next.Timestamp != nil {
		merged.Timestamp = next.Timestamp
	}
	if next.Ask != nil {
		merged.Ask = next.Ask
	}
	if next.AskSize != nil {
		merged.AskSize = next.AskSize
	}
	if next.Bid != nil {
		merged.Bid = next.Bid
	}
	if next.BidSize != nil {
		merged.BidSize = next.BidSize
	}
	if next.Last != nil {
		merged.Last = next.Last
	}
	if next.LastSize != nil {
		merged.LastSize = next.LastSize
	}
	if next.Volume != nil {
		merged.Volume = next.Volume
	}
	if next.Turnover != nil {
		merged.Turnover = next.Turnover
	}
	if next.Open != nil {
		merged.Open = next.Open
	}
	if next.High != nil {
		merged.High = next.High
	}
	if next.Low != nil {
		merged.Low = next.Low
	}
	if next.Close != nil {
		merged.Close = next.Close
	}
	if next.Change != nil {
		merged.Change = next.Change
	}
	if next.OpenInterest != nil {
		merged.OpenInterest = next.OpenInterest
	}

	return merged
}
