package validator

import (
	"fmt"
	"strings"
	"testing"

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
		{"Active status", "active", nil},
		{"Draft status", "draft", nil},
		{"Sold status", "sold", nil},
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



func TestValidateAdLocation(t *testing.T) {
	assert.NoError(t, ValidateAdLocation("Москва"))
	assert.ErrorIs(t, ValidateAdLocation(""), ErrAdLocationEmpty)
	assert.ErrorIs(t, ValidateAdLocation("A"), ErrAdLocationTooShort)
	assert.ErrorIs(t, ValidateAdLocation(strings.Repeat("A", 101)), ErrAdLocationTooLong)
}

func TestValidateCharacteristics(t *testing.T) {
	defs := []CategoryCharacteristicDef{
		{ID: 1, AllowedValues: []string{"Красный", "Синий", "Зелёный"}},
		{ID: 2, AllowedValues: nil}, // свободный ввод
	}

	tests := []struct {
		name    string
		inputs  []CharacteristicInput
		wantErr error
	}{
		{
			name: "Valid input with enum value",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Красный"},
			},
			wantErr: nil,
		},
		{
			name: "Valid input with free text",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 2, Value: "XL"},
			},
			wantErr: nil,
		},
		{
			name: "Invalid category_characteristic_id",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 999, Value: "test"},
			},
			wantErr: ErrCharacteristicIDInvalid,
		},
		{
			name: "Value not in allowed enum",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Жёлтый"},
			},
			wantErr: ErrCharacteristicValueNotInEnum,
		},
		{
			name: "Value too long",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 2, Value: strings.Repeat("A", 501)},
			},
			wantErr: ErrCharacteristicValueTooLong,
		},
		{
			name: "Empty value is allowed (means deletion)",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: ""},
			},
			wantErr: nil,
		},
		{
			name:    "Empty inputs is valid",
			inputs:  []CharacteristicInput{},
			wantErr: nil,
		},
		{
			name: "Multiple valid inputs",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 1, Value: "Синий"},
				{CategoryCharacteristicID: 2, Value: "M"},
			},
			wantErr: nil,
		},
		{
			name: "Value exactly 500 chars is valid",
			inputs: []CharacteristicInput{
				{CategoryCharacteristicID: 2, Value: strings.Repeat("Б", 500)},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCharacteristics(tt.inputs, defs)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestValidateCustomCharacteristics(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []CustomCharacteristicInput
		wantErr error
	}{
		{
			name: "Valid input",
			inputs: []CustomCharacteristicInput{
				{Name: "Материал", Value: "Дерево"},
			},
			wantErr: nil,
		},
		{
			name:    "Empty inputs is valid",
			inputs:  []CustomCharacteristicInput{},
			wantErr: nil,
		},
		{
			name: "More than 10 items",
			inputs: func() []CustomCharacteristicInput {
				items := make([]CustomCharacteristicInput, 11)
				for i := range items {
					items[i] = CustomCharacteristicInput{
						Name:  fmt.Sprintf("Характеристика_%d", i),
						Value: "val",
					}
				}
				return items
			}(),
			wantErr: ErrCustomCharacteristicsTooMany,
		},
		{
			name: "Empty name",
			inputs: []CustomCharacteristicInput{
				{Name: "", Value: "value"},
			},
			wantErr: ErrCustomCharNameEmpty,
		},
		{
			name: "Name too long",
			inputs: []CustomCharacteristicInput{
				{Name: strings.Repeat("A", 101), Value: "value"},
			},
			wantErr: ErrCustomCharNameTooLong,
		},
		{
			name: "Duplicate names",
			inputs: []CustomCharacteristicInput{
				{Name: "Цвет", Value: "Красный"},
				{Name: "Цвет", Value: "Синий"},
			},
			wantErr: ErrCustomCharNameDuplicate,
		},
		{
			name: "Value too long",
			inputs: []CustomCharacteristicInput{
				{Name: "Поле", Value: strings.Repeat("B", 501)},
			},
			wantErr: ErrCustomCharValueTooLong,
		},
		{
			name: "Empty value is allowed (means deletion)",
			inputs: []CustomCharacteristicInput{
				{Name: "Материал", Value: ""},
			},
			wantErr: nil,
		},
		{
			name: "Exactly 10 items is valid",
			inputs: func() []CustomCharacteristicInput {
				items := make([]CustomCharacteristicInput, 10)
				for i := range items {
					items[i] = CustomCharacteristicInput{
						Name:  fmt.Sprintf("Характеристика_%d", i),
						Value: "val",
					}
				}
				return items
			}(),
			wantErr: nil,
		},
		{
			name: "Name exactly 100 chars is valid",
			inputs: []CustomCharacteristicInput{
				{Name: strings.Repeat("A", 100), Value: "val"},
			},
			wantErr: nil,
		},
		{
			name: "Value exactly 500 chars is valid",
			inputs: []CustomCharacteristicInput{
				{Name: "Поле", Value: strings.Repeat("B", 500)},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomCharacteristics(tt.inputs)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
