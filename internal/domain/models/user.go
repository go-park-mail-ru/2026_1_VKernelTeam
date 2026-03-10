// Package models содержит структуры данных, используемые в домене
// приложения.
package models

import "time"

// User представляет зарегистрированного пользователя системы. Поле
// PassHash содержит хэш пароля, IsAdmin указывает на административные права.
type User struct {
	ID        int64
	Name      string
	Email     string
	PassHash  []byte
	IsAdmin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
