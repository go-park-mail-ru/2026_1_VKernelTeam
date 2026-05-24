// Package review — хранилище отзывов в PostgreSQL.
package review

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opReviewCreate          = "db.review.Create"
	opReviewUpdate          = "db.review.Update"
	opReviewDelete          = "db.review.Delete"
	opReviewGetByID         = "db.review.GetByID"
	opReviewGetResponseByID = "db.review.GetResponseByID"
	opReviewListByReceiver  = "db.review.ListByReceiver"
	opReviewListBySender    = "db.review.ListBySender"
	opReviewSummary         = "db.review.SummaryByReceiver"
	opReviewExistsPurchase  = "db.review.ExistsPurchaseRequest"
	opReviewProductSeller   = "db.review.GetProductSellerID"

	pgUniqueViolation = "23505"
)

// reviewResponseSelect — общий SELECT-фрагмент для выдачи dto.ReviewResponse
// с превью отправителя и товара. Используется в GetResponseByID, ListByReceiver
// и ListBySender. Конкретный WHERE/ORDER/LIMIT добавляется на месте.
const reviewResponseSelect = `
		SELECT r.id, r.receiver_id, r.rating, r.content, r.created_at, r.updated_at,
		       u.id, u.first_name, COALESCE(u.avatar_path, ''),
		       p.id, p.title, p.price, p.status,
		       COALESCE(
		           (SELECT file_path FROM product_image
		            WHERE product_id = p.id ORDER BY sort_order ASC LIMIT 1),
		           ''
		       ) AS photo
		FROM review r
		JOIN "user" u  ON u.id = r.sender_id
		JOIN product p ON p.id = r.product_id`

// Sentinel-ошибки слоя хранения отзывов.
var (
	ErrReviewNotFound      = errors.New("review not found")
	ErrReviewAlreadyExists = errors.New("review already exists")
	ErrProductNotFound     = errors.New("product not found")
)

// PgxPool — минимальный контракт пула pgx, который позволяет подменять его
// pgxmock в тестах.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// ReviewStorage отвечает за CRUD-операции и выборки отзывов.
type ReviewStorage struct {
	pool PgxPool
	log  *slog.Logger
}

// NewStorage создаёт хранилище отзывов поверх пула pgx.
func NewStorage(pool PgxPool, log *slog.Logger) *ReviewStorage {
	return &ReviewStorage{pool: pool, log: log}
}

// Create вставляет новый отзыв и возвращает его с заполнёнными id и таймстампами.
// При нарушении UNIQUE (sender_id, product_id, receiver_id) возвращает ErrReviewAlreadyExists.
func (s *ReviewStorage) Create(ctx context.Context, r *models.Review) (models.Review, error) {
	const query = `
		INSERT INTO review (sender_id, receiver_id, product_id, rating, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	out := *r
	err := s.pool.QueryRow(ctx, query, r.SenderID, r.ReceiverID, r.ProductID, r.Rating, r.Content).
		Scan(&out.ID, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return models.Review{}, ErrReviewAlreadyExists
		}
		s.log.ErrorContext(ctx, "failed to insert review",
			slog.String("op", opReviewCreate),
			slog.String("error", err.Error()),
		)
		return models.Review{}, fmt.Errorf("ReviewStorage.Create: %w", err)
	}
	return out, nil
}

// Update меняет rating/content отзыва, принадлежащего senderID. Возвращает
// ErrReviewNotFound, если отзыв не найден или принадлежит другому пользователю.
func (s *ReviewStorage) Update(
	ctx context.Context, id, senderID int64, rating int, content string,
) (models.Review, error) {
	const query = `
		UPDATE review
		SET rating = $3, content = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND sender_id = $2
		RETURNING id, sender_id, receiver_id, product_id, rating, content, created_at, updated_at
	`

	var r models.Review
	err := s.pool.QueryRow(ctx, query, id, senderID, rating, content).Scan(
		&r.ID, &r.SenderID, &r.ReceiverID, &r.ProductID,
		&r.Rating, &r.Content, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Review{}, ErrReviewNotFound
		}
		s.log.ErrorContext(ctx, "failed to update review",
			slog.String("op", opReviewUpdate),
			slog.String("error", err.Error()),
		)
		return models.Review{}, fmt.Errorf("ReviewStorage.Update: %w", err)
	}
	return r, nil
}

// Delete удаляет отзыв, принадлежащий senderID. Возвращает ErrReviewNotFound,
// если ничего не удалено.
func (s *ReviewStorage) Delete(ctx context.Context, id, senderID int64) error {
	const query = `DELETE FROM review WHERE id = $1 AND sender_id = $2`

	res, err := s.pool.Exec(ctx, query, id, senderID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to delete review",
			slog.String("op", opReviewDelete),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("ReviewStorage.Delete: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

// GetByID возвращает отзыв по id. ErrReviewNotFound, если строки нет.
func (s *ReviewStorage) GetByID(ctx context.Context, id int64) (models.Review, error) {
	const query = `
		SELECT id, sender_id, receiver_id, product_id, rating, content, created_at, updated_at
		FROM review
		WHERE id = $1
	`

	var r models.Review
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.SenderID, &r.ReceiverID, &r.ProductID,
		&r.Rating, &r.Content, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Review{}, ErrReviewNotFound
		}
		s.log.ErrorContext(ctx, "failed to get review by id",
			slog.String("op", opReviewGetByID),
			slog.String("error", err.Error()),
		)
		return models.Review{}, fmt.Errorf("ReviewStorage.GetByID: %w", err)
	}
	return r, nil
}

// GetResponseByID возвращает отзыв в виде dto.ReviewResponse с заполненными
// превью отправителя и товара (тот же JOIN, что и в ListByReceiver, но WHERE r.id = $1).
// Используется в CreateReview/UpdateReview, чтобы вернуть клиенту полноценный
// ответ сразу после INSERT/UPDATE без N+1.
// ErrReviewNotFound, если строки нет.
func (s *ReviewStorage) GetResponseByID(ctx context.Context, reviewID int64) (dto.ReviewResponse, error) {
	const query = `
		` + reviewResponseSelect + `
		WHERE r.id = $1
`

	var item dto.ReviewResponse
	err := s.pool.QueryRow(ctx, query, reviewID).Scan(
		&item.ID, &item.ReceiverID, &item.Rating, &item.Content, &item.CreatedAt, &item.UpdatedAt,
		&item.Sender.ID, &item.Sender.Name, &item.Sender.AvatarPath,
		&item.Product.ID, &item.Product.Title, &item.Product.Price, &item.Product.Status, &item.Product.Photo,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.ReviewResponse{}, ErrReviewNotFound
		}
		s.log.ErrorContext(ctx, "failed to get review response by id",
			slog.String("op", opReviewGetResponseByID),
			slog.String("error", err.Error()),
		)
		return dto.ReviewResponse{}, fmt.Errorf("ReviewStorage.GetResponseByID: %w", err)
	}
	return item, nil
}

// ListByReceiver возвращает страницу отзывов о receiverID с курсорной пагинацией.
// cursor=nil — первая страница; cursor=N — следующая страница (id < N).
// Превью отправителя и товара джойнится одним SQL, чтобы избежать N+1.
func (s *ReviewStorage) ListByReceiver(
	ctx context.Context, receiverID int64, cursor *int64, limit int,
) ([]dto.ReviewResponse, error) {
	const query = `
		` + reviewResponseSelect + `
		WHERE r.receiver_id = $1
		  AND ($2::bigint IS NULL OR r.id < $2)
		ORDER BY r.id DESC
		LIMIT $3
`

	rows, err := s.pool.Query(ctx, query, receiverID, cursor, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query reviews by receiver",
			slog.String("op", opReviewListByReceiver),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("ReviewStorage.ListByReceiver: %w", err)
	}
	defer rows.Close()

	out := []dto.ReviewResponse{}
	for rows.Next() {
		var item dto.ReviewResponse
		if err := rows.Scan(
			&item.ID, &item.ReceiverID, &item.Rating, &item.Content, &item.CreatedAt, &item.UpdatedAt,
			&item.Sender.ID, &item.Sender.Name, &item.Sender.AvatarPath,
			&item.Product.ID, &item.Product.Title, &item.Product.Price, &item.Product.Status, &item.Product.Photo,
		); err != nil {
			return nil, fmt.Errorf("ReviewStorage.ListByReceiver: scan: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ReviewStorage.ListByReceiver: rows: %w", err)
	}
	return out, nil
}

// ListBySender возвращает страницу отзывов, оставленных senderID.
func (s *ReviewStorage) ListBySender(
	ctx context.Context, senderID int64, cursor *int64, limit int,
) ([]dto.ReviewResponse, error) {
	const query = `
		` + reviewResponseSelect + `
		WHERE r.sender_id = $1
		  AND ($2::bigint IS NULL OR r.id < $2)
		ORDER BY r.id DESC
		LIMIT $3
`

	rows, err := s.pool.Query(ctx, query, senderID, cursor, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query reviews by sender",
			slog.String("op", opReviewListBySender),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("ReviewStorage.ListBySender: %w", err)
	}
	defer rows.Close()

	out := []dto.ReviewResponse{}
	for rows.Next() {
		var item dto.ReviewResponse
		if err := rows.Scan(
			&item.ID, &item.ReceiverID, &item.Rating, &item.Content, &item.CreatedAt, &item.UpdatedAt,
			&item.Sender.ID, &item.Sender.Name, &item.Sender.AvatarPath,
			&item.Product.ID, &item.Product.Title, &item.Product.Price, &item.Product.Status, &item.Product.Photo,
		); err != nil {
			return nil, fmt.Errorf("ReviewStorage.ListBySender: scan: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ReviewStorage.ListBySender: rows: %w", err)
	}
	return out, nil
}

// SummaryByReceiver собирает агрегат: распределение оценок по таблице review и
// total/average из денормализованных полей "user". Если пользователя нет —
// возвращается нулевой ответ без ошибки.
func (s *ReviewStorage) SummaryByReceiver(
	ctx context.Context, receiverID int64,
) (dto.ReviewSummaryResponse, error) {
	resp := dto.ReviewSummaryResponse{Distribution: map[int]int{}}

	const distQuery = `
		SELECT rating, COUNT(*)::int
		FROM review
		WHERE receiver_id = $1
		GROUP BY rating
	`

	rows, err := s.pool.Query(ctx, distQuery, receiverID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query review distribution",
			slog.String("op", opReviewSummary),
			slog.String("error", err.Error()),
		)
		return dto.ReviewSummaryResponse{}, fmt.Errorf("ReviewStorage.SummaryByReceiver: dist: %w", err)
	}

	func() {
		defer rows.Close()
		for rows.Next() {
			var rating, count int
			if err = rows.Scan(&rating, &count); err != nil {
				return
			}
			resp.Distribution[rating] = count
		}
		err = rows.Err()
	}()
	if err != nil {
		return dto.ReviewSummaryResponse{}, fmt.Errorf("ReviewStorage.SummaryByReceiver: dist scan: %w", err)
	}

	const userQuery = `SELECT rating, reviews_count FROM "user" WHERE id = $1`

	var avg float64
	var total int
	err = s.pool.QueryRow(ctx, userQuery, receiverID).Scan(&avg, &total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return resp, nil
		}
		s.log.ErrorContext(ctx, "failed to query user aggregates",
			slog.String("op", opReviewSummary),
			slog.String("error", err.Error()),
		)
		return dto.ReviewSummaryResponse{}, fmt.Errorf("ReviewStorage.SummaryByReceiver: user: %w", err)
	}
	resp.Average = avg
	resp.Total = total
	return resp, nil
}

// ExistsPurchaseRequest проверяет, что между покупателем и продавцом по товару
// есть чат с сообщением типа 'order' от покупателя. Это бизнес-аналог
// «факта оформления заказа», на котором завязано право оставить отзыв.
func (s *ReviewStorage) ExistsPurchaseRequest(
	ctx context.Context, buyerID, sellerID, productID int64,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM chat c
			JOIN message m ON m.chat_id = c.id
			WHERE c.product_id = $3
			  AND c.buyer_id   = $1
			  AND c.seller_id  = $2
			  AND m.sender_id  = $1
			  AND m.msg_type   = 'order'
		)
	`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, buyerID, sellerID, productID).Scan(&exists); err != nil {
		s.log.ErrorContext(ctx, "failed to check purchase request",
			slog.String("op", opReviewExistsPurchase),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("ReviewStorage.ExistsPurchaseRequest: %w", err)
	}
	return exists, nil
}

// GetProductSellerID возвращает seller_id товара. ErrProductNotFound, если товара нет.
func (s *ReviewStorage) GetProductSellerID(ctx context.Context, productID int64) (int64, error) {
	const query = `SELECT seller_id FROM product WHERE id = $1`

	var sellerID int64
	err := s.pool.QueryRow(ctx, query, productID).Scan(&sellerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrProductNotFound
		}
		s.log.ErrorContext(ctx, "failed to get product seller id",
			slog.String("op", opReviewProductSeller),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("ReviewStorage.GetProductSellerID: %w", err)
	}
	return sellerID, nil
}
