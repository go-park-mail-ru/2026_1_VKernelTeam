package validator

import (
	"errors"
	"regexp"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
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
	ErrAdTitleEmpty              = errors.New("title cannot be empty")
	ErrAdTitleTooShort           = errors.New("title must be at least 5 characters long")
	ErrAdTitleTooLong            = errors.New("title must be at most 150 characters long")
	ErrAdDescriptionEmpty        = errors.New("description cannot be empty")
	ErrAdDescriptionTooShort     = errors.New("description must be at least 10 characters long")
	ErrAdDescriptionTooLong      = errors.New("description must be at most 5000 characters long")
	ErrAdPriceNegative           = errors.New("price cannot be negative")
	ErrCategoryIDInvalid         = errors.New("category ID must be a positive integer")
	ErrUserIDInvalid             = errors.New("user ID must be a positive integer")
	ErrProductIDInvalid          = errors.New("product ID must be a positive integer")
	ErrAdStatusInvalid           = errors.New("invalid ad status")
	ErrAdLocationEmpty           = errors.New("location cannot be empty")
	ErrAdLocationTooShort        = errors.New("location must be at least 2 characters long")
	ErrAdLocationTooLong         = errors.New("location must be at most 100 characters long")
)

var (

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

	allowedAdStatuses = map[string]bool{
		models.AdStatusDraft:    true,
		models.AdStatusActive:   true,
		models.AdStatusReserved: true,
		models.AdStatusSold:     true,
		models.AdStatusArchived: true,
	}
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

//    title text NOT NULL CHECK (length(title) BETWEEN 5 AND 150),
//    description text NOT NULL CHECK (length(description) BETWEEN 10 AND 5000),

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

func ValidateUserID(userID int64) error {
	if userID <= 0 {
		return ErrUserIDInvalid
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

// ValidateCreateAdRequest консолидирует валидацию для запроса на создание объявления
func ValidateCreateAdRequest(req *dto.CreateAdRequest) *dto.ValidationErrors {
	errs := dto.ValidationErrors{}

	if err := ValidateCategoryID(req.CategoryID); err != nil {
		errs.CategoryID = err.Error()
	}
	if err := ValidateAdTitle(req.Title); err != nil {
		errs.Title = err.Error()
	}
	if err := ValidateAdDescription(req.Description); err != nil {
		errs.Description = err.Error()
	}
	if err := ValidateAdPrice(req.Price); err != nil {
		errs.Price = err.Error()
	}
	if err := ValidateAdStatus(req.Status); err != nil {
		errs.Status = err.Error()
	}
	if err := ValidateAdLocation(req.Location); err != nil {
		errs.Location = err.Error()
	}

	return &errs
}

// ValidateUpdateAdRequest консолидирует валидацию для запроса на обновление объявления
func ValidateUpdateAdRequest(req *dto.UpdateAdRequest) *dto.ValidationErrors {
	errs := dto.ValidationErrors{}

	if req.CategoryID != 0 {
		if err := ValidateCategoryID(req.CategoryID); err != nil {
			errs.CategoryID = err.Error()
		}
	}
	if req.Title != "" {
		if err := ValidateAdTitle(req.Title); err != nil {
			errs.Title = err.Error()
		}
	}
	if req.Description != "" {
		if err := ValidateAdDescription(req.Description); err != nil {
			errs.Description = err.Error()
		}
	}
	if req.Price != 0 || req.Price < 0 {
		if err := ValidateAdPrice(req.Price); err != nil {
			errs.Price = err.Error()
		}
	}
	if req.Status != "" {
		if err := ValidateAdStatus(req.Status); err != nil {
			errs.Status = err.Error()
		}
	}
	if req.Location != "" {
		if err := ValidateAdLocation(req.Location); err != nil {
			errs.Location = err.Error()
		}
	}

	return &errs
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
