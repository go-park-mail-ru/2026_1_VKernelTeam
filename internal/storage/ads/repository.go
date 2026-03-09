package ads

import (
	"sync"
	"time"

	models "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// AdsRepository отвечает за доступ к данным
type AdsRepository struct {
	mu sync.RWMutex
	data []models.Ad // список объявлений
}

func NewAdsRepository() *AdsRepository {
	return &AdsRepository{
		data: generateMockAds(),
	}
}

// generateMockAds returns a slice of mock ads suitable for testing and development.
// It covers various cases: normal ads, free ads, ads with many photos/tags, and edge cases.
func generateMockAds() []models.Ad {
	now := time.Now().Truncate(0)

	return []models.Ad{
		{
			ID:          1,
			Title:       "MacBook Pro 16 M1 Max",
			Description: "Отличное состояние, полный комплект, использовался для программирования.",
			Price:       250000,
			Photos:      []string{"https://example.com/macbook1.jpg", "https://example.com/macbook2.jpg"},
			Tags:        []string{"ноутбук", "apple", "macbook", "электроника"},
			SellerID:    1,
			CreatedAt:   now.Add(-24 * time.Hour),
			Views:       150,
		},
		{
			ID:          2,
			Title:       "Диван раскладной Икеа",
			Description: "Состояние хорошее, есть пара пятен. Самовывоз.",
			Price:       5000,
			Photos:      []string{"https://example.com/sofa1.jpg"},
			Tags:        []string{"мебель", "диван", "икеа"},
			SellerID:    2,
			CreatedAt:   now.Add(-48 * time.Hour),
			Views:       45,
		},
		{
			ID:          3,
			Title:       "Отдам даром котят",
			Description: "Три милых котенка ищут дом. Возраст 2 месяца, к лотку приучены.",
			Price:       0, // Free
			Photos:      []string{"https://example.com/cat1.jpg", "https://example.com/cat2.jpg", "https://example.com/cat3.jpg"},
			Tags:        []string{"животные", "котята", "даром", "в добрые руки"},
			SellerID:    3,
			CreatedAt:   now.Add(-2 * time.Hour),
			Views:       500,
		},
		{
			ID:          4,
			Title:       "Велосипед горный Stern",
			Description: "Почти новый, катался пару раз. Рама 20 дюймов.",
			Price:       15000,
			Photos:      nil, // No photos
			Tags:        []string{"спорт", "велосипед", "активный отдых"},
			SellerID:    1,
			CreatedAt:   now.Add(-72 * time.Hour),
			Views:       12,
		},
		{
			ID:          5,
			Title:       "Коллекция марок СССР",
			Description: "Большая коллекция, разные годы. Возможен торг.",
			Price:       999999, // Expensive
			Photos:      []string{"https://example.com/stamps.jpg"},
			Tags:        []string{"коллекционирование", "марки", "хобби", "ссср", "раритет"},
			SellerID:    2,
			CreatedAt:   now.Add(-100 * 24 * time.Hour),
			Views:       1024,
		},
		{
			ID:          6,
			Title:       "Услуги репетитора по математике",
			Description: "Подготовка к ЕГЭ и ОГЭ. Опыт 5 лет.",
			Price:       1500,
			Photos:      []string{"https://example.com/tutor.jpg"},
			Tags:        []string{"услуги", "репетитор", "образование", "математика"},
			SellerID:    4,
			CreatedAt:   now.Add(-1 * time.Hour),
			Views:       5,
		},
		{
			ID:          7,
			Title:       "Empty tags and empty photos",
			Description: "This ad has no tags and no photos to test edge cases.",
			Price:       100,
			Photos:      []string{},
			Tags:        []string{},
			SellerID:    5,
			CreatedAt:   now,
			Views:       0,
		},
		{
			ID:          8,
			Title:       "Очень длинное название объявления, которое должно тестировать как фронтенд и бекенд справляются с длинными строками",
			Description: "Очень длинное описание объявления, которое должно тестировать как фронтенд и бекенд справляются с длинными строками. Очень длинное описание объявления, которое должно тестировать как фронтенд и бекенд справляются с длинными строками. Очень длинное описание объявления, которое должно тестировать как фронтенд и бекенд справляются с длинными строками.",
			Price:       123456789,
			Photos:      []string{"https://example.com/long.jpg"},
			Tags:        []string{"тест", "длинный", "крайний случай", "очень много тегов", "один", "два", "три", "четыре", "пять", "шесть", "семь", "восемь"},
			SellerID:    1,
			CreatedAt:   now,
			Views:       999999,
		},
		{
			ID:          9,
			Title:       "Книга 'Гарри Поттер и Философский камень' РОСМЭН",
			Description: "Идеальное состояние, первое издание.",
			Price:       3500,
			Photos:      []string{"https://example.com/hp1.jpg", "https://example.com/hp2.jpg"},
			Tags:        []string{"книги", "гарри поттер", "росмэн"},
			SellerID:    3,
			CreatedAt:   now.Add(-5 * 24 * time.Hour),
			Views:       300,
		},
		{
			ID:          10,
			Title:       "iPhone 13 Pro 256GB",
			Description: "Разбит экран, FaceID работает, батарея 85%.",
			Price:       45000,
			Photos:      []string{"https://example.com/iphone-broken1.jpg", "https://example.com/iphone-broken2.jpg"},
			Tags:        []string{"телефон", "iphone", "apple", "на запчасти"},
			SellerID:    5,
			CreatedAt:   now.Add(-10 * time.Minute),
			Views:       42,
		},
	}
}

// метод для получения данных (чтобы не обращаться к полю data напрямую)
func (r *AdsRepository) GetAll() []models.Ad {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// создаем новый слайс и копируем туда данные
	result := make([]models.Ad, len(r.data))
	copy(result, r.data)
	return result
}
