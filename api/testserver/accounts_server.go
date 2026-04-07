package testserver

import (
	"context"

	tradeapiv1 "github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"google.golang.org/genproto/googleapis/type/decimal"
)

// MockAccountsServer implements accounts.AccountsServiceServer for testing.
type MockAccountsServer struct {
	accounts.UnimplementedAccountsServiceServer

	// Positions keyed by account ID.
	Positions map[string][]*accounts.Position

	// TradeHistory keyed by account ID.
	TradeHistory map[string][]*tradeapiv1.AccountTrade

	// GetAccountError, if set, is returned by GetAccount.
	GetAccountError error
}

// NewMockAccountsServer creates a MockAccountsServer with default data.
func NewMockAccountsServer() *MockAccountsServer {
	return &MockAccountsServer{
		Positions: map[string][]*accounts.Position{
			"ACC001": DefaultAccountPositions("ACC001"),
		},
		TradeHistory: map[string][]*tradeapiv1.AccountTrade{
			"ACC001": DefaultTrades("ACC001"),
		},
	}
}

// GetAccount returns account details with positions.
func (m *MockAccountsServer) GetAccount(_ context.Context, req *accounts.GetAccountRequest) (*accounts.GetAccountResponse, error) {
	if m.GetAccountError != nil {
		return nil, m.GetAccountError
	}

	positions := m.Positions[req.AccountId]
	return &accounts.GetAccountResponse{
		AccountId: req.AccountId,
		Equity:    &decimal.Decimal{Value: "500000.00"},
		Positions: positions,
	}, nil
}

// Trades returns trade history.
func (m *MockAccountsServer) Trades(_ context.Context, req *accounts.TradesRequest) (*accounts.TradesResponse, error) {
	trades := m.TradeHistory[req.AccountId]
	return &accounts.TradesResponse{
		Trades: trades,
	}, nil
}
