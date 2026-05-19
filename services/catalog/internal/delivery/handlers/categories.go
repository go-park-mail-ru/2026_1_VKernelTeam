package handlers

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/responser"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/domain/models"
)

var hardcodedCategories = []models.Category{
	{ID: 1, Name: "Электроника"},
	{ID: 2, Name: "Недвижимость"},
	{ID: 3, Name: "Транспорт"},
	{ID: 4, Name: "Хобби и отдых"},
	{ID: 5, Name: "Музыка"},
	{ID: 6, Name: "Ремонт"},
	{ID: 7, Name: "Туризм"},
	{ID: 8, Name: "Техника для дома"},
	{ID: 9, Name: "Игрушки"},
	{ID: 10, Name: "Настольные игры"},
	{ID: 11, Name: "Одежда"},
	{ID: 12, Name: "Обувь"},
	{ID: 13, Name: "Аксессуары"},
	{ID: 14, Name: "Книги"},
	{ID: 15, Name: "Красота и здоровье"},
	{ID: 16, Name: "Животные"},
	{ID: 17, Name: "Для дома и дачи"},
	{ID: 18, Name: "Запчасти"},
	{ID: 19, Name: "Спорт"},
	{ID: 20, Name: "Канцелярия"},
	{ID: 21, Name: "Авто"},
	{ID: 22, Name: "Работа"},
	{ID: 23, Name: "Товары для детей"},
}

// HandleGetCategories возвращает плоский список категорий верхнего уровня.
// @Summary Получить список категорий
// @Tags categories
// @Produce json
// @Success 200 {array} models.Category
// @Router /categories [get]
func (h *AdsHandlers) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	responser.RespondWithJSON(w, http.StatusOK, hardcodedCategories)
}
