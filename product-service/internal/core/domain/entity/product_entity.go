package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type ProductEntity struct {
	ID           int64           `json:"id"`
	CategorySlug *string         `json:"category_slug"`
	ParentID     *int64          `json:"parent_id"`
	Name         string          `json:"name"`
	Image        string          `json:"image"`
	Description  *string         `json:"description"`
	RegulerPrice int64           `json:"reguler_price"`
	SalePrice    int64           `json:"sale_price"`
	Unit         string          `json:"unit"`
	Weight       int64           `json:"weight"`
	Stock        int             `json:"stock"`
	Variant      int             `json:"variant"`
	Status       string          `json:"status"`
	CategoryName *string         `json:"category_name"`
	Child        []ProductEntity `json:"child,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    *time.Time      `json:"updated_at,omitempty"`
}

type QueryStringProduct struct {
	Search       string `json:"search"`
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
	OrderBy      string `json:"order_by"`
	OrderType    string `json:"order_type"`
	CategorySlug string `json:"category_slug"`
	StartPrice   int64  `json:"start_price"`
	EndPrice     int64  `json:"end_price"`
	Status       string `json:"status"`
}

type PublishOrderItemEntity struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
	// Optional: set by order-service for idempotent processing (recommended).
	OrderID     *int64 `json:"order_id,omitempty"`
	OrderItemID *int64 `json:"order_item_id,omitempty"`
	// Optional: unique key per logical operation (e.g. UUID from order service).
	IdempotencyKey *string `json:"idempotency_key,omitempty"`
}

// DedupKeyForStock returns a stable key for stock_deductions. When order metadata
// is absent, the SHA-256 of the raw message body is used (exact duplicate redeliveries only).
func DedupKeyForStock(p PublishOrderItemEntity, rawBody []byte) string {
	if p.IdempotencyKey != nil && *p.IdempotencyKey != "" {
		return "idemp:" + *p.IdempotencyKey
	}
	if p.OrderID != nil && p.OrderItemID != nil {
		return fmt.Sprintf("order:%d:item:%d", *p.OrderID, *p.OrderItemID)
	}
	if p.OrderID != nil {
		return fmt.Sprintf("order:%d:product:%d:qty:%d", *p.OrderID, p.ProductID, p.Quantity)
	}
	sum := sha256.Sum256(rawBody)
	return "body:" + hex.EncodeToString(sum[:])
}

// EffectiveVersionTime orders events for Elasticsearch: prefers updated_at when it is newer than created_at.
func (p ProductEntity) EffectiveVersionTime() time.Time {
	t := p.CreatedAt
	if p.UpdatedAt != nil && !p.UpdatedAt.IsZero() && p.UpdatedAt.After(t) {
		return *p.UpdatedAt
	}
	return t
}
