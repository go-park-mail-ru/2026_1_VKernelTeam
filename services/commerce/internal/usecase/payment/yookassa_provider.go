package payment

//go:generate easyjson -all $GOFILE

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	yookassaHTTPTimeout = 10 * time.Second
	yookassaMaxBodySize = 1 << 20 // 1 MiB — защита от перегруженного payload'а.

	// Статусы ЮКассы (см. https://yookassa.ru/developers/payment-acceptance/getting-started/payment-process).
	yooStatusPending           = "pending"
	yooStatusWaitingForCapture = "waiting_for_capture"
	yooStatusSucceeded         = "succeeded"
	yooStatusCanceled          = "canceled"
)

// YooKassaConfig — параметры YooKassaProvider.
type YooKassaConfig struct {
	ShopID    string
	SecretKey string
	APIURL    string // обычно https://api.yookassa.ru/v3
	ReturnURL string
}

// YooKassaProvider — реальный платёжный провайдер. Поддерживает тестовый и боевой режимы
// (различаются только парой shopId/secretKey).
type YooKassaProvider struct {
	cfg    YooKassaConfig
	client *http.Client
	log    *slog.Logger
}

// NewYooKassaProvider создаёт провайдера с HTTP-клиентом и заданным таймаутом.
// Если httpClient nil — используется *http.Client с таймаутом по умолчанию.
func NewYooKassaProvider(cfg YooKassaConfig, client *http.Client, log *slog.Logger) *YooKassaProvider {
	if client == nil {
		client = &http.Client{Timeout: yookassaHTTPTimeout}
	}
	return &YooKassaProvider{
		cfg:    cfg,
		client: client,
		log:    log,
	}
}

// yooAmount — представление денежной суммы в API ЮКассы.
type yooAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type yooConfirmationReq struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type yooConfirmationResp struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url"`
}

type yooCreatePaymentReq struct {
	Amount       yooAmount          `json:"amount"`
	Capture      bool               `json:"capture"`
	Confirmation yooConfirmationReq `json:"confirmation"`
	Description  string             `json:"description,omitempty"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
}

type yooPaymentResp struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Amount       yooAmount           `json:"amount"`
	Confirmation yooConfirmationResp `json:"confirmation"`
	Paid         bool                `json:"paid"`
}

// InitPayment создаёт платёж в ЮКассе. Возвращает pending + confirmation_url
// (для редиректа пользователя). Терминальный статус прилетит через webhook.
func (y *YooKassaProvider) InitPayment(
	ctx context.Context,
	p models.Payment,
	idempotencyKey string,
) (InitResult, error) {
	body := yooCreatePaymentReq{
		Amount:  yooAmount{Value: rublesToYooValue(p.Amount), Currency: "RUB"},
		Capture: true,
		Confirmation: yooConfirmationReq{
			Type:      "redirect",
			ReturnURL: y.cfg.ReturnURL,
		},
		Description: fmt.Sprintf("Top-up wallet uid=%d payment=%d", p.UserID, p.ID),
		Metadata: map[string]string{
			"user_id":    strconv.FormatInt(p.UserID, 10),
			"payment_id": strconv.FormatInt(p.ID, 10),
		},
	}
	payload, err := easyjson.Marshal(body)
	if err != nil {
		return InitResult{}, fmt.Errorf("yookassa.InitPayment: marshal: %w", err)
	}

	rawResp, err := y.doRequest(ctx, http.MethodPost, "/payments", payload, idempotencyKey)
	if err != nil {
		return InitResult{}, fmt.Errorf("yookassa.InitPayment: %w", err)
	}

	var parsed yooPaymentResp
	if err := easyjson.Unmarshal(rawResp, &parsed); err != nil {
		return InitResult{}, fmt.Errorf("yookassa.InitPayment: parse response: %w", err)
	}

	return InitResult{
		Status:          mapYooStatus(parsed.Status),
		ProviderRef:     parsed.ID,
		ConfirmationURL: parsed.Confirmation.ConfirmationURL,
		RawPayload:      rawResp,
	}, nil
}

// GetPayment запрашивает актуальное состояние платежа по providerRef.
func (y *YooKassaProvider) GetPayment(ctx context.Context, providerRef string) (InitResult, error) {
	if providerRef == "" {
		return InitResult{}, errors.New("yookassa.GetPayment: empty providerRef")
	}
	rawResp, err := y.doRequest(ctx, http.MethodGet, "/payments/"+providerRef, nil, "")
	if err != nil {
		return InitResult{}, fmt.Errorf("yookassa.GetPayment: %w", err)
	}

	var parsed yooPaymentResp
	if err := easyjson.Unmarshal(rawResp, &parsed); err != nil {
		return InitResult{}, fmt.Errorf("yookassa.GetPayment: parse response: %w", err)
	}
	return InitResult{
		Status:          mapYooStatus(parsed.Status),
		ProviderRef:     parsed.ID,
		ConfirmationURL: parsed.Confirmation.ConfirmationURL,
		RawPayload:      rawResp,
	}, nil
}

// doRequest выполняет HTTP-запрос к ЮКассе с Basic Auth и (опционально) Idempotence-Key.
func (y *YooKassaProvider) doRequest(
	ctx context.Context,
	method, path string,
	body []byte,
	idempotencyKey string,
) ([]byte, error) {
	url := strings.TrimRight(y.cfg.APIURL, "/") + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.SetBasicAuth(y.cfg.ShopID, y.cfg.SecretKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotence-Key", idempotencyKey)
	}

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, yookassaMaxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		y.log.ErrorContext(ctx, "yookassa api error",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(rawResp)),
		)
		return nil, fmt.Errorf("yookassa http %d: %s", resp.StatusCode, truncate(string(rawResp), 256))
	}
	return rawResp, nil
}

// mapYooStatus переводит статусы ЮКассы во внутренние models.PaymentStatus*.
//
//   - succeeded                 → succeeded
//   - canceled                  → cancelled
//   - pending / waiting_capture → pending
//   - что-то ещё                → failed
func mapYooStatus(s string) string {
	switch s {
	case yooStatusSucceeded:
		return models.PaymentStatusSucceeded
	case yooStatusCanceled:
		return models.PaymentStatusCancelled
	case yooStatusPending, yooStatusWaitingForCapture:
		return models.PaymentStatusPending
	default:
		return models.PaymentStatusFailed
	}
}

// rublesToYooValue переводит сумму в целых рублях в строку "X.00" — формат ЮКассы.
func rublesToYooValue(rubles int64) string {
	return strconv.FormatInt(rubles, 10) + ".00"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
