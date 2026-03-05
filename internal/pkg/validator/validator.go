package validator

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// ошибки валидации
var (
	ErrInvalidEmailFormat     = errors.New("invalid email format")
	ErrPasswordTooShort       = errors.New("password must be at least 8 characters long")
	ErrPasswordRequiresDigit  = errors.New("password must contain at least one digit")
	ErrPasswordRequiresLetter = errors.New("password must contain at least one latin letter")
)

// регулярное выражение для проверки email
var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

// ValidateEmail проверяет валидность email
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmailFormat
	}
	return nil
}

// ValidatePassword проверяет пароль на соответствие требованиям:
// - минимум 8 символов
// - содержит хотя бы одну латинскую букву
// - содержит хотя бы одну цифру
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasLetter, hasDigit bool
	for _, char := range password {
		if unicode.IsLetter(char) && char <= unicode.MaxASCII {
			hasLetter = true
		} else if unicode.IsDigit(char) {
			hasDigit = true
		}
	}

	if !hasLetter {
		return ErrPasswordRequiresLetter
	}
	if !hasDigit {
		return ErrPasswordRequiresDigit
	}

	return nil
}
