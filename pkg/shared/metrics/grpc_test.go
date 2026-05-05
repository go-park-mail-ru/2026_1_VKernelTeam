package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor_OK(t *testing.T) {
	m := NewGRPC("test_grpc_ok")
	interceptor := m.UnaryServerInterceptor()

	resp, err := interceptor(
		context.Background(),
		"req",
		&grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/ValidateToken"},
		func(ctx context.Context, req any) (any, error) { return "resp", nil },
	)

	require.NoError(t, err)
	assert.Equal(t, "resp", resp)
	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("/auth.v1.AuthService/ValidateToken", "OK")))
}

func TestUnaryServerInterceptor_StatusError(t *testing.T) {
	m := NewGRPC("test_grpc_err")
	interceptor := m.UnaryServerInterceptor()

	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/GetUser"},
		func(ctx context.Context, req any) (any, error) {
			return nil, status.Error(codes.NotFound, "missing")
		},
	)

	require.Error(t, err)
	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("/auth.v1.AuthService/GetUser", "NotFound")))
}

func TestUnaryServerInterceptor_PlainError(t *testing.T) {
	m := NewGRPC("test_grpc_plain")
	interceptor := m.UnaryServerInterceptor()

	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/CheckRole"},
		func(ctx context.Context, req any) (any, error) { return nil, errors.New("boom") },
	)

	require.Error(t, err)
	// status.Code на nil-Error возвращает Unknown.
	assert.Equal(t, float64(1), testutil.ToFloat64(m.requests.WithLabelValues("/auth.v1.AuthService/CheckRole", "Unknown")))
}

func TestNewGRPC_IsIdempotent(t *testing.T) {
	a := NewGRPC("test_grpc_idem")
	b := NewGRPC("test_grpc_idem")
	assert.Same(t, a, b)
}
