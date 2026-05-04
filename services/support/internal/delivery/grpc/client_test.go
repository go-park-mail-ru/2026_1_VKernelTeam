package grpc

import (
	"context"
	"net"
	"testing"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakeAuthServer — заглушка AuthService для bufconn-теста.
type fakeAuthServer struct {
	authv1.UnimplementedAuthServiceServer

	validateFn  func(req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error)
	checkRoleFn func(req *authv1.CheckRoleRequest) (*authv1.CheckRoleResponse, error)
}

func (f *fakeAuthServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	return f.validateFn(req)
}

func (f *fakeAuthServer) CheckRole(ctx context.Context, req *authv1.CheckRoleRequest) (*authv1.CheckRoleResponse, error) {
	return f.checkRoleFn(req)
}

// startFakeAuth поднимает gRPC-сервер на bufconn и возвращает AuthClient к нему.
func startFakeAuth(t *testing.T, fake *fakeAuthServer) (*AuthClient, func()) {
	t.Helper()
	const buf = 1024 * 1024
	lis := bufconn.Listen(buf)

	srv := grpclib.NewServer()
	authv1.RegisterAuthServiceServer(srv, fake)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpclib.NewClient(
		"passthrough:///bufnet",
		grpclib.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpclib.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := &AuthClient{
		client: authv1.NewAuthServiceClient(conn),
		conn:   conn,
	}

	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	}
	return client, cleanup
}

func TestNewAuthClient_OK(t *testing.T) {
	c, err := NewAuthClient("localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NoError(t, c.Close())
}

func TestValidateToken_OK(t *testing.T) {
	fake := &fakeAuthServer{
		validateFn: func(req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
			assert.Equal(t, "tok", req.GetToken())
			return &authv1.ValidateTokenResponse{UserId: 7, Role: "user"}, nil
		},
	}
	client, cleanup := startFakeAuth(t, fake)
	defer cleanup()

	uid, role, err := client.ValidateToken(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, int64(7), uid)
	assert.Equal(t, "user", role)
}

func TestValidateToken_Error(t *testing.T) {
	fake := &fakeAuthServer{
		validateFn: func(req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "bad token")
		},
	}
	client, cleanup := startFakeAuth(t, fake)
	defer cleanup()

	_, _, err := client.ValidateToken(context.Background(), "bad")
	assert.Error(t, err)
}

func TestGetUserRole_OK(t *testing.T) {
	fake := &fakeAuthServer{
		checkRoleFn: func(req *authv1.CheckRoleRequest) (*authv1.CheckRoleResponse, error) {
			assert.Equal(t, int64(42), req.GetUserId())
			return &authv1.CheckRoleResponse{ActualRole: "admin", Allowed: true}, nil
		},
	}
	client, cleanup := startFakeAuth(t, fake)
	defer cleanup()

	role, err := client.GetUserRole(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, "admin", role)
}

func TestGetUserRole_Error(t *testing.T) {
	fake := &fakeAuthServer{
		checkRoleFn: func(req *authv1.CheckRoleRequest) (*authv1.CheckRoleResponse, error) {
			return nil, status.Error(codes.NotFound, "user not found")
		},
	}
	client, cleanup := startFakeAuth(t, fake)
	defer cleanup()

	_, err := client.GetUserRole(context.Background(), 42)
	assert.Error(t, err)
}
