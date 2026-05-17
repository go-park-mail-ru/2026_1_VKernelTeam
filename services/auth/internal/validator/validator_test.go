package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testEmail = "user@example.com"
	caseEmpty = "empty"
	testName  = "John"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantClean string
		wantErr   error
	}{
		{"valid lowercase", testEmail, testEmail, nil},
		{"valid uppercase normalized", "USER@Example.COM", testEmail, nil},
		{"valid with whitespace trimmed", "  user@example.com  ", testEmail, nil},
		{"valid with subdomain", "u@mail.example.co.uk", "u@mail.example.co.uk", nil},
		{"valid with plus tag", "user+tag@example.com", "user+tag@example.com", nil},
		{"valid numeric TLD", "u@example.123", "u@example.123", nil},
		{"missing @", "userexample.com", "userexample.com", ErrInvalidEmailFormat},
		{"missing domain", "user@", "user@", ErrInvalidEmailFormat},
		{"missing local part", "@example.com", "@example.com", ErrInvalidEmailFormat},
		{"single-char TLD", "u@example.c", "u@example.c", ErrInvalidEmailFormat},
		{"spaces inside", "us er@example.com", "us er@example.com", ErrInvalidEmailFormat},
		{caseEmpty, "", "", ErrInvalidEmailFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateEmail(tt.input)
			assert.Equal(t, tt.wantClean, got)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid", "Password1", nil},
		{"valid with specials", "Pa$$w0rd!#", nil},
		{"valid cyrillic letter and digit", "пароль1abcd", nil},
		{"too short", "Pass1", ErrPasswordTooShort},
		{"no letter", "12345678", ErrPasswordRequiresLetter},
		{"no digit", "Password", ErrPasswordRequiresDigit},
		{"only cyrillic letters", "парольпароль", ErrPasswordRequiresLetter},
		{caseEmpty, "", ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantClean string
		wantErr   error
	}{
		{"valid latin", testName, testName, nil},
		{"valid cyrillic", "Иван", "Иван", nil},
		{"valid with apostrophe", "O'Neill", "O'Neill", nil},
		{"valid with hyphen", "Anna-Maria", "Anna-Maria", nil},
		{"valid with space", "Анна Мария", "Анна Мария", nil},
		{"trimmed", "   John   ", testName, nil},
		{caseEmpty, "", "", ErrNameEmpty},
		{"whitespace only treated as empty", "   ", "", ErrNameEmpty},
		{"too short", "Al", "", ErrNameTooShort},
		{"too long", strings.Repeat("a", 51), "", ErrNameTooLong},
		{"contains digit", "John2", "", ErrNameInvalid},
		{"contains symbol", "John!", "", ErrNameInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateName(tt.input)
			assert.Equal(t, tt.wantClean, got)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserID(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr error
	}{
		{"positive", 1, nil},
		{"large positive", 1_000_000, nil},
		{"zero", 0, ErrUserIDInvalid},
		{"negative", -1, ErrUserIDInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserID(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
