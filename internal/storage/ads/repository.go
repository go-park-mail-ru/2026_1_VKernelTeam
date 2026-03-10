package ads

import (
	"sync"
	"time"

	models "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// AdsRepository отвечает за доступ к данным
type AdsRepository struct {
	mu   sync.RWMutex
	Data []models.Ad // список объявлений
}

func NewAdsRepository() *AdsRepository {
	return &AdsRepository{
		Data: generateMockAds(),
	}
}

// generateMockAds возвращает ровно карточки с заполненными основными полями
func generateMockAds() []models.Ad {
	now := time.Now()

	return []models.Ad{
		{
			ID:             1,
			SellerID:       101,
			CategoryID:     1,
			Title:          "MacBook Pro 16 M1 Max",
			Description:    "Отличное состояние, полный комплект, использовался только для программирования. Батарея 95%.",
			Price:          250000,
			Status:         "active",
			Location:       "Москва",
			Photos:         []string{"/static/img/1.png"},
			ViewsCount:     150,
			FavoritesCount: 15,
			CreatedAt:      now.Add(-24 * time.Hour),
		},
		{
			ID:             2,
			SellerID:       102,
			CategoryID:     2,
			Title:          "Диван-кровать Икеа",
			Description:    "Хорошее состояние, есть небольшие пятна на обивке. Только самовывоз.",
			Price:          50000,
			Status:         "active",
			Location:       "Санкт-Петербург",
			Photos:         []string{"/static/img/2.png"},
			ViewsCount:     45,
			FavoritesCount: 3,
			CreatedAt:      now.Add(-48 * time.Hour),
		},
		{
			ID:             3,
			SellerID:       103,
			CategoryID:     3,
			Title:          "Горный велосипед Stern",
			Description:    "Почти новый, катались всего пару раз. Рама 20 дюймов, колеса 27.5.",
			Price:          55000,
			Status:         "active",
			Location:       "Казань",
			Photos:         []string{"/static/img/3.png"},
			ViewsCount:     12,
			FavoritesCount: 1,
			CreatedAt:      now.Add(-72 * time.Hour),
		},
		{
			ID:             4,
			SellerID:       104,
			CategoryID:     4,
			Title:          "Комплект книг Гарри Поттер",
			Description:    "Полное собрание от издательства РОСМЭН. Идеальное состояние, как новые.",
			Price:          3500,
			Status:         "active",
			Location:       "Новосибирск",
			Photos:         []string{"/static/img/4.png"},
			ViewsCount:     300,
			FavoritesCount: 42,
			CreatedAt:      now.Add(-2 * time.Hour),
		},
	}
}

// метод для получения данных (чтобы не обращаться к полю data напрямую)
func (r *AdsRepository) GetAll() []models.Ad {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// создаем новый слайс и копируем туда данные
	result := make([]models.Ad, len(r.Data))
	copy(result, r.Data)
	return result
}
