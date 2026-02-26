package auth

import (
	"context"

	ssov1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyEmailErr    = "email is required"
	emptyPasswordErr = "password is required"
	emptyAppIdErr    = "app_id is required"

	emptyEmail = ""
	emptyPassword = ""
	emptyAppId = 0
	emptyUserId = 0
)

type serverApi struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

type Auth interface {
	Login(ctx context.Context, email, password string, appId int64) (string, error)
	RegisterNewUser(ctx context.Context, email, password string) (int64, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

func RegisterServerApi(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &serverApi{auth: auth})
}

func (s *serverApi) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if req.GetEmail() == emptyEmail {
		return nil, status.Error(codes.InvalidArgument, emptyEmailErr)
	}

	if req.GetPassword() == emptyPassword {
		return nil, status.Error(codes.InvalidArgument, emptyPasswordErr)
	}

	if req.GetAppId() == emptyAppId {
		return nil, status.Error(codes.InvalidArgument, emptyAppIdErr)
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), int64(req.GetAppId()))
	if err != nil {
		return nil, err
	}
	return &ssov1.LoginResponse{Token: token}, nil


}

func (s *serverApi) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if req.GetEmail() == emptyEmail {
		return nil, status.Error(codes.InvalidArgument, emptyEmailErr)
	}

	if req.GetPassword() == emptyPassword {
		return nil, status.Error(codes.InvalidArgument, emptyPasswordErr)
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		// TODO errors
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &ssov1.RegisterResponse{UserId: userID}, nil
}

func (s *serverApi) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponse, error) {
	if req.GetUserId() == emptyUserId {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	idAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &ssov1.IsAdminResponse{IsAdmin: idAdmin}, nil

}
