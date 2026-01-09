package api

import (
	"context"
	"testing"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
)

// mockAccountsServiceClient is a manual mock for accounts.AccountsServiceClient
type mockAccountsServiceClient struct {
	accounts.AccountsServiceClient
	GetAccountFunc func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error)
}

func (m *mockAccountsServiceClient) GetAccount(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
	return m.GetAccountFunc(ctx, in, opts...)
}

// mockAuthServiceClient is a manual mock for auth.AuthServiceClient
type mockAuthServiceClient struct {
	auth.AuthServiceClient
	TokenDetailsFunc func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error)
}

func (m *mockAuthServiceClient) TokenDetails(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
	return m.TokenDetailsFunc(ctx, in, opts...)
}

func TestGetAccountDetails(t *testing.T) {
	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId: in.AccountId,
				Type:      "test-type",
				Status:    "test-status",
				Equity: &decimal.Decimal{
					Value: "100",
				},
				UnrealizedProfit: &decimal.Decimal{
					Value: "10.5",
				},
				Positions: []*accounts.Position{
					{
						Symbol:       "GAZP",
						Quantity:     &decimal.Decimal{Value: "10"},
						AveragePrice: &decimal.Decimal{Value: "150"},
						CurrentPrice: &decimal.Decimal{Value: "160"},
						DailyPnl:     &decimal.Decimal{Value: "100"},
						UnrealizedPnl: &decimal.Decimal{Value: "100"},
					},
				},
			}, nil
		},
	}

	client := &Client{
		accountsClient: mockAccounts,
		assetMicCache: map[string]string{
			"GAZP": "GAZP@TQBR",
		},
	}

	account, positions, err := client.GetAccountDetails("test-acc")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if account.ID != "test-acc" {
		t.Errorf("Expected ID test-acc, got %s", account.ID)
	}
	if account.Equity != "100" {
		t.Errorf("Expected Equity 100, got %s", account.Equity)
	}
	if account.UnrealizedPnL != "10.5" {
		t.Errorf("Expected UnrealizedPnL 10.5, got %s", account.UnrealizedPnL)
	}
	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}
	if positions[0].Symbol != "GAZP@TQBR" {
		t.Errorf("Expected symbol GAZP@TQBR, got %s", positions[0].Symbol)
	}
	if positions[0].Ticker != "GAZP" || positions[0].MIC != "TQBR" {
		t.Errorf("Ticker/MIC mismatch")
	}
}

func TestGetAccounts(t *testing.T) {
	mockAuth := &mockAuthServiceClient{
		TokenDetailsFunc: func(ctx context.Context, in *auth.TokenDetailsRequest, opts ...grpc.CallOption) (*auth.TokenDetailsResponse, error) {
			return &auth.TokenDetailsResponse{
				AccountIds: []string{"acc1", "acc2"},
			}, nil
		},
	}

	mockAccounts := &mockAccountsServiceClient{
		GetAccountFunc: func(ctx context.Context, in *accounts.GetAccountRequest, opts ...grpc.CallOption) (*accounts.GetAccountResponse, error) {
			return &accounts.GetAccountResponse{
				AccountId: in.AccountId,
				Type:      "test-type",
				Status:    "test-status",
			}, nil
		},
	}

	client := &Client{
		authClient:     mockAuth,
		accountsClient: mockAccounts,
	}

	accs, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(accs) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accs))
	}
	if accs[0].ID != "acc1" || accs[1].ID != "acc2" {
		t.Errorf("Account IDs mismatch")
	}
}