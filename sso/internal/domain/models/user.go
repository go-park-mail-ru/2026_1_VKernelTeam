// Пакет models содержит структуры данных, используемые в домене
// приложения.
package models

type User struct {
	ID       int64
	Email    string
	PassHash []byte
	IsAdmin  bool
}

// User представляет зарегистрированного пользователя системы. Поле
// PassHash содержит хэш пароля, IsAdmin указывает на административные права.
