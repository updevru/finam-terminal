package ui

import (
	"testing"
	"time"

	"finam-terminal/models"
)

// TestApplyOrderModalLot_UpdatesLabel verifies that a lot resolved in the
// background lands on the modal and is reflected in the quantity label.
func TestApplyOrderModalLot_UpdatesLabel(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.orderModal.SetInstrument("SBER")
	app.orderModal.SetLotSize(10)

	app.applyOrderModalLot("SBER", 5)

	if got := app.orderModal.GetLotSize(); got != 5 {
		t.Errorf("modal lot size = %v, want 5", got)
	}
	if label := app.orderModal.quantity.GetLabel(); label != "Lots (size - 5): " {
		t.Errorf("quantity label = %q, want %q", label, "Lots (size - 5): ")
	}
}

// TestApplyOrderModalLot_IgnoresOtherInstrument verifies that a late answer for
// one instrument never rewrites a modal the user has since reopened on another.
func TestApplyOrderModalLot_IgnoresOtherInstrument(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.orderModal.SetInstrument("GAZP")
	app.orderModal.SetLotSize(1)

	app.applyOrderModalLot("SBER", 5)

	if got := app.orderModal.GetLotSize(); got != 1 {
		t.Errorf("modal lot size = %v, want 1 (untouched)", got)
	}
}

// TestApplyModifyModalLot_RecalculatesUntouchedQuantity verifies that the
// pre-filled lot count of the modify-order modal follows the corrected lot size.
func TestApplyModifyModalLot_RecalculatesUntouchedQuantity(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.orderModal.SetInstrument("SBER")
	app.orderModal.SetLotSize(10)
	app.orderModal.SetQuantity(10) // 100 shares / asset lot 10

	app.applyModifyModalLot("SBER", 5, 100, 10)

	if got := app.orderModal.GetLotSize(); got != 5 {
		t.Errorf("modal lot size = %v, want 5", got)
	}
	if got := app.orderModal.GetQuantity(); got != 20 {
		t.Errorf("quantity = %v, want 20 (100 shares / trade lot 5)", got)
	}
}

// TestApplyModifyModalLot_KeepsEditedQuantity verifies that a quantity the user
// already edited is left alone while the lot label still gets corrected.
func TestApplyModifyModalLot_KeepsEditedQuantity(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.orderModal.SetInstrument("SBER")
	app.orderModal.SetLotSize(10)
	app.orderModal.SetQuantity(3) // user typed over the pre-filled 10

	app.applyModifyModalLot("SBER", 5, 100, 10)

	if got := app.orderModal.GetLotSize(); got != 5 {
		t.Errorf("modal lot size = %v, want 5", got)
	}
	if got := app.orderModal.GetQuantity(); got != 3 {
		t.Errorf("quantity = %v, want 3 (user edit preserved)", got)
	}
}

// TestOpenOrderModalWithTicker_WarmsLotSize verifies that opening the modal from
// the search window kicks off a background lot resolution for that account.
func TestOpenOrderModalWithTicker_WarmsLotSize(t *testing.T) {
	called := make(chan [2]string, 1)
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 0 },
		EnsureLotSizeFunc: func(accountID, symbol string) float64 {
			called <- [2]string{accountID, symbol}
			return 5
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0

	app.OpenOrderModalWithTicker("SBER")

	select {
	case got := <-called:
		if got[0] != "acc1" || got[1] != "SBER" {
			t.Errorf("EnsureLotSize called with (%q, %q), want (\"acc1\", \"SBER\")", got[0], got[1])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("EnsureLotSize was not called for the opened modal")
	}
}

// TestOpenOrderModalWithTicker_ReadsLotAfterSnapshots verifies that the modal
// picks up a lot size the snapshot call warmed as a side effect.
func TestOpenOrderModalWithTicker_ReadsLotAfterSnapshots(t *testing.T) {
	lotAfterSnapshots := 0.0
	mock := &mockClient{
		GetSnapshotsFunc: func(accountID string, symbols []string) (map[string]models.Quote, error) {
			lotAfterSnapshots = 5 // the API call warmed the cache
			return map[string]models.Quote{"SBER": {Last: "280"}}, nil
		},
		GetLotSizeFunc: func(ticker string) float64 { return lotAfterSnapshots },
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0

	app.OpenOrderModalWithTicker("SBER")

	if got := app.orderModal.GetLotSize(); got != 5 {
		t.Errorf("modal lot size = %v, want 5 (lot read after GetSnapshots)", got)
	}
}

// TestOpenOrderModal_WarmsLotSize verifies that opening the modal from the
// positions table also resolves the lot in the background.
func TestOpenOrderModal_WarmsLotSize(t *testing.T) {
	called := make(chan [2]string, 1)
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 10 },
		EnsureLotSizeFunc: func(accountID, symbol string) float64 {
			called <- [2]string{accountID, symbol}
			return 5
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Ticker: "SBER", Quantity: "100", CurrentPrice: "280"}}
	app.portfolioView.TabbedView.PositionsTable.Select(1, 0)

	app.OpenOrderModal()

	select {
	case got := <-called:
		if got[0] != "acc1" || got[1] != "SBER" {
			t.Errorf("EnsureLotSize called with (%q, %q), want (\"acc1\", \"SBER\")", got[0], got[1])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("EnsureLotSize was not called for the opened modal")
	}
}

// TestShowModifyOrderModal_WarmsLotSize verifies the same for the modify path,
// which pre-fills a lot count derived from the lot size.
func TestShowModifyOrderModal_WarmsLotSize(t *testing.T) {
	called := make(chan [2]string, 1)
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 10 },
		EnsureLotSizeFunc: func(accountID, symbol string) float64 {
			called <- [2]string{accountID, symbol}
			return 5
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Limit", Status: "Active", Quantity: "100", LimitPrice: "280"},
	}
	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	if got := app.orderModal.GetQuantity(); got != 10 {
		t.Errorf("pre-filled quantity = %v, want 10 (100 shares / lot 10)", got)
	}

	select {
	case got := <-called:
		if got[0] != "acc1" || got[1] != "SBER" {
			t.Errorf("EnsureLotSize called with (%q, %q), want (\"acc1\", \"SBER\")", got[0], got[1])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("EnsureLotSize was not called for the modify modal")
	}
}
