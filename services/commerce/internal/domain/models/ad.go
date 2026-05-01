package models

// Ad — облегчённая модель объявления, нужная commerce-сервису.
// Полная модель живёт в catalog/internal/domain/models/ad.go;
// commerce получает её через gRPC catalog.GetAd и использует только
// поля, необходимые для логики корзины и чатов.
const (
	AdStatusActive   = "active"
	AdStatusReserved = "reserved"
	AdStatusSold     = "sold"
)

type Ad struct {
	ID       int64
	SellerID int64
	Title    string
	Price    int64
	Status   string
}
