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

// Sentinel-ошибки валидации полей объявления и характеристик.
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
	ErrAdCoordsHalfMissing   = errors.New("lat and lon must be provided together")
	ErrAdLatOutOfRange       = errors.New("lat must be between -90 and 90")
	ErrAdLonOutOfRange       = errors.New("lon must be between -180 and 180")

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

// ValidateAdTitle проверяет заголовок объявления на непустоту и длину 5-150 символов.
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

// ValidateAdDescription проверяет описание объявления на непустоту и длину 10-5000 символов.
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

// ValidateAdPrice проверяет, что цена объявления не отрицательная.
func ValidateAdPrice(price int64) error {
	if price < 0 {
		return ErrAdPriceNegative
	}
	return nil
}

// ValidateCategoryID проверяет, что идентификатор категории положителен.
func ValidateCategoryID(categoryID int64) error {
	if categoryID <= 0 {
		return ErrCategoryIDInvalid
	}
	return nil
}

// ValidateProductID проверяет, что идентификатор товара положителен.
func ValidateProductID(productID int64) error {
	if productID <= 0 {
		return ErrProductIDInvalid
	}
	return nil
}

// ValidateAdStatus проверяет, что статус входит в список допустимых.
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

// ValidateAdLocation проверяет, что строка адреса задана и её длина в пределах 2-100 символов.
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

// ValidateAdCoords проверяет координаты адреса.
// Оба значения опциональны, но передаваться должны парой: либо оба nil, либо оба заданы.
func ValidateAdCoords(lat, lon *float64) error {
	if lat == nil && lon == nil {
		return nil
	}
	if lat == nil || lon == nil {
		return ErrAdCoordsHalfMissing
	}
	if *lat < -90 || *lat > 90 {
		return ErrAdLatOutOfRange
	}
	if *lon < -180 || *lon > 180 {
		return ErrAdLonOutOfRange
	}
	return nil
}
