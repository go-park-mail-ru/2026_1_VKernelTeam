package ads

import (
	"sync"
	"time"

	models "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/domain/models"
)

// AdsRepository отвечает за доступ к данным
type AdsRepository struct {
	sync.RWMutex
	Data []models.Ad // список объявлений
}

// конструктор
func NewAdsRepository() *AdsRepository {
	return &AdsRepository{
		Data: []models.Ad{
			{
				ID:          1,
				Title:       "Продам гараж",
				Description: "Очень ухоженный",
				Price:       1_000_000,
				Photos: []string{
					"/static/img/garage_1.png",
					"/static/img/garage_2.png",
				},
				Tags:      []string{"недвижимость", "гараж"},
				SellerID:  1,
				CreatedAt: time.Now().Truncate(0),
				Views:     12,
			},
		},
	}
}

// метод для получения данных (чтобы не обращаться к полю data напрямую)
func (r *AdsRepository) GetAll() []models.Ad {
	r.RLock()
	defer r.RUnlock()

	// создаем новый слайс и копируем туда данные
	result := make([]models.Ad, len(r.Data))
	copy(result, r.Data)
	return result
}
