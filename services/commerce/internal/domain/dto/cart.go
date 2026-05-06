package dto

// CartItemResponse представляет один товар в корзине
type CartItemResponse struct {
	ProductID  int64  `json:"product_id"`
	Title      string `json:"title"`
	Price      int64  `json:"price"`
	SellerID   int64  `json:"seller_id"`
	SellerName string `json:"seller_name"`
	ImagePath  string `json:"image_path,omitempty"`
}

// CartResponse ответ на запрос GET /api/cart
type CartResponse struct {
	Items      []CartItemResponse `json:"items"`
	TotalPrice int64              `json:"total_price"`
}

// AddToCartRequest для POST /api/cart
type AddToCartRequest struct {
	ProductID int64 `json:"product_id"`
}
