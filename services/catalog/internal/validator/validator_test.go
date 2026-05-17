package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	validName   = "valid"
	validMinLen = "valid min length"
	validMaxLen = "valid max length"
	emptyName   = "empty"
	tooShort    = "too short"
	tooLong     = "too long"
	redValue    = "red"
	colorField  = "Цвет"
)

func TestValidateAdTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{validName, "iPhone 14 Pro", nil},
		{"valid cyrillic", "Айфон 14 Про", nil},
		{validMinLen, "12345", nil},
		{validMaxLen, strings.Repeat("a", 150), nil},
		{emptyName, "", ErrAdTitleEmpty},
		{tooShort, "abcd", ErrAdTitleTooShort},
		{tooLong, strings.Repeat("a", 151), ErrAdTitleTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdTitle(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAdDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{validName, "Хорошее объявление с описанием", nil},
		{validMinLen, "1234567890", nil},
		{validMaxLen, strings.Repeat("a", 5000), nil},
		{emptyName, "", ErrAdDescriptionEmpty},
		{tooShort, "short", ErrAdDescriptionTooShort},
		{tooLong, strings.Repeat("a", 5001), ErrAdDescriptionTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdDescription(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAdPrice(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr error
	}{
		{"positive", 1000, nil},
		{"zero allowed", 0, nil},
		{"max int64", 1<<62 - 1, nil},
		{"negative", -1, ErrAdPriceNegative},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdPrice(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategoryID(t *testing.T) {
	assert.NoError(t, ValidateCategoryID(1))
	assert.NoError(t, ValidateCategoryID(1_000_000))
	assert.ErrorIs(t, ValidateCategoryID(0), ErrCategoryIDInvalid)
	assert.ErrorIs(t, ValidateCategoryID(-1), ErrCategoryIDInvalid)
}

func TestValidateProductID(t *testing.T) {
	assert.NoError(t, ValidateProductID(1))
	assert.NoError(t, ValidateProductID(42))
	assert.ErrorIs(t, ValidateProductID(0), ErrProductIDInvalid)
	assert.ErrorIs(t, ValidateProductID(-7), ErrProductIDInvalid)
}

func TestValidateAdStatus(t *testing.T) {
	for _, status := range []string{"draft", "active", "reserved", "sold", "archived"} {
		t.Run("valid_"+status, func(t *testing.T) {
			assert.NoError(t, ValidateAdStatus(status))
		})
	}

	for _, status := range []string{"", "DRAFT", "unknown", "active "} {
		t.Run("invalid_"+status, func(t *testing.T) {
			assert.ErrorIs(t, ValidateAdStatus(status), ErrAdStatusInvalid)
		})
	}
}

func TestValidateAdLocation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{validName, "Москва", nil},
		{validMinLen, "ab", nil},
		{validMaxLen, strings.Repeat("a", 100), nil},
		{emptyName, "", ErrAdLocationEmpty},
		{tooShort, "a", ErrAdLocationTooShort},
		{tooLong, strings.Repeat("a", 101), ErrAdLocationTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdLocation(tt.input)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCharacteristics(t *testing.T) {
	defs := []CategoryCharacteristicDef{
		{ID: 1, AllowedValues: []string{redValue, "green", "blue"}},
		{ID: 2, AllowedValues: nil}, // free-form
	}

	t.Run("all valid enum + free-form", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 1, Value: redValue},
			{CategoryCharacteristicID: 2, Value: "any text"},
		}, defs)
		assert.NoError(t, err)
	})

	t.Run("empty value allowed (delete)", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 1, Value: ""},
		}, defs)
		assert.NoError(t, err)
	})

	t.Run("unknown id rejected", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 99, Value: redValue},
		}, defs)
		assert.ErrorIs(t, err, ErrCharacteristicIDInvalid)
	})

	t.Run("value not in enum rejected", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 1, Value: "purple"},
		}, defs)
		assert.ErrorIs(t, err, ErrCharacteristicValueNotInEnum)
	})

	t.Run("value too long rejected", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 2, Value: strings.Repeat("a", 501)},
		}, defs)
		assert.ErrorIs(t, err, ErrCharacteristicValueTooLong)
	})

	t.Run("empty inputs no defs ok", func(t *testing.T) {
		assert.NoError(t, ValidateCharacteristics(nil, nil))
	})
}

func TestValidateCustomCharacteristics(t *testing.T) {
	t.Run(validName, func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: colorField, Value: "Красный"},
			{Name: "Материал", Value: "Хлопок"},
		})
		assert.NoError(t, err)
	})

	t.Run("empty list ok", func(t *testing.T) {
		assert.NoError(t, ValidateCustomCharacteristics(nil))
	})

	t.Run("too many", func(t *testing.T) {
		inputs := make([]CustomCharacteristicInput, 11)
		for i := range inputs {
			inputs[i] = CustomCharacteristicInput{Name: string(rune('a' + i)), Value: "v"}
		}
		err := ValidateCustomCharacteristics(inputs)
		assert.ErrorIs(t, err, ErrCustomCharacteristicsTooMany)
	})

	t.Run("empty name", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: "", Value: "x"},
		})
		assert.ErrorIs(t, err, ErrCustomCharNameEmpty)
	})

	t.Run("name too long", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: strings.Repeat("a", 101), Value: "x"},
		})
		assert.ErrorIs(t, err, ErrCustomCharNameTooLong)
	})

	t.Run("duplicate names", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: colorField, Value: "красный"},
			{Name: colorField, Value: "синий"},
		})
		assert.ErrorIs(t, err, ErrCustomCharNameDuplicate)
	})

	t.Run("value too long", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: "Описание", Value: strings.Repeat("a", 501)},
		})
		assert.ErrorIs(t, err, ErrCustomCharValueTooLong)
	})

	t.Run("empty value allowed", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: colorField, Value: ""},
		})
		assert.NoError(t, err)
	})
}
