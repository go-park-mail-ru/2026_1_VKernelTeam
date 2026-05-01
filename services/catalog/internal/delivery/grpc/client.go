// Package grpc — клиент к Auth gRPC из catalog-сервиса.
//
// Используется для:
//   - ValidateToken: проверка JWT токена (auth middleware)
//   - GetUser: данные продавца в ответе ad-by-id (опционально, в будущем)
package grpc

import (
	"context"
	"fmt"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient обёртка над gRPC-клиентом Auth.
type AuthClient struct {
	client authv1.AuthServiceClient
	conn   *grpclib.ClientConn
}

// NewAuthClient создаёт подключение к Auth gRPC серверу.
func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpclib.NewClient(addr,
		grpclib.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewAuthClient: %w", err)
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

// ValidateToken проверяет JWT через Auth-сервис, возвращает user_id и роль.
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (int64, string, error) {
	resp, err := c.client.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
	if err != nil {
		return 0, "", fmt.Errorf("auth.ValidateToken: %w", err)
	}
	return resp.GetUserId(), resp.GetRole(), nil
}
