package validator

import (
	"errors"
	"slices"
	"unicode/utf8"
)

// CharacteristicInput описывает входное значение категорийной характеристики.
// Локальный тип, чтобы validator не зависел от domain-моделей сервиса.
type CharacteristicInput struct {
	CategoryCharacteristicID int64
	Value                    string
}

// CustomCharacteristicInput описывает входную пользовательскую характеристику.
type CustomCharacteristicInput struct {
	Name  string
	Value string
}

// CategoryCharacteristicDef описывает определение категорийной характеристики
// (для проверки allowed_values при валидации).
type CategoryCharacteristicDef struct {
	ID            int64
	AllowedValues []string
}

var (
	ErrAdTitleEmpty          = errors.New("title cannot be empty")
	ErrAdTitleTooShort       = errors.New("title must be at least 5 characters long")
	ErrAdTitleTooLong        = errors.New("title must be at most 150 characters long")
	ErrAdDescriptionEmpty    = errors.New("description cannot be empty")
	ErrAdDescriptionTooShort = errors.New("description must be at least 10 characters long")
	ErrAdDescriptionTooLong  = errors.New("description must be at most 5000 characters long")
	ErrAdPriceNegative       = errors.New("price cannot be negative")
	ErrCategoryIDInvalid     = errors.New("category ID must be a positive integer")
	ErrProductIDInvalid      = errors.New("product ID must be a positive integer")
	ErrAdStatusInvalid       = errors.New("invalid ad status")
	ErrAdLocationEmpty       = errors.New("location cannot be empty")
	ErrAdLocationTooShort    = errors.New("location must be at least 2 characters long")
	ErrAdLocationTooLong     = errors.New("location must be at most 100 characters long")

	ErrCharacteristicIDInvalid      = errors.New("category_characteristic_id not found in category definitions")
	ErrCharacteristicValueTooLong   = errors.New("characteristic value must be at most 500 characters")
	ErrCharacteristicValueEmpty     = errors.New("characteristic value cannot be empty")
	ErrCharacteristicValueNotInEnum = errors.New("characteristic value is not in allowed values")
	ErrCustomCharacteristicsTooMany = errors.New("custom characteristics limit is 10")
	ErrCustomCharNameEmpty          = errors.New("custom characteristic name cannot be empty")
	ErrCustomCharNameTooLong        = errors.New("custom characteristic name must be at most 100 characters")
	ErrCustomCharNameDuplicate      = errors.New("custom characteristic names must be unique")
	ErrCustomCharValueTooLong       = errors.New("custom characteristic value must be at most 500 characters")

	allowedAdStatuses = map[string]bool{
		"draft":    true,
		"active":   true,
		"reserved": true,
		"sold":     true,
		"archived": true,
	}
)

func ValidateAdTitle(title string) error {
	if title == "" {
		return ErrAdTitleEmpty
	}
	titleLen := len([]rune(title))
	if titleLen < 5 {
		return ErrAdTitleTooShort
	}
	if titleLen > 150 {
		return ErrAdTitleTooLong
	}
	return nil
}

func ValidateAdDescription(description string) error {
	if description == "" {
		return ErrAdDescriptionEmpty
	}
	descLen := len([]rune(description))
	if descLen < 10 {
		return ErrAdDescriptionTooShort
	}
	if descLen > 5000 {
		return ErrAdDescriptionTooLong
	}
	return nil
}

func ValidateAdPrice(price int64) error {
	if price < 0 {
		return ErrAdPriceNegative
	}
	return nil
}

func ValidateCategoryID(categoryID int64) error {
	if categoryID <= 0 {
		return ErrCategoryIDInvalid
	}
	return nil
}

func ValidateProductID(productID int64) error {
	if productID <= 0 {
		return ErrProductIDInvalid
	}
	return nil
}

func ValidateAdStatus(status string) error {
	if !allowedAdStatuses[status] {
		return ErrAdStatusInvalid
	}
	return nil
}

// ValidateCharacteristics проверяет категорийные характеристики:
// - category_characteristic_id существует в определениях категории;
// - если allowed_values задан, value должен входить в список;
// - value: 1-500 символов.
func ValidateCharacteristics(inputs []CharacteristicInput, defs []CategoryCharacteristicDef) error {
	defMap := make(map[int64]CategoryCharacteristicDef, len(defs))
	for _, d := range defs {
		defMap[d.ID] = d
	}

	for _, inp := range inputs {
		def, ok := defMap[inp.CategoryCharacteristicID]
		if !ok {
			return ErrCharacteristicIDInvalid
		}

		// Пустое значение допустимо (означает удаление), пропускаем остальные проверки.
		if inp.Value == "" {
			continue
		}

		valLen := utf8.RuneCountInString(inp.Value)
		if valLen > 500 {
			return ErrCharacteristicValueTooLong
		}

		if def.AllowedValues != nil {
			if !slices.Contains(def.AllowedValues, inp.Value) {
				return ErrCharacteristicValueNotInEnum
			}
		}
	}

	return nil
}

// ValidateCustomCharacteristics проверяет пользовательские характеристики:
// - не более 10 штук;
// - name: 1-100 символов, уникальные;
// - value: 0-500 символов (пустая строка допустима — означает удаление).
func ValidateCustomCharacteristics(inputs []CustomCharacteristicInput) error {
	if len(inputs) > 10 {
		return ErrCustomCharacteristicsTooMany
	}

	seen := make(map[string]struct{}, len(inputs))
	for _, inp := range inputs {
		nameLen := utf8.RuneCountInString(inp.Name)
		if nameLen == 0 {
			return ErrCustomCharNameEmpty
		}
		if nameLen > 100 {
			return ErrCustomCharNameTooLong
		}

		if _, exists := seen[inp.Name]; exists {
			return ErrCustomCharNameDuplicate
		}
		seen[inp.Name] = struct{}{}

		valLen := utf8.RuneCountInString(inp.Value)
		if valLen > 500 {
			return ErrCustomCharValueTooLong
		}
	}

	return nil
}

func ValidateAdLocation(location string) error {
	if location == "" {
		return ErrAdLocationEmpty
	}
	locLen := len([]rune(location))
	if locLen < 2 {
		return ErrAdLocationTooShort
	}
	if locLen > 100 {
		return ErrAdLocationTooLong
	}
	return nil
}
