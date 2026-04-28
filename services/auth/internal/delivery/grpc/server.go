// Package grpc реализует gRPC-сервер AuthService.
//
// Зачем это нужно:
// В монолите все сервисы вызывали auth usecase напрямую через Go-интерфейсы.
// В микросервисной архитектуре Catalog, Commerce и Support — отдельные процессы.
// Они не могут вызвать Go-функцию в другом процессе. Поэтому Auth предоставляет
// gRPC API — бинарный протокол поверх HTTP/2, быстрее REST и с типизацией.
//
// Кто будет вызывать:
//   - Catalog: ValidateToken (в auth middleware), GetUser (данные продавца)
//   - Commerce: ValidateToken, GetUser/GetUsersByIDs (участники чата)
//   - Support: ValidateToken, CheckRole (проверка admin/support)
package grpc

import (
	"context"
	"errors"
	"log/slog"

	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
	userrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserProvider — интерфейс для получения данных пользователей из БД.
// Auth gRPC сервер работает напрямую с repository, а не через usecase,
// потому что gRPC методы — это простые запросы данных без бизнес-логики.
type UserProvider interface {
	UserByID(ctx context.Context, userID int64) (models.User, error)
	GetUserRole(ctx context.Context, userID int64) (string, error)
}

// TokenValidator — интерфейс для валидации JWT токенов.
type TokenValidator interface {
	ValidateTokenAndGetUser(ctx context.Context, tokenString string) (models.User, error)
}

// Server реализует authv1.AuthServiceServer — интерфейс,
// сгенерированный protoc из auth.proto.
type Server struct {
	// Встраиваем UnimplementedAuthServiceServer.
	// Это обязательно — если в proto добавится новый RPC метод,
	// старый код не сломается (вернёт "not implemented" по умолчанию).
	authv1.UnimplementedAuthServiceServer

	log            *slog.Logger
	userProvider   UserProvider
	tokenValidator TokenValidator
}

// NewServer создаёт gRPC сервер Auth.
func NewServer(
	log *slog.Logger,
	userProvider UserProvider,
	tokenValidator TokenValidator,
) *Server {
	return &Server{
		log:            log,
		userProvider:   userProvider,
		tokenValidator: tokenValidator,
	}
}

// ValidateToken проверяет JWT токен и возвращает user_id и роль.
//
// Это самый частый вызов — каждый запрос к Catalog/Commerce/Support
// проходит через auth middleware, который вызывает этот метод.
func (s *Server) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	// req.Token — это JWT строка из cookie, которую другой сервис получил от клиента

	if req.GetToken() == "" {
		// codes.InvalidArgument — gRPC-аналог HTTP 400
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	user, err := s.tokenValidator.ValidateTokenAndGetUser(ctx, req.GetToken())
	if err != nil {
		s.log.WarnContext(ctx, "gRPC ValidateToken failed",
			slog.String("error", err.Error()),
		)
		// codes.Unauthenticated — gRPC-аналог HTTP 401
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return &authv1.ValidateTokenResponse{
		UserId: user.ID,
		Role:   user.Role,
		// JTI пока не возвращаем — для blacklist проверки
	}, nil
}

// GetUser возвращает данные пользователя по ID.
//
// Используется когда другому сервису нужно показать информацию о пользователе:
// - Catalog: имя и аватар продавца в карточке объявления
// - Commerce: участники чата
func (s *Server) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.UserResponse, error) {
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.userProvider.UserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			// codes.NotFound — gRPC-аналог HTTP 404
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.ErrorContext(ctx, "gRPC GetUser failed",
			slog.Int64("user_id", req.GetUserId()),
			slog.String("error", err.Error()),
		)
		// codes.Internal — gRPC-аналог HTTP 500
		return nil, status.Error(codes.Internal, "internal error")
	}

	return userToProto(user), nil
}

// GetUsersByIDs — пакетное получение пользователей.
//
// Зачем пакетный метод: Commerce показывает список чатов, в каждом — покупатель
// и продавец. Вместо 20 отдельных GetUser вызовов, делаем один GetUsersByIDs.
// Это называется "batching" — уменьшает количество сетевых вызовов.
func (s *Server) GetUsersByIDs(ctx context.Context, req *authv1.GetUsersByIDsRequest) (*authv1.GetUsersByIDsResponse, error) {
	if len(req.GetUserIds()) == 0 {
		return &authv1.GetUsersByIDsResponse{}, nil
	}

	users := make([]*authv1.UserResponse, 0, len(req.GetUserIds()))

	for _, id := range req.GetUserIds() {
		user, err := s.userProvider.UserByID(ctx, id)
		if err != nil {
			if errors.Is(err, userrepo.ErrUserNotFound) {
				// Пропускаем несуществующих — не ломаем весь запрос из-за одного
				continue
			}
			s.log.ErrorContext(ctx, "gRPC GetUsersByIDs: failed to get user",
				slog.Int64("user_id", id),
				slog.String("error", err.Error()),
			)
			continue
		}
		users = append(users, userToProto(user))
	}

	return &authv1.GetUsersByIDsResponse{Users: users}, nil
}

// CheckRole проверяет, имеет ли пользователь нужную роль.
//
// Используется Support-сервисом для эндпоинтов админки:
// GET /support/tickets/all, PATCH /support/tickets/{id}/status
// Только пользователи с ролью "support" или "admin" имеют доступ.
func (s *Server) CheckRole(ctx context.Context, req *authv1.CheckRoleRequest) (*authv1.CheckRoleResponse, error) {
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetRequiredRole() == "" {
		return nil, status.Error(codes.InvalidArgument, "required_role is required")
	}

	role, err := s.userProvider.GetUserRole(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	// admin имеет доступ ко всему
	allowed := role == req.GetRequiredRole() || role == "admin"

	return &authv1.CheckRoleResponse{
		Allowed:    allowed,
		ActualRole: role,
	}, nil
}

// userToProto конвертирует доменную модель User в protobuf UserResponse.
// Это нужно потому что gRPC работает с protobuf-структурами,
// а не с нашими Go-структурами.
func userToProto(u models.User) *authv1.UserResponse {
	return &authv1.UserResponse{
		Id:         u.ID,
		Email:      u.Email,
		FirstName:  u.Name,
		AvatarPath: u.AvatarPath,
		Rating:     u.Rating,
		Role:       u.Role,
	}
}
