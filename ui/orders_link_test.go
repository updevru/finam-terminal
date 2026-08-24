package ui

import (
	"finam-terminal/models"
	"strings"
	"testing"
)

// TestOrdersTable_TriggeredLinkGrouping verifies that when a parent stop order
// and the exchange order it triggered are both present, the child is rendered
// directly under its parent with a ↳ marker (unambiguous by adjacency) and the
// parent itself carries no marker.
func TestOrdersTable_TriggeredLinkGrouping(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "ORD001", Symbol: "SBER@TQBR", Name: "Сбербанк-STOP", Side: "Buy", Type: "Stop", Status: "New", Quantity: "10", TriggeredOrderID: "ORD002"},
		{ID: "ORD002", Symbol: "SBER@TQBR", Name: "Сбербанк-EXCH", Side: "Buy", Type: "Limit", Status: "New", Quantity: "10"},
	}

	updateOrdersTable(app)
	table := app.portfolioView.TabbedView.OrdersTable

	// Row 1: parent, no marker.
	parentCell := table.GetCell(1, 0).Text
	if strings.Contains(parentCell, "↳") {
		t.Errorf("parent row should carry no ↳ marker, got %q", parentCell)
	}
	if !strings.Contains(parentCell, "Сбербанк-STOP") {
		t.Errorf("row 1 should be the parent, got %q", parentCell)
	}
	// Row 2: child rendered directly under the parent, indented with ↳.
	childCell := table.GetCell(2, 0).Text
	if !strings.Contains(childCell, "↳") {
		t.Errorf("child row should carry a ↳ marker, got %q", childCell)
	}
	if !strings.Contains(childCell, "Сбербанк-EXCH") {
		t.Errorf("row 2 should be the triggered child, got %q", childCell)
	}
}

// TestOrdersTable_TriggeredLinkGroupingParentAfterChild verifies grouping works
// regardless of the original order (child listed before its parent).
func TestOrdersTable_TriggeredLinkGroupingParentAfterChild(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "ORD002", Symbol: "SBER@TQBR", Name: "Сбербанк-EXCH", Side: "Buy", Type: "Limit", Status: "New", Quantity: "10"},
		{ID: "ORD001", Symbol: "SBER@TQBR", Name: "Сбербанк-STOP", Side: "Buy", Type: "Stop", Status: "New", Quantity: "10", TriggeredOrderID: "ORD002"},
	}

	updateOrdersTable(app)
	table := app.portfolioView.TabbedView.OrdersTable

	// Parent emitted first (it owns the link), child grouped right under it.
	if got := table.GetCell(1, 0).Text; !strings.Contains(got, "Сбербанк-STOP") || strings.Contains(got, "↳") {
		t.Errorf("row 1 should be the parent without marker, got %q", got)
	}
	if got := table.GetCell(2, 0).Text; !strings.Contains(got, "Сбербанк-EXCH") || !strings.Contains(got, "↳") {
		t.Errorf("row 2 should be the grouped child with ↳, got %q", got)
	}
}

// TestOrdersTable_TriggeredLinkOneSided verifies that when only one side of the
// link is present (the counterpart already executed and left the set), the order
// is rendered normally without a marker — no noise, no panic.
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

	// No marker on either row — the linked counterpart is not in the set.
	if got := table.GetCell(1, 0).Text; strings.Contains(got, "↳") {
		t.Errorf("one-sided parent should have no ↳ marker, got %q", got)
	}
	if got := table.GetCell(2, 0).Text; strings.Contains(got, "↳") {
		t.Errorf("unrelated row should have no ↳ marker, got %q", got)
	}
}
