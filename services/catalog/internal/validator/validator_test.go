package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAdTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid", "iPhone 14 Pro", nil},
		{"valid cyrillic", "Айфон 14 Про", nil},
		{"valid min length", "12345", nil},
		{"valid max length", strings.Repeat("a", 150), nil},
		{"empty", "", ErrAdTitleEmpty},
		{"too short", "abcd", ErrAdTitleTooShort},
		{"too long", strings.Repeat("a", 151), ErrAdTitleTooLong},
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
		{"valid", "Хорошее объявление с описанием", nil},
		{"valid min length", "1234567890", nil},
		{"valid max length", strings.Repeat("a", 5000), nil},
		{"empty", "", ErrAdDescriptionEmpty},
		{"too short", "short", ErrAdDescriptionTooShort},
		{"too long", strings.Repeat("a", 5001), ErrAdDescriptionTooLong},
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
		{"valid", "Москва", nil},
		{"valid min length", "ab", nil},
		{"valid max length", strings.Repeat("a", 100), nil},
		{"empty", "", ErrAdLocationEmpty},
		{"too short", "a", ErrAdLocationTooShort},
		{"too long", strings.Repeat("a", 101), ErrAdLocationTooLong},
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
		{ID: 1, AllowedValues: []string{"red", "green", "blue"}},
		{ID: 2, AllowedValues: nil}, // free-form
	}

	t.Run("all valid enum + free-form", func(t *testing.T) {
		err := ValidateCharacteristics([]CharacteristicInput{
			{CategoryCharacteristicID: 1, Value: "red"},
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
			{CategoryCharacteristicID: 99, Value: "red"},
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
	t.Run("valid", func(t *testing.T) {
		err := ValidateCustomCharacteristics([]CustomCharacteristicInput{
			{Name: "Цвет", Value: "Красный"},
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
			{Name: "Цвет", Value: "красный"},
			{Name: "Цвет", Value: "синий"},
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
			{Name: "Цвет", Value: ""},
		})
		assert.NoError(t, err)
	})
}
