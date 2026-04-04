package testserver

import (
	"context"
	"fmt"
	"sync"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/orders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockOrdersServer implements orders.OrdersServiceServer for testing.
type MockOrdersServer struct {
	orders.UnimplementedOrdersServiceServer

	// ActiveOrders keyed by account ID.
	ActiveOrders map[string][]*orders.OrderState

	// RecordedOrders stores all PlaceOrder requests for assertion.
	RecordedOrders []*orders.Order

	// RecordedSLTPOrders stores all PlaceSLTPOrder requests.
	RecordedSLTPOrders []*orders.SLTPOrder

	// RecordedCancellations stores all CancelOrder requests.
	RecordedCancellations []*orders.CancelOrderRequest

	// PlaceOrderError, if set, is returned by PlaceOrder.
	PlaceOrderError error

	// CancelOrderError, if set, is returned by CancelOrder.
	CancelOrderError error

	Mu          sync.Mutex
	nextOrderID int
}

// NewMockOrdersServer creates a MockOrdersServer with default data.
func NewMockOrdersServer() *MockOrdersServer {
	return &MockOrdersServer{
		ActiveOrders: map[string][]*orders.OrderState{
			"ACC001": DefaultOrders("ACC001"),
		},
		nextOrderID: 100,
	}
}

// PlaceOrder records the request and returns a new order ID.
func (m *MockOrdersServer) PlaceOrder(_ context.Context, req *orders.Order) (*orders.OrderState, error) {
	if m.PlaceOrderError != nil {
		return nil, m.PlaceOrderError
	}

	m.Mu.Lock()
	m.RecordedOrders = append(m.RecordedOrders, req)
	orderID := m.nextOrderID
	m.nextOrderID++
	m.Mu.Unlock()

	return &orders.OrderState{
		OrderId: fmt.Sprintf("ORD%03d", orderID),
		Status:  orders.OrderStatus_ORDER_STATUS_NEW,
		Order:   req,
	}, nil
}

// PlaceSLTPOrder records the request and returns a new order ID.
func (m *MockOrdersServer) PlaceSLTPOrder(_ context.Context, req *orders.SLTPOrder) (*orders.OrderState, error) {
	m.Mu.Lock()
	m.RecordedSLTPOrders = append(m.RecordedSLTPOrders, req)
	orderID := m.nextOrderID
	m.nextOrderID++
	m.Mu.Unlock()

	return &orders.OrderState{
		OrderId: fmt.Sprintf("ORD%03d", orderID),
		Status:  orders.OrderStatus_ORDER_STATUS_NEW,
	}, nil
}

// CancelOrder records the cancellation.
func (m *MockOrdersServer) CancelOrder(_ context.Context, req *orders.CancelOrderRequest) (*orders.OrderState, error) {
	if m.CancelOrderError != nil {
		return nil, m.CancelOrderError
	}

	m.Mu.Lock()
	m.RecordedCancellations = append(m.RecordedCancellations, req)
	m.Mu.Unlock()

	return &orders.OrderState{
		OrderId: req.OrderId,
		Status:  orders.OrderStatus_ORDER_STATUS_CANCELED,
	}, nil
}

// GetOrders returns active orders for the account.
func (m *MockOrdersServer) GetOrders(_ context.Context, req *orders.OrdersRequest) (*orders.OrdersResponse, error) {
	activeOrders := m.ActiveOrders[req.AccountId]
	if activeOrders == nil {
		return nil, status.Errorf(codes.NotFound, "no orders for account %s", req.AccountId)
	}
	return &orders.OrdersResponse{
		Orders: activeOrders,
	}, nil
}
