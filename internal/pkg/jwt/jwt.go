// Package jwt создан для генерации и проверки JWT-токенов.
package jwt

import (
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
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

// ParseToken парсит и валидирует JWT-токен, возвращачя объект токена или ошибку
func ParseToken(tokenString string, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return token, nil
}
