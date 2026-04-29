// Package grpcclient реализует gRPC-клиент к Auth-сервису.
//
// Support-сервис вызывает Auth для:
//   - ValidateToken: проверка JWT токена (auth middleware)
//   - CheckRole / GetUserRole: проверка роли support/admin
package grpcclient

import (
	"context"
	"fmt"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient обёртка над gRPC-клиентом Auth-сервиса.
type AuthClient struct {
	client authv1.AuthServiceClient
	conn   *grpc.ClientConn
}

// NewAuthClient создаёт подключение к Auth gRPC серверу.
func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpcclient.NewAuthClient: %w", err)
	}

	return &AuthClient{
		client: authv1.NewAuthServiceClient(conn),
		conn:   conn,
	}, nil
}

// Close закрывает gRPC соединение.
func (c *AuthClient) Close() error {
	return c.conn.Close()
}

// ValidateToken проверяет JWT токен через Auth-сервис.
// Возвращает user_id и роль пользователя.
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (int64, string, error) {
	resp, err := c.client.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return 0, "", fmt.Errorf("auth.ValidateToken: %w", err)
	}

	return resp.GetUserId(), resp.GetRole(), nil
}

// GetUserRole возвращает роль пользователя по ID.
// Реализует интерфейс middleware.RoleProvider и supportmessage.RoleProvider.
func (c *AuthClient) GetUserRole(ctx context.Context, userID int64) (string, error) {
	resp, err := c.client.CheckRole(ctx, &authv1.CheckRoleRequest{
		UserId:       userID,
		RequiredRole: "user",
	})
	if err != nil {
		return "", fmt.Errorf("auth.GetUserRole: %w", err)
	}

	return resp.GetActualRole(), nil
}
