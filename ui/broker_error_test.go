package ui

import (
	"errors"
	"strings"
	"testing"

	"finam-terminal/models"
)

// TestBrokerDataError_IsPlainRussian verifies the message shown when data could
// not be loaded: a plain explanation plus the advice to try again later, rather
// than the broker's raw wire text.
func TestBrokerDataError_IsPlainRussian(t *testing.T) {
	msg := brokerDataError()

	if !strings.Contains(msg, "Ошибка при загрузке данных от брокера") {
		t.Errorf("message = %q, want it to state the load failure plainly", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "позже") {
		t.Errorf("message = %q, want it to advise retrying later", msg)
	}
}

// TestBrokerDataErrorWithKey_MentionsTheRetryKey verifies that where a manual
// refresh key exists, the message names it.
func TestBrokerDataErrorWithKey_MentionsTheRetryKey(t *testing.T) {
	msg := brokerDataErrorWithKey("R")

	if !strings.Contains(msg, "Ошибка при загрузке данных от брокера") {
		t.Errorf("message = %q, want the standard load-failure text", msg)
	}
	if !strings.Contains(msg, "R") {
		t.Errorf("message = %q, want it to name the refresh key", msg)
	}
}

// TestIndexTable_ErrorRowHidesBrokerText verifies the Index tab no longer prints
// the broker's raw message. The technical detail belongs in the log; the row the
// user reads should say what happened and what to do.
func TestIndexTable_ErrorRowHidesBrokerText(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexLoadErr = "rpc error: code = Unavailable desc = Invalid arguments:account_id"

	updateIndexTable(app)

	got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 0).Text
	for _, leak := range []string{"rpc error", "Unavailable", "account_id"} {
		if strings.Contains(got, leak) {
			t.Errorf("error row %q leaks the broker's raw text (%q)", got, leak)
		}
	}
	if !strings.Contains(got, "Ошибка при загрузке данных от брокера") {
		t.Errorf("error row = %q, want the standard load-failure message", got)
	}
	if !strings.Contains(got, "R") {
		t.Errorf("error row = %q, want it to name the refresh key", got)
	}
}

// TestLoadIndexSync_StillLogsTheDetail verifies the raw cause is not lost: it is
// recorded for diagnosis even though the user sees the plain message.
func TestLoadIndexSync_StillRecordsTheCause(t *testing.T) {
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			return nil, errors.New("Constituents not found")
		},
	}
	app := NewApp(mock, nil)

	app.loadIndexSync()

	app.dataMutex.RLock()
	defer app.dataMutex.RUnlock()
	if app.indexLoadErr == "" {
		t.Error("the failure was not recorded at all")
	}
}

// TestPositionsTable_BrokerErrorIsPlain verifies an account the broker could not
// load shows the same plain message instead of its raw error string.
func TestPositionsTable_BrokerErrorIsPlain(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{
		{ID: "acc1", LoadError: "rpc error: code = Internal desc = boom"},
	})
	app.selectedIdx = 0

	updatePositionsTable(app)

	got := app.portfolioView.TabbedView.PositionsTable.GetCell(1, 0).Text
	if strings.Contains(got, "rpc error") || strings.Contains(got, "boom") {
		t.Errorf("positions row %q leaks the broker's raw text", got)
	}
	if !strings.Contains(got, "Ошибка при загрузке данных от брокера") {
		t.Errorf("positions row = %q, want the standard load-failure message", got)
	}
}

// TestInfoPanel_BrokerErrorIsPlain verifies the account summary panel does the
// same.
func TestInfoPanel_BrokerErrorIsPlain(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{
		{ID: "acc1", LoadError: "rpc error: code = Internal desc = boom"},
	})
	app.selectedIdx = 0

	updateInfoPanel(app)

	got := app.portfolioView.SummaryArea.GetText(false)
	if strings.Contains(got, "rpc error") || strings.Contains(got, "boom") {
		t.Errorf("info panel %q leaks the broker's raw text", got)
	}
	if !strings.Contains(got, "Ошибка при загрузке данных от брокера") {
		t.Errorf("info panel = %q, want the standard load-failure message", got)
	}
}
