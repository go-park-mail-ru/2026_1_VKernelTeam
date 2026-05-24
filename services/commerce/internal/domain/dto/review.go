package dto

//go:generate easyjson -all $GOFILE

import "time"

// CreateReviewRequest — тело POST /reviews.
type CreateReviewRequest struct {
	ReceiverID int64  `json:"receiver_id"`
	ProductID  int64  `json:"product_id"`
	Rating     int    `json:"rating"`
	Content    string `json:"content"`
}

// UpdateReviewRequest — тело PUT /reviews/{id}.
type UpdateReviewRequest struct {
	Rating  int    `json:"rating"`
	Content string `json:"content"`
}

// ReviewResponse — отзыв с превью отправителя и товара.
type ReviewResponse struct {
	ID         int64       `json:"id"`
	Sender     UserPreview `json:"sender"`
	ReceiverID int64       `json:"receiver_id"`
	Product    AdPreview   `json:"product"`
	Rating     int         `json:"rating"`
	Content    string      `json:"content"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// ReviewListResponse — страница отзывов с курсором следующей страницы.
type ReviewListResponse struct {
	Reviews    []ReviewResponse `json:"reviews"`
	NextCursor *int64           `json:"next_cursor,omitempty"`
}

// ReviewSummaryResponse — агрегат рейтинга продавца: среднее, total и
// распределение по оценкам (ключи 1..5).
type ReviewSummaryResponse struct {
	Average      float64     `json:"average"`
	Total        int         `json:"total"`
	Distribution map[int]int `json:"distribution"`
}

// CreateReviewResponse — ответ на POST /reviews: созданный отзыв.
type CreateReviewResponse struct {
	Review ReviewResponse `json:"review"`
}
