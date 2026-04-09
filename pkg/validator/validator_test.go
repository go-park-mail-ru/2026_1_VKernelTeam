package validator

import (
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    string
		wantErr error
	}{
		{
			name:    "Valid email",
			email:   "user@example.com",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "Valid email with spaces and caps",
			email:   "  User@EXAMPLE.com  ",
			want:    "user@example.com",
			wantErr: nil,
		},
		{
			name:    "Invalid format - no @",
			email:   "invalid-email.com",
			want:    "invalid-email.com",
			wantErr: ErrInvalidEmailFormat,
		},
		{
			name:    "Valid TLD with digits",
			email:   "test@domain.123",
			want:    "test@domain.123",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateEmail(tt.email)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr error
	}{
		{
			name:    "Valid password",
			pass:    "password123",
			wantErr: nil,
		},
		{
			name:    "Valid with special chars",
			pass:    "Admin!@#45",
			wantErr: nil,
		},
		{
			name:    "Too short",
			pass:    "short1",
			wantErr: ErrPasswordTooShort,
		},
		{
			name:    "No letters",
			pass:    "1234567890",
			wantErr: ErrPasswordRequiresLetter,
		},
		{
			name:    "No digits",
			pass:    "onlyletters",
			wantErr: ErrPasswordRequiresDigit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.pass)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:    "Valid name Cyrillic",
			input:   "Иван",
			want:    "Иван",
			wantErr: nil,
		},
		{
			name:    "Valid name with spaces",
			input:   "  Д'Артаньян-Иванов  ",
			want:    "Д'Артаньян-Иванов",
			wantErr: nil,
		},
		{
			name:    "Too short after trim",
			input:   "  Li  ",
			want:    "",
			wantErr: ErrNameTooShort,
		},
		{
			name:    "Invalid characters",
			input:   "Ivan123",
			want:    "",
			wantErr: ErrNameInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateName(tt.input)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateAdTitle(t *testing.T) {
	assert.NoError(t, ValidateAdTitle("Продам гараж"))
	assert.ErrorIs(t, ValidateAdTitle(""), ErrAdTitleEmpty)
	assert.ErrorIs(t, ValidateAdTitle("Кот"), ErrAdTitleTooShort)
}

func TestValidateAdPrice(t *testing.T) {
	assert.NoError(t, ValidateAdPrice(100))
	assert.NoError(t, ValidateAdPrice(0))
	assert.ErrorIs(t, ValidateAdPrice(-1), ErrAdPriceNegative)
}

func TestValidateCategoryID(t *testing.T) {
	assert.NoError(t, ValidateCategoryID(1))
	assert.ErrorIs(t, ValidateCategoryID(0), ErrCategoryIDInvalid)
	assert.ErrorIs(t, ValidateCategoryID(-5), ErrCategoryIDInvalid)
}

func TestValidateAdStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr error
	}{
		{"Active status", models.AdStatusActive, nil},
		{"Draft status", models.AdStatusDraft, nil},
		{"Sold status", models.AdStatusSold, nil},
		{"Invalid status", "something_else", ErrAdStatusInvalid},
		{"Empty status", "", ErrAdStatusInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdStatus(tt.status)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestValidateCreateAdRequest(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			CategoryID:  1,
			Title:       "Valid Title",
			Description: "This is a valid description",
			Price:       1000,
			Status:      models.AdStatusActive,
			Location:    "Moscow",
		}
		errs := ValidateCreateAdRequest(req)
		assert.Equal(t, dto.ValidationErrors{}, *errs)
	})

	t.Run("Multiple errors", func(t *testing.T) {
		req := &dto.CreateAdRequest{
			CategoryID:  -1,
			Title:       "123",
			Description: "",
			Price:       -100,
			Status:      "invalid",
			Location:    "",
		}
		errs := ValidateCreateAdRequest(req)

		assert.NotEmpty(t, errs.CategoryID)
		assert.NotEmpty(t, errs.Title)
		assert.NotEmpty(t, errs.Description)
		assert.NotEmpty(t, errs.Price)
		assert.NotEmpty(t, errs.Status)
		assert.NotEmpty(t, errs.Location)
	})
}

func TestValidateUpdateAdRequest(t *testing.T) {
	t.Run("Partial update - only title", func(t *testing.T) {
		title := "New valid title"
		req := &dto.UpdateAdRequest{
			Title: &title,
		}
		errs := ValidateUpdateAdRequest(req)
		assert.Equal(t, dto.ValidationErrors{}, *errs)
	})

	t.Run("Invalid partial update", func(t *testing.T) {
		price := int64(-50)
		req := &dto.UpdateAdRequest{
			Price: &price,
		}
		errs := ValidateUpdateAdRequest(req)
		assert.Equal(t, ErrAdPriceNegative.Error(), errs.Price)
		assert.Empty(t, errs.Title)
	})
}

func TestValidateAdLocation(t *testing.T) {
	assert.NoError(t, ValidateAdLocation("Москва"))
	assert.ErrorIs(t, ValidateAdLocation(""), ErrAdLocationEmpty)
	assert.ErrorIs(t, ValidateAdLocation("A"), ErrAdLocationTooShort)
	assert.ErrorIs(t, ValidateAdLocation(strings.Repeat("A", 101)), ErrAdLocationTooLong)
}
