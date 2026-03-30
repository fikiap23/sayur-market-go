package message

import (
	"context"
	"encoding/json"
	"fmt"

	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// UpdateStockHandler processes stock-update messages from the order service.
// Implements rabbitmq.MessageHandler.
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

	var product model.Product
	if err := h.db.WithContext(ctx).First(&product, orderItem.ProductID).Error; err != nil {
		return fmt.Errorf("find product %d: %w", orderItem.ProductID, err)
	}

	if product.Stock < int(orderItem.Quantity) {
		return fmt.Errorf("insufficient stock for product %d: have %d, need %d",
			orderItem.ProductID, product.Stock, orderItem.Quantity)
	}

	product.Stock -= int(orderItem.Quantity)
	if err := h.db.WithContext(ctx).Save(&product).Error; err != nil {
		return fmt.Errorf("update stock for product %d: %w", orderItem.ProductID, err)
	}

	h.logger.Info().
		Int64("product_id", orderItem.ProductID).
		Int64("quantity", orderItem.Quantity).
		Int("remaining_stock", product.Stock).
		Msg("stock updated")
	return nil
}
