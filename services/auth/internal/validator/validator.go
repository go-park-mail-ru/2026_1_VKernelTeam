package validator

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidEmailFormat        = errors.New("invalid email format")
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters long")
	ErrPasswordRequiresDigit     = errors.New("password must contain at least one digit")
	ErrPasswordRequiresLetter    = errors.New("password must contain at least one latin letter")
	ErrPasswordContainsForbidden = errors.New("password contains forbidden characters")
	ErrNameEmpty                 = errors.New("name cannot be empty")
	ErrNameTooShort              = errors.New("name must be at least 3 characters long")
	ErrNameTooLong               = errors.New("name must be no more than 50 characters long")
	ErrNameInvalid               = errors.New("name contains invalid characters")
	ErrUserIDInvalid             = errors.New("user ID must be a positive integer")

	reHasLetter = regexp.MustCompile(`[a-zA-Z]`)
	reHasDigit  = regexp.MustCompile(`[0-9]`)

	// emailRegex: разрешает цифры в TLD (например .xn--p1ai), требует минимум 2 символа в TLD.
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z0-9]{2,}$`)

	// nameRegex: буквы (латиница и кириллица), пробелы, апострофы, дефисы; длина 3-50.
	nameRegex = regexp.MustCompile(`^[\p{L}\s'-]{3,50}$`)
)

// ValidateEmail проверяет валидность email и приводит его к нижнему регистру.
func ValidateEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	if !emailRegex.MatchString(email) {
		return email, ErrInvalidEmailFormat
	}
	return email, nil
}

// ValidatePassword проверяет пароль:
// - минимум 8 символов;
// - хотя бы одна латинская буква;
// - хотя бы одна цифра.
// Спецсимволы разрешены сознательно.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return ErrPasswordTooShort
	}
	if !reHasLetter.MatchString(password) {
		return ErrPasswordRequiresLetter
	}
	if !reHasDigit.MatchString(password) {
		return ErrPasswordRequiresDigit
	}
	return nil
}

// ValidateName проверяет имя: 3-50 рун, буквы (латиница/кириллица), пробелы, апострофы, дефисы.
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return "", ErrNameEmpty
	}

	count := utf8.RuneCountInString(name)
	if count < 3 {
		return "", ErrNameTooShort
	}
	if count > 50 {
		return "", ErrNameTooLong
	}

	if !nameRegex.MatchString(name) {
		return "", ErrNameInvalid
	}
	return name, nil
}

func ValidateUserID(userID int64) error {
	if userID <= 0 {
		return ErrUserIDInvalid
	}
	return nil
}
