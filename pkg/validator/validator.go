package validator

import (
	"errors"
	"regexp"
)

// ошибки валидации
var (
	ErrInvalidEmailFormat        = errors.New("invalid email format")
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters long")
	ErrPasswordRequiresDigit     = errors.New("password must contain at least one digit")
	ErrPasswordRequiresLetter    = errors.New("password must contain at least one latin letter")
	ErrPasswordContainsForbidden = errors.New("password contains forbidden characters")
	ErrNameEmpty                 = errors.New("name cannot be empty")
	ErrNameInvalid               = errors.New("name contains invalid characters")

	// Только латиница, цифры и _
	reStrict = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	// Проверка наличия хотя бы одной буквы
	reHasLetter = regexp.MustCompile(`[a-zA-Z]`)
	// Проверка наличия хотя бы одной цифры
	reHasDigit = regexp.MustCompile(`[0-9]`)
	// регулярное выражение для проверки email
	emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	// регулярное выражение для имени: буквы (латиница и кириллица), пробелы, апострофы, дефисы
	nameRegex = regexp.MustCompile(`^[\p{L}\s'-]+$`)
)

// ValidateEmail проверяет валидность email
func ValidateEmail(email string) error {
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

	if !reStrict.MatchString(password) {
		return ErrPasswordContainsForbidden
	}

	if !reHasLetter.MatchString(password) {
		return ErrPasswordRequiresLetter
	}

	if !reHasDigit.MatchString(password) {
		return ErrPasswordRequiresDigit
	}

	return nil
}

// ValidateName проверяет валидность имени: не пустое и содержит только буквы (латиница и кириллица), пробелы, апострофы, дефисы
func ValidateName(name string) error {
	if name == "" {
		return ErrNameEmpty
	}
	if !nameRegex.MatchString(name) {
		return ErrNameInvalid
	}
	return nil
}
