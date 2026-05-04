package grpc

import (
	"context"
	"errors"
	"testing"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/grpc/mocks"
	"github.com/golang/mock/gomock"
)

func TestAuthClient_ValidateToken_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAuthServiceClient(ctrl)
	mock.EXPECT().
		ValidateToken(gomock.Any(), &authv1.ValidateTokenRequest{Token: "tok"}).
		Return(&authv1.ValidateTokenResponse{UserId: 42, Role: "user"}, nil)

	c := &AuthClient{client: mock}
	uid, role, err := c.ValidateToken(context.Background(), "tok")
	if err != nil || uid != 42 || role != "user" {
		t.Fatalf("unexpected: %d/%s/%v", uid, role, err)
	}
}

func TestAuthClient_ValidateToken_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockAuthServiceClient(ctrl)
	mock.EXPECT().
		ValidateToken(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("rpc fail"))

	c := &AuthClient{client: mock}
	if _, _, err := c.ValidateToken(context.Background(), "tok"); err == nil {
		t.Fatal("expected error")
	}
}
