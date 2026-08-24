package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	tradeapiv1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/corporateactions"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/genproto/googleapis/type/interval"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// MakeJWT generates a minimal valid JWT with the given expiry time.
func MakeJWT(expiry time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{
		"exp": expiry.Unix(),
		"sub": "test-user",
	})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	sig := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
	return fmt.Sprintf("%s.%s.%s", header, payload, sig)
}

// DefaultAssets returns a set of realistic instruments for testing.
func DefaultAssets() []*assets.Asset {
	return []*assets.Asset{
		{Ticker: "SBER", Symbol: "SBER@TQBR", Name: "Сбер Банк", Mic: "TQBR"},
		{Ticker: "GAZP", Symbol: "GAZP@TQBR", Name: "Газпром", Mic: "TQBR"},
		{Ticker: "LKOH", Symbol: "LKOH@TQBR", Name: "ЛУКОЙЛ", Mic: "TQBR"},
		{Ticker: "YNDX", Symbol: "YNDX@TQBR", Name: "Яндекс", Mic: "TQBR"},
		{Ticker: "ROSN", Symbol: "ROSN@TQBR", Name: "Роснефть", Mic: "TQBR"},
	}
}

// DefaultAccountPositions returns positions for a test account.
func DefaultAccountPositions(accountID string) []*accounts.Position {
	return []*accounts.Position{
		{
			Symbol:       "SBER@TQBR",
			Quantity:     &decimal.Decimal{Value: "100"},
			AveragePrice: &decimal.Decimal{Value: "280.50"},
			CurrentPrice: &decimal.Decimal{Value: "285.00"},
		},
		{
			Symbol:       "GAZP@TQBR",
			Quantity:     &decimal.Decimal{Value: "50"},
			AveragePrice: &decimal.Decimal{Value: "155.00"},
			CurrentPrice: &decimal.Decimal{Value: "160.30"},
		},
		// Zero-quantity position (should be filtered by client)
		{
			Symbol:       "LKOH@TQBR",
			Quantity:     &decimal.Decimal{Value: "0"},
			AveragePrice: &decimal.Decimal{Value: "7000.00"},
			CurrentPrice: &decimal.Decimal{Value: "7100.00"},
		},
	}
}

// DefaultQuote returns a realistic quote for the given symbol.
func DefaultQuote(symbol string) *marketdata.Quote {
	quotes := map[string]*marketdata.Quote{
		"SBER@TQBR": {
			Symbol: "SBER@TQBR",
			Last:   &decimal.Decimal{Value: "285.00"},
			Bid:    &decimal.Decimal{Value: "284.90"},
			Ask:    &decimal.Decimal{Value: "285.10"},
		},
		"GAZP@TQBR": {
			Symbol: "GAZP@TQBR",
			Last:   &decimal.Decimal{Value: "160.30"},
			Bid:    &decimal.Decimal{Value: "160.20"},
			Ask:    &decimal.Decimal{Value: "160.40"},
		},
	}
	if q, ok := quotes[symbol]; ok {
		return q
	}
	return nil
}

// DefaultBars returns 5 candlesticks for testing.
func DefaultBars(symbol string) []*marketdata.Bar {
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	bars := make([]*marketdata.Bar, 5)
	for i := range bars {
		t := base.Add(time.Duration(i) * time.Hour)
		bars[i] = &marketdata.Bar{
			Timestamp: timestamppb.New(t),
			Open:      &decimal.Decimal{Value: fmt.Sprintf("%.2f", 280.0+float64(i))},
			High:      &decimal.Decimal{Value: fmt.Sprintf("%.2f", 282.0+float64(i))},
			Low:       &decimal.Decimal{Value: fmt.Sprintf("%.2f", 279.0+float64(i))},
			Close:     &decimal.Decimal{Value: fmt.Sprintf("%.2f", 281.0+float64(i))},
			Volume:    &decimal.Decimal{Value: fmt.Sprintf("%d", 1000+i*100)},
		}
	}
	return bars
}

// DefaultOrders returns a mix of order types for testing.
func DefaultOrders(accountID string) []*orders.OrderState {
	return []*orders.OrderState{
		{
			OrderId: "ORD001",
			Status:  orders.OrderStatus_ORDER_STATUS_NEW,
			// This stop order spawned ORD002 (both present in the active set → link is visible on both sides)
			TriggeredOrderId: "ORD002",
			Order: &orders.Order{
				AccountId:  accountID,
				Symbol:     "SBER@TQBR",
				Side:       tradeapiv1.Side_SIDE_BUY,
				Type:       orders.OrderType_ORDER_TYPE_LIMIT,
				Quantity:   &decimal.Decimal{Value: "10"},
				LimitPrice: &decimal.Decimal{Value: "280.00"},
			},
		},
		{
			OrderId: "ORD002",
			Status:  orders.OrderStatus_ORDER_STATUS_NEW,
			Order: &orders.Order{
				AccountId: accountID,
				Symbol:    "GAZP@TQBR",
				Side:      tradeapiv1.Side_SIDE_SELL,
				Type:      orders.OrderType_ORDER_TYPE_MARKET,
				Quantity:  &decimal.Decimal{Value: "5"},
			},
		},
	}
}

// DefaultTrades returns trade history entries for testing.
func DefaultTrades(accountID string) []*tradeapiv1.AccountTrade {
	t := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	return []*tradeapiv1.AccountTrade{
		{
			TradeId:   "TRD001",
			AccountId: accountID,
			Symbol:    "SBER@TQBR",
			Side:      tradeapiv1.Side_SIDE_BUY,
			Size:      &decimal.Decimal{Value: "10"},
			Price:     &decimal.Decimal{Value: "280.50"},
			Timestamp: timestamppb.New(t),
		},
		{
			TradeId:   "TRD002",
			AccountId: accountID,
			Symbol:    "GAZP@TQBR",
			Side:      tradeapiv1.Side_SIDE_SELL,
			Size:      &decimal.Decimal{Value: "5"},
			Price:     &decimal.Decimal{Value: "160.00"},
			Timestamp: timestamppb.New(t.Add(30 * time.Minute)),
		},
		// Bond trade carries accrued interest (2.16.0) and an explicit price currency
		{
			TradeId:         "TRD003",
			AccountId:       accountID,
			Symbol:          "SU26238@TQOB",
			Side:            tradeapiv1.Side_SIDE_BUY,
			Size:            &decimal.Decimal{Value: "3"},
			Price:           &decimal.Decimal{Value: "650.10"},
			AccruedInterest: &decimal.Decimal{Value: "12.34"},
			Currency:        "RUB",
			Timestamp:       timestamppb.New(t.Add(time.Hour)),
		},
	}
}

// DefaultAssetInfo returns a GetAssetResponse for the given symbol.
func DefaultAssetInfo(symbol string) *assets.GetAssetResponse {
	ticker := symbolTicker(symbol)
	mic := "TQBR"
	if i := len(ticker); i < len(symbol) {
		mic = symbol[i+1:] // extract MIC from ticker@MIC
	}
	return &assets.GetAssetResponse{
		Ticker:   ticker,
		Board:    mic,
		Mic:      mic,
		Name:     "Test Asset " + symbol,
		LotSize:  &decimal.Decimal{Value: "10"},
		Decimals: 2,
	}
}

// DefaultAssetParams returns trading parameters for the given symbol.
// TradeLotSize is deliberately 5 while DefaultAssetInfo reports LotSize 10, so
// tests can prove that the trade lot (GetAssetParams.trade_lot_size) wins over
// the asset lot (GetAsset.lot_size) end to end.
func DefaultAssetParams(symbol string) *assets.GetAssetParamsResponse {
	return &assets.GetAssetParamsResponse{
		Symbol:       symbol,
		Longable:     &assets.Longable{Value: assets.Longable_AVAILABLE},
		Shortable:    &assets.Shortable{Value: assets.Shortable_AVAILABLE},
		TradeLotSize: 5,
	}
}

// DefaultSchedule returns a trading schedule.
func DefaultSchedule() *assets.ScheduleResponse {
	today := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	return &assets.ScheduleResponse{
		Sessions: []*assets.ScheduleResponse_Sessions{
			{
				Type: "main",
				Interval: &interval.Interval{
					StartTime: timestamppb.New(today.Add(7 * time.Hour)),
					EndTime:   timestamppb.New(today.Add(15*time.Hour + 40*time.Minute)),
				},
			},
		},
	}
}

// DefaultDividends returns past and future dividend fixtures. The past set is
// returned DESC-friendly and the future set ASC-friendly so client-side merging
// and IsFuture flagging can be asserted.
func DefaultDividends() (past, future []*corporateactions.Dividend) {
	past = []*corporateactions.Dividend{
		{
			Date:     &date.Date{Year: 2026, Month: 3, Day: 15},
			Amount:   &decimal.Decimal{Value: "15.5"},
			Currency: "RUB",
		},
	}
	future = []*corporateactions.Dividend{
		{
			Date:     &date.Date{Year: 2026, Month: 9, Day: 15},
			Amount:   &decimal.Decimal{Value: "20.0"},
			Currency: "RUB",
		},
	}
	return past, future
}

// DefaultSplits returns past and future split fixtures. The future split has a
// nil NewLot wrapper to exercise nil-safe handling of *wrapperspb.Int32Value.
func DefaultSplits() (past, future []*corporateactions.SplitInfo) {
	past = []*corporateactions.SplitInfo{
		{
			ExecDate:         &date.Date{Year: 2025, Month: 6, Day: 1},
			OldRatio:         &decimal.Decimal{Value: "1"},
			NewRatio:         &decimal.Decimal{Value: "10"},
			NewLot:           wrapperspb.Int32(1),
			ConvertationType: corporateactions.ConvertationType_ORDINARY,
		},
	}
	future = []*corporateactions.SplitInfo{
		{
			ExecDate:         &date.Date{Year: 2026, Month: 8, Day: 1},
			OldRatio:         &decimal.Decimal{Value: "2"},
			NewRatio:         &decimal.Decimal{Value: "1"},
			NewLot:           nil, // exercise nil Int32 wrapper
			ConvertationType: corporateactions.ConvertationType_TENDER_OFFER,
		},
	}
	return past, future
}

// DefaultBondEvents returns past and future bond-event fixtures covering all
// three oneof branches (coupon, amortization, offer) and nil pointer wrappers
// (the offer has nil Value and nil Currency).
func DefaultBondEvents() (past, future []*corporateactions.BondEvent) {
	past = []*corporateactions.BondEvent{
		{
			Date:     &date.Date{Year: 2026, Month: 1, Day: 20},
			Type:     corporateactions.BondEventType_COUPON,
			Value:    &decimal.Decimal{Value: "34.9"},
			Currency: wrapperspb.String("RUB"),
			EventDetails: &corporateactions.BondEvent_CouponDetails{
				CouponDetails: &corporateactions.CouponEventDetails{
					RecordDate:   &date.Date{Year: 2026, Month: 1, Day: 18},
					StartDate:    &date.Date{Year: 2025, Month: 7, Day: 20},
					FaceValue:    &decimal.Decimal{Value: "1000"},
					ValuePercent: &decimal.Decimal{Value: "6.98"},
				},
			},
		},
	}
	future = []*corporateactions.BondEvent{
		{
			Date:     &date.Date{Year: 2026, Month: 10, Day: 20},
			Type:     corporateactions.BondEventType_AMORTIZATION,
			Value:    &decimal.Decimal{Value: "200"},
			Currency: wrapperspb.String("RUB"),
			EventDetails: &corporateactions.BondEvent_AmortizationDetails{
				AmortizationDetails: &corporateactions.AmortizationEventDetails{
					NewFaceValue:        &decimal.Decimal{Value: "800"},
					InitialFaceValue:    &decimal.Decimal{Value: "1000"},
					AmortizationPercent: &decimal.Decimal{Value: "20"},
				},
			},
		},
		{
			Date:     &date.Date{Year: 2026, Month: 11, Day: 15},
			Type:     corporateactions.BondEventType_OFFER,
			Value:    nil, // offer: only price matters
			Currency: nil, // exercise nil StringValue wrapper
			EventDetails: &corporateactions.BondEvent_OfferDetails{
				OfferDetails: &corporateactions.OfferEventDetails{
					OfferType: wrapperspb.String("PUT"),
					Price:     &decimal.Decimal{Value: "100"},
					StartDate: &date.Date{Year: 2026, Month: 11, Day: 10},
					EndDate:   &date.Date{Year: 2026, Month: 11, Day: 14},
					Agent:     wrapperspb.String("Sberbank CIB"),
				},
			},
		},
	}
	return past, future
}

func symbolTicker(symbol string) string {
	for i, c := range symbol {
		if c == '@' {
			return symbol[:i]
		}
	}
	return symbol
}
