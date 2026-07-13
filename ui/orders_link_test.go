package ui

import (
	"finam-terminal/models"
	"strings"
	"testing"
)

// TestOrdersTable_TriggeredLinkMarker verifies the ↳ cross-reference marker:
// a parent stop order and the exchange order it triggered are both marked with
// the linked order's ID when both are present in the active set.
func TestOrdersTable_TriggeredLinkMarker(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "ORD001", Symbol: "SBER@TQBR", Name: "Сбербанк", Side: "Buy", Type: "Stop", Status: "New", Quantity: "10", TriggeredOrderID: "ORD002"},
		{ID: "ORD002", Symbol: "SBER@TQBR", Name: "Сбербанк", Side: "Buy", Type: "Limit", Status: "New", Quantity: "10"},
	}

	updateOrdersTable(app)
	table := app.portfolioView.TabbedView.OrdersTable

	// Parent row references the child it spawned.
	parentCell := table.GetCell(1, 0).Text
	if !strings.Contains(parentCell, "↳") || !strings.Contains(parentCell, "ORD002") {
		t.Errorf("parent row: expected ↳ marker referencing ORD002, got %q", parentCell)
	}
	// Child row references its parent.
	childCell := table.GetCell(2, 0).Text
	if !strings.Contains(childCell, "↳") || !strings.Contains(childCell, "ORD001") {
		t.Errorf("child row: expected ↳ marker referencing ORD001, got %q", childCell)
	}
}

// TestOrdersTable_TriggeredLinkOneSided verifies graceful degradation when only
// one side of the link is present in the active set — no panic, and the parent
// still shows the marker even if the child has already executed and left the set.
func TestOrdersTable_TriggeredLinkOneSided(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		// Parent present, child (ORD999) absent from the set.
		{ID: "ORD001", Symbol: "SBER@TQBR", Name: "Сбербанк", Side: "Buy", Type: "Stop", Status: "New", Quantity: "10", TriggeredOrderID: "ORD999"},
		// Unrelated order with no link.
		{ID: "ORD500", Symbol: "GAZP@TQBR", Name: "Газпром", Side: "Sell", Type: "Market", Status: "New", Quantity: "5"},
	}

	// Must not panic.
	updateOrdersTable(app)
	table := app.portfolioView.TabbedView.OrdersTable

	parentCell := table.GetCell(1, 0).Text
	if !strings.Contains(parentCell, "↳") || !strings.Contains(parentCell, "ORD999") {
		t.Errorf("parent row (one-sided): expected ↳ marker referencing ORD999, got %q", parentCell)
	}
	// Unrelated order has no marker.
	unrelatedCell := table.GetCell(2, 0).Text
	if strings.Contains(unrelatedCell, "↳") {
		t.Errorf("unrelated row: expected no ↳ marker, got %q", unrelatedCell)
	}
}
