package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
	userrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	roleUser    = "user"
	roleSupport = "support"
)

type stubUserProvider struct {
	byID func(ctx context.Context, id int64) (models.User, error)
	role func(ctx context.Context, id int64) (string, error)
}

func (s stubUserProvider) UserByID(ctx context.Context, id int64) (models.User, error) {
	return s.byID(ctx, id)
}

func (s stubUserProvider) GetUserRole(ctx context.Context, id int64) (string, error) {
	return s.role(ctx, id)
}

type stubTokenValidator struct {
	fn func(ctx context.Context, token string) (models.User, error)
}

func (s stubTokenValidator) ValidateTokenAndGetUser(ctx context.Context, token string) (models.User, error) {
	return s.fn(ctx, token)
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func codeOf(t *testing.T, err error) codes.Code {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %v", err)
	return st.Code()
}

func TestValidateToken(t *testing.T) {
	t.Run("empty token -> InvalidArgument", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{})
		_, err := s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: ""})
		assert.Equal(t, codes.InvalidArgument, codeOf(t, err))
	})

	t.Run("validator error -> Unauthenticated", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{
			fn: func(ctx context.Context, token string) (models.User, error) {
				return models.User{}, errors.New("expired")
			},
		})
		_, err := s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: "x"})
		assert.Equal(t, codes.Unauthenticated, codeOf(t, err))
	})

	t.Run("ok", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{
			fn: func(ctx context.Context, token string) (models.User, error) {
				return models.User{ID: 42, Role: roleUser}, nil
			},
		})
		resp, err := s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: "x"})
		require.NoError(t, err)
		assert.Equal(t, int64(42), resp.GetUserId())
		assert.Equal(t, roleUser, resp.GetRole())
	})
}

func TestGetUser(t *testing.T) {
	t.Run("zero id -> InvalidArgument", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{})
		_, err := s.GetUser(context.Background(), &authv1.GetUserRequest{UserId: 0})
		assert.Equal(t, codes.InvalidArgument, codeOf(t, err))
	})

	t.Run("not found -> NotFound", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			byID: func(ctx context.Context, id int64) (models.User, error) {
				return models.User{}, userrepo.ErrUserNotFound
			},
		}, stubTokenValidator{})
		_, err := s.GetUser(context.Background(), &authv1.GetUserRequest{UserId: 5})
		assert.Equal(t, codes.NotFound, codeOf(t, err))
	})

	t.Run("internal error", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			byID: func(ctx context.Context, id int64) (models.User, error) {
				return models.User{}, errors.New("db down")
			},
		}, stubTokenValidator{})
		_, err := s.GetUser(context.Background(), &authv1.GetUserRequest{UserId: 5})
		assert.Equal(t, codes.Internal, codeOf(t, err))
	})

	t.Run("ok", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			byID: func(ctx context.Context, id int64) (models.User, error) {
				return models.User{ID: 7, Email: "a@b", Name: "Bob", Role: roleUser, Rating: 4.2}, nil
			},
		}, stubTokenValidator{})
		resp, err := s.GetUser(context.Background(), &authv1.GetUserRequest{UserId: 7})
		require.NoError(t, err)
		assert.Equal(t, int64(7), resp.GetId())
		assert.Equal(t, "a@b", resp.GetEmail())
		assert.Equal(t, "Bob", resp.GetFirstName())
		assert.Equal(t, roleUser, resp.GetRole())
		assert.InDelta(t, 4.2, resp.GetRating(), 0.001)
	})
}

func TestGetUsersByIDs(t *testing.T) {
	t.Run("empty ids -> empty response", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{})
		resp, err := s.GetUsersByIDs(context.Background(), &authv1.GetUsersByIDsRequest{})
		require.NoError(t, err)
		assert.Empty(t, resp.GetUsers())
	})

	t.Run("skips not found and internal errors", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			byID: func(ctx context.Context, id int64) (models.User, error) {
				switch id {
				case 1:
					return models.User{ID: 1, Name: "one"}, nil
				case 2:
					return models.User{}, userrepo.ErrUserNotFound
				case 3:
					return models.User{}, errors.New("db down")
				default:
					return models.User{ID: id}, nil
				}
			},
		}, stubTokenValidator{})

		resp, err := s.GetUsersByIDs(context.Background(), &authv1.GetUsersByIDsRequest{
			UserIds: []int64{1, 2, 3, 4},
		})
		require.NoError(t, err)
		assert.Len(t, resp.GetUsers(), 2)
	})
}

func TestCheckRole(t *testing.T) {
	t.Run("zero id -> InvalidArgument", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{})
		_, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 0, RequiredRole: roleAdmin})
		assert.Equal(t, codes.InvalidArgument, codeOf(t, err))
	})

	t.Run("missing required_role -> InvalidArgument", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{}, stubTokenValidator{})
		_, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: ""})
		assert.Equal(t, codes.InvalidArgument, codeOf(t, err))
	})

	t.Run("user not found -> NotFound", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			role: func(ctx context.Context, id int64) (string, error) {
				return "", userrepo.ErrUserNotFound
			},
		}, stubTokenValidator{})
		_, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: roleAdmin})
		assert.Equal(t, codes.NotFound, codeOf(t, err))
	})

	t.Run("internal error", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			role: func(ctx context.Context, id int64) (string, error) {
				return "", errors.New("db down")
			},
		}, stubTokenValidator{})
		_, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: roleAdmin})
		assert.Equal(t, codes.Internal, codeOf(t, err))
	})

	t.Run("admin allowed for any role", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			role: func(ctx context.Context, id int64) (string, error) { return roleAdmin, nil },
		}, stubTokenValidator{})
		resp, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: roleSupport})
		require.NoError(t, err)
		assert.True(t, resp.GetAllowed())
		assert.Equal(t, roleAdmin, resp.GetActualRole())
	})

	t.Run("matching role allowed", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			role: func(ctx context.Context, id int64) (string, error) { return roleSupport, nil },
		}, stubTokenValidator{})
		resp, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: roleSupport})
		require.NoError(t, err)
		assert.True(t, resp.GetAllowed())
	})

	t.Run("mismatched role forbidden", func(t *testing.T) {
		s := NewServer(newTestLogger(), stubUserProvider{
			role: func(ctx context.Context, id int64) (string, error) { return roleUser, nil },
		}, stubTokenValidator{})
		resp, err := s.CheckRole(context.Background(), &authv1.CheckRoleRequest{UserId: 1, RequiredRole: roleAdmin})
		require.NoError(t, err)
		assert.False(t, resp.GetAllowed())
	})
}
