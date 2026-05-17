package dto

import "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"

// CharacteristicInput описывает входные данные категорийной характеристики.
type CharacteristicInput struct {
	CategoryCharacteristicID int64  `json:"category_characteristic_id"`
	Value                    string `json:"value"`
}

// CustomCharacteristicInput описывает входные данные пользовательской характеристики.
type CustomCharacteristicInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreateAdRequest — структура для создания объявления.
type CreateAdRequest struct {
	UserID                  int64                       `json:"-"`
	CategoryID              int64                       `json:"category_id"`
	Title                   string                      `json:"title"`
	Description             string                      `json:"description"`
	Price                   int64                       `json:"price"`
	Status                  string                      `json:"status"`
	Photos                  []string                    `json:"-"`
	Location                string                      `json:"location"`
	CategoryCharacteristics []CharacteristicInput       `json:"category_characteristics"`
	CustomCharacteristics   []CustomCharacteristicInput `json:"custom_characteristics"`
}

// UpdateAdRequest — структура для обновления объявления.
type UpdateAdRequest struct {
	ID                      int64                       `json:"-"`
	UserID                  int64                       `json:"-"`
	CategoryID              *int64                      `json:"category_id,omitempty"`
	Title                   *string                     `json:"title,omitempty"`
	Description             *string                     `json:"description,omitempty"`
	Price                   *int64                      `json:"price,omitempty"`
	Status                  *string                     `json:"status,omitempty"`
	Location                *string                     `json:"location,omitempty"`
	Photos                  []string                    `json:"-"`
	CategoryCharacteristics []CharacteristicInput       `json:"category_characteristics"`
	CustomCharacteristics   []CustomCharacteristicInput `json:"custom_characteristics"`
}

// FavoriteRequest — структура для добавления объявления в избранное.
type FavoriteRequest struct {
	AdID int64 `json:"ad_id"`
}

// PriceHistoryResponse — ответ ручки истории цен.
type PriceHistoryResponse struct {
	History []models.PricePoint `json:"history"`
}
