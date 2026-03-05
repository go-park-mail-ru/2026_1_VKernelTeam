package validator

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"Valid email", "test@example.com", nil},
		{"Valid email with dot", "test.user@example.co.uk", nil},
		{"Valid email with plus", "user+label@example.org", nil},
		{"Invalid format - no domain", "test@", ErrInvalidEmailFormat},
		{"Invalid format - no user", "@example.com", ErrInvalidEmailFormat},
		{"Invalid characters", "test!@example.com", ErrInvalidEmailFormat},
		{"Empty email", "", ErrInvalidEmailFormat},
		{"Spaces padding", "  test@example.com  ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateEmail(tt.email); err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"Valid password", "Password123", nil},
		{"Too short", "Pass12", ErrPasswordTooShort},
		{"No digits", "Password", ErrPasswordRequiresDigit},
		{"No letters", "123456789", ErrPasswordRequiresLetter},
		{"Cyrillic letters only", "Пароль123", ErrPasswordRequiresLetter},
		{"Empty password", "", ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePassword(tt.password); err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
