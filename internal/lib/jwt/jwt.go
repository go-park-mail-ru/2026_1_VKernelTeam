// Package jwt содержит утилиты для генерации и проверки JWT-токенов.
// В текущей реализации генерация ещё не реализована.
package jwt

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
)

// NewToken формирует новый JWT-токен для указанного пользователя и
// приложения с заданным временем жизни. В качестве payload может быть
// использована информация о user и app. Возвращает строковое представление
// токена или ошибку.
// TODO: реализовать создание токена с помощью библиотеки github.com/golang-jwt/jwt.
func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	//TODO: implement JWT token generation using a library like github.com/golang-jwt/jwt
	return "2345egfvdf", nil
}
