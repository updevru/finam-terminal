package ui

import (
	"finam-terminal/models"
)

// mockClient is a shared mock for testing UI components
type mockClient struct {
	GetAccountsFunc       func() ([]models.AccountInfo, error)
	GetAccountDetailsFunc func(accountID string) (*models.AccountInfo, []models.Position, error)
	GetQuotesFunc         func(accountID string, symbols []string) (map[string]*models.Quote, error)
	PlaceOrderFunc        func(accountID string, symbol string, buySell string, quantity float64) (string, error)
	ClosePositionFunc     func(accountID string, symbol string, currentQuantity string, closeQuantity float64) (string, error)

	SearchSecuritiesFunc func(query string) ([]models.SecurityInfo, error)
	GetSnapshotsFunc     func(symbols []string) (map[string]models.Quote, error)
}

func (m *mockClient) GetAccounts() ([]models.AccountInfo, error) {
	if m.GetAccountsFunc != nil {
		return m.GetAccountsFunc()
	}
	return nil, nil
}

func (m *mockClient) GetAccountDetails(accountID string) (*models.AccountInfo, []models.Position, error) {
	if m.GetAccountDetailsFunc != nil {
		return m.GetAccountDetailsFunc(accountID)
	}
	return &models.AccountInfo{}, nil, nil
}

func (m *mockClient) GetQuotes(accountID string, symbols []string) (map[string]*models.Quote, error) {
	if m.GetQuotesFunc != nil {
		return m.GetQuotesFunc(accountID, symbols)
	}
	return make(map[string]*models.Quote), nil
}

func (m *mockClient) PlaceOrder(accountID string, symbol string, buySell string, quantity float64) (string, error) {
	if m.PlaceOrderFunc != nil {
		return m.PlaceOrderFunc(accountID, symbol, buySell, quantity)
	}
	return "tx-123", nil
}

func (m *mockClient) ClosePosition(accountID string, symbol string, currentQuantity string, closeQuantity float64) (string, error) {
	if m.ClosePositionFunc != nil {
		return m.ClosePositionFunc(accountID, symbol, currentQuantity, closeQuantity)
	}
	return "tx-123", nil
}

func (m *mockClient) SearchSecurities(query string) ([]models.SecurityInfo, error) {
	if m.SearchSecuritiesFunc != nil {
		return m.SearchSecuritiesFunc(query)
	}
	return nil, nil
}

func (m *mockClient) GetSnapshots(symbols []string) (map[string]models.Quote, error) {
	if m.GetSnapshotsFunc != nil {
		return m.GetSnapshotsFunc(symbols)
	}
	return make(map[string]models.Quote), nil
}
