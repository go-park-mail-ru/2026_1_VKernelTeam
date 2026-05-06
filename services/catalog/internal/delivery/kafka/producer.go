// Package kafka - domain-обёртка над общим Kafka producer для catalog.
//
// Публикует события в clover.catalog.ad-events:
//   - ad.sold     - при покупке (вызывается из gRPC UpdateAdStatus)
//   - ad.deleted  - при удалении объявления (вызывается из usecase DeleteAd)
package kafka

import (
	"context"
	"log/slog"
	"strconv"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

// Producer публикует ad-события в clover.catalog.ad-events.
type Producer struct {
	inner *sharedkafka.Producer
}

// NewProducer создаёт producer.
func NewProducer(brokers []string, log *slog.Logger) *Producer {
	return &Producer{
		inner: sharedkafka.NewProducer(brokers, sharedkafka.TopicCatalogAdEvents, log),
	}
}

// Close закрывает Kafka writer.
func (p *Producer) Close() error {
	return p.inner.Close()
}

// PublishAdDeleted публикует ad.deleted с ключом ad_id.
func (p *Producer) PublishAdDeleted(ctx context.Context, adID int64) error {
	return p.inner.Publish(ctx, strconv.FormatInt(adID, 10), sharedkafka.EventAdDeleted, sharedkafka.AdPayload{AdID: adID})
}

// PublishAdSold публикует ad.sold с ключом ad_id и buyer_id в payload.
func (p *Producer) PublishAdSold(ctx context.Context, adID, buyerID int64) error {
	return p.inner.Publish(ctx, strconv.FormatInt(adID, 10), sharedkafka.EventAdSold, sharedkafka.AdPayload{AdID: adID, BuyerID: buyerID})
}
