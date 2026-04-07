package testserver

import (
	"context"
	"net"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/accounts"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/assets"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// TestServer is an in-process gRPC server for integration testing.
type TestServer struct {
	server   *grpc.Server
	listener *bufconn.Listener

	Auth       *MockAuthServer
	Accounts   *MockAccountsServer
	MarketData *MockMarketDataServer
	Assets     *MockAssetsServer
	Orders     *MockOrdersServer
}

// NewTestServer creates a new TestServer with all mock services registered.
func NewTestServer() *TestServer {
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()

	ts := &TestServer{
		server:     srv,
		listener:   lis,
		Auth:       NewMockAuthServer(),
		Accounts:   NewMockAccountsServer(),
		MarketData: NewMockMarketDataServer(),
		Assets:     NewMockAssetsServer(),
		Orders:     NewMockOrdersServer(),
	}

	auth.RegisterAuthServiceServer(srv, ts.Auth)
	accounts.RegisterAccountsServiceServer(srv, ts.Accounts)
	marketdata.RegisterMarketDataServiceServer(srv, ts.MarketData)
	assets.RegisterAssetsServiceServer(srv, ts.Assets)
	orders.RegisterOrdersServiceServer(srv, ts.Orders)

	return ts
}

// Start begins serving in a background goroutine.
func (ts *TestServer) Start() {
	go func() {
		_ = ts.server.Serve(ts.listener)
	}()
}

// Stop gracefully stops the server.
func (ts *TestServer) Stop() {
	ts.server.GracefulStop()
}

// Dial returns a client connection to the in-process server.
func (ts *TestServer) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return ts.listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
