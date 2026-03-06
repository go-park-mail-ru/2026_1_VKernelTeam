// Package jwt создан для генерации и проверки JWT-токенов.
package jwt

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// NewToken формирует новый JWT-токен для указанного пользователя
// с заданным временем жизни. В качестве payload может быть
// использована информация о user. Возвращает строковое представление
// токена или ошибку.
func NewToken(user models.User, duration time.Duration, secret string) (string, error) {
	claims := jwt.MapClaims{
		"uid": user.ID,
		"exp": time.Now().Add(duration).Unix(), // срок годности
		"jti": uuid.New().String(),             // уникальный id
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret)) // подписываем токен секретным ключом
}
