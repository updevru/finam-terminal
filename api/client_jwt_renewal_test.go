package api

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/grpc"
)

func TestNextBackoff(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want time.Duration
	}{
		{jwtRenewalInitialBackoff, 2 * time.Second},
		{16 * time.Second, jwtRenewalMaxBackoff},
		{jwtRenewalMaxBackoff, jwtRenewalMaxBackoff},
	}

	for _, c := range cases {
		if got := nextBackoff(c.in); got != c.want {
			t.Errorf("nextBackoff(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSubscribeJwtRenewal_ReconnectsAfterStreamError(t *testing.T) {
	var callCount atomic.Int64
	firstRecvCalled := make(chan struct{})
	const secondToken = "second-token"

	mockAuth := &mockAuthServiceClient{
		SubscribeJwtRenewalFunc: func(ctx context.Context, in *auth.SubscribeJwtRenewalRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[auth.SubscribeJwtRenewalResponse], error) {
			n := callCount.Add(1)
			if n == 1 {
				return &fakeJwtRenewalStream{
					ctx: ctx,
					recvFunc: func() (*auth.SubscribeJwtRenewalResponse, error) {
						close(firstRecvCalled)
						return nil, errors.New("stream dropped")
					},
				}, nil
			}

			var sent atomic.Bool
			return &fakeJwtRenewalStream{
				ctx: ctx,
				recvFunc: func() (*auth.SubscribeJwtRenewalResponse, error) {
					if !sent.Swap(true) {
						return &auth.SubscribeJwtRenewalResponse{Token: secondToken}, nil
					}
					<-ctx.Done()
					return nil, ctx.Err()
				},
			}, nil
		},
	}

	client := &Client{authClient: mockAuth, apiToken: "test-secret"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.subscribeJwtRenewal(ctx)

	select {
	case <-firstRecvCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first SubscribeJwtRenewal stream to be established")
	}

	deadline := time.After(5 * time.Second)
	for {
		client.tokenMutex.RLock()
		tok := client.token
		client.tokenMutex.RUnlock()
		if tok == secondToken {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for token update after reconnect; callCount=%d", callCount.Load())
		case <-time.After(20 * time.Millisecond):
		}
	}

	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 SubscribeJwtRenewal calls (reconnect), got %d", callCount.Load())
	}
}

func TestSubscribeJwtRenewal_RetriesOnConnectError(t *testing.T) {
	var callCount atomic.Int64

	mockAuth := &mockAuthServiceClient{
		SubscribeJwtRenewalFunc: func(ctx context.Context, in *auth.SubscribeJwtRenewalRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[auth.SubscribeJwtRenewalResponse], error) {
			n := callCount.Add(1)
			if n == 1 {
				return nil, errors.New("connection refused")
			}
			return &fakeJwtRenewalStream{
				ctx: ctx,
				recvFunc: func() (*auth.SubscribeJwtRenewalResponse, error) {
					<-ctx.Done()
					return nil, ctx.Err()
				},
			}, nil
		},
	}

	client := &Client{authClient: mockAuth, apiToken: "test-secret"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.subscribeJwtRenewal(ctx)

	deadline := time.After(5 * time.Second)
	for callCount.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for retry after connect error; callCount=%d", callCount.Load())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestSubscribeJwtRenewal_StopsOnCancelWithoutErrorLogs(t *testing.T) {
	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	recvCalled := make(chan struct{})
	mockAuth := &mockAuthServiceClient{
		SubscribeJwtRenewalFunc: func(ctx context.Context, in *auth.SubscribeJwtRenewalRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[auth.SubscribeJwtRenewalResponse], error) {
			return &fakeJwtRenewalStream{
				ctx: ctx,
				recvFunc: func() (*auth.SubscribeJwtRenewalResponse, error) {
					close(recvCalled)
					<-ctx.Done()
					return nil, ctx.Err()
				},
			}, nil
		},
	}

	client := &Client{authClient: mockAuth, apiToken: "test-secret"}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		client.subscribeJwtRenewal(ctx)
		close(done)
	}()

	select {
	case <-recvCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream to be established")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("subscribeJwtRenewal did not stop after context cancellation")
	}

	logs := logBuf.String()
	if strings.Contains(logs, "[ERROR]") || strings.Contains(logs, "[WARN]") {
		t.Errorf("expected no ERROR/WARN logs on clean stop, got:\n%s", logs)
	}
}
