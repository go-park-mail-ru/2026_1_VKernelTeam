package dto

//go:generate easyjson -all $GOFILE

import "time"

// PurchaseSourceCart / PurchaseSourceChat — допустимые значения поля source.
const (
	PurchaseSourceCart = "cart"
	PurchaseSourceChat = "chat"
)

// SellerPreview — краткая информация о продавце в карточке покупки.
type SellerPreview struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AvatarPath string `json:"avatar_path,omitempty"`
}

// PurchaseItem — одна покупка в выдаче GET /profile/purchases.
// Снимок данных товара (title/photo/price/location) живёт в product/order_item.
// Если товар soft-deleted (product.deleted_at), всё равно отдаём содержимое product —
// фронт не должен падать.
type PurchaseItem struct {
	OrderID     int64         `json:"order_id"`
	ProductID   int64         `json:"product_id"`
	Title       string        `json:"title"`
	Price       int64         `json:"price"`
	Photo       string        `json:"photo,omitempty"`
	Location    string        `json:"location,omitempty"`
	Seller      SellerPreview `json:"seller"`
	Source      string        `json:"source"`
	PurchasedAt time.Time     `json:"purchased_at"`
	ChatID      *int64        `json:"chat_id,omitempty"`
}

// PurchaseListResponse — страница покупок с курсором на следующую страницу.
// next_cursor — id последнего order в текущей странице; nil, когда страниц больше нет.
type PurchaseListResponse struct {
	Purchases  []PurchaseItem `json:"purchases"`
	NextCursor *int64         `json:"next_cursor,omitempty"`
}
