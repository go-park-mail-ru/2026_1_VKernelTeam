package ads

import (
	"sync"
	"time"
)

// AdsRepository отвечает за доступ к данным
type AdsRepository struct {
	sync.RWMutex
	data []Ad // список объявлений
}

// конструктор
func NewAdsRepository() *AdsRepository {
	return &AdsRepository{
		data: []Ad{
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
func (r *AdsRepository) GetAll() []Ad {
	r.RLock()
	defer r.RUnlock()

	// создаем новый слайс и копируем туда данные
	result := make([]Ad, len(r.data))
	copy(result, r.data)
	return result
}
