package storage

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
)

// TokenRevoker описывает интерфейс для отзыва токенов
type TokenRevoker interface {
	Add(jti string, exp time.Time)
}

// UserSaver предоставляет метод сохранения пользователя в хранилище.
type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (uid int64, err error)
}

// UserProvider предоставляет методы получения данных о пользователе и
// проверки его административных прав.
type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}
