// Package testreset реализует use-case очистки тестового пользователя.
//
// Доступ разрешён только аккаунтам, чей email начинается с TestEmailPrefix.
// Сам аккаунт (id, email, пароль, имя) не удаляется — сбрасываются лишь
// аватар и рейтинг, остальные связанные данные удаляются репозиторием.
package testreset

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/domain/models"
)

// TestEmailPrefix — обязательный префикс email тестового пользователя.
// Юзер с email, не начинающимся с этой строки, получит ErrNotTestUser.
const TestEmailPrefix = "clover-tester"

// ErrNotTestUser возвращается, если у пользователя нет тестового префикса email.
var ErrNotTestUser = errors.New("not a test user")

// UserEmailLookup читает пользователя для проверки префикса email.
type UserEmailLookup interface {
	UserByID(ctx context.Context, userID int64) (models.User, error)
}

// Wiper удаляет все связанные с пользователем данные в БД.
type Wiper interface {
	WipeUserData(ctx context.Context, userID int64) error
}

// UseCase оркеструет проверку доступа и саму очистку.
type UseCase struct {
	log   *slog.Logger
	users UserEmailLookup
	wiper Wiper
}

// New создаёт use-case очистки.
func New(log *slog.Logger, users UserEmailLookup, wiper Wiper) *UseCase {
	return &UseCase{log: log, users: users, wiper: wiper}
}

// Reset чистит данные пользователя, если он тестовый. Иначе — ErrNotTestUser.
func (uc *UseCase) Reset(ctx context.Context, userID int64) error {
	user, err := uc.users.UserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(strings.ToLower(user.Email), TestEmailPrefix) {
		uc.log.WarnContext(ctx, "reset attempt by non-test user",
			slog.Int64("user_id", userID),
			slog.String("email", user.Email),
		)
		return ErrNotTestUser
	}

	return uc.wiper.WipeUserData(ctx, userID)
}
