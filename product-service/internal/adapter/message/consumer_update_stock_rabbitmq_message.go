package message

import (
	"context"
	"encoding/json"
	"fmt"

	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpdateStockHandler processes stock-update messages from the order service.
// Implements rabbitmq.MessageHandler.
//
// Processing is transactional: a row in stock_deductions claims idempotency first,
// then the product row is locked (SELECT FOR UPDATE) and stock is decremented.
type UpdateStockHandler struct {
	db     *gorm.DB
	logger zerolog.Logger
}

func NewUpdateStockHandler(db *gorm.DB, logger zerolog.Logger) *UpdateStockHandler {
	return &UpdateStockHandler{
		db:     db,
		logger: logger.With().Str("component", "update_stock_handler").Logger(),
	}
}

func (h *UpdateStockHandler) Handle(ctx context.Context, body []byte) error {
	var orderItem entity.PublishOrderItemEntity
	if err := json.Unmarshal(body, &orderItem); err != nil {
		return fmt.Errorf("unmarshal order item: %w", err)
	}

	if orderItem.ProductID <= 0 || orderItem.Quantity <= 0 {
		return fmt.Errorf("invalid order item: product_id=%d quantity=%d", orderItem.ProductID, orderItem.Quantity)
	}

	dedupKey := entity.DedupKeyForStock(orderItem, body)
	if len(dedupKey) > 512 {
		dedupKey = dedupKey[:512]
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Claim idempotency first. ON CONFLICT DO NOTHING → duplicate message → skip work.
		claim := model.StockDeduction{
			DedupKey:  dedupKey,
			ProductID: orderItem.ProductID,
			Quantity:  orderItem.Quantity,
		}
		res := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedup_key"}},
			DoNothing: true,
		}).Create(&claim)
		if res.Error != nil {
			return fmt.Errorf("claim stock deduction: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			h.logger.Info().
				Str("dedup_key", dedupKey).
				Int64("product_id", orderItem.ProductID).
				Msg("stock deduction already applied (idempotent skip)")
			return nil
		}

		var product model.Product
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&product, orderItem.ProductID).Error; err != nil {
			return fmt.Errorf("find product %d: %w", orderItem.ProductID, err)
		}

		if product.Stock < int(orderItem.Quantity) {
			return fmt.Errorf("insufficient stock for product %d: have %d, need %d",
				orderItem.ProductID, product.Stock, orderItem.Quantity)
		}

		product.Stock -= int(orderItem.Quantity)
		if err := tx.WithContext(ctx).Save(&product).Error; err != nil {
			return fmt.Errorf("update stock for product %d: %w", orderItem.ProductID, err)
		}

		h.logger.Info().
			Int64("product_id", orderItem.ProductID).
			Int64("quantity", orderItem.Quantity).
			Int("remaining_stock", product.Stock).
			Msg("stock updated")
		return nil
	})
}
