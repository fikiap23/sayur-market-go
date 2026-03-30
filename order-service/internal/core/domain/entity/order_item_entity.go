package entity

type OrderItemEntity struct {
	ID            int64  `json:"id"`
	OrderID       int64  `json:"order_id"`
	ProductID     int64  `json:"product_id"`
	Quantity      int64  `json:"quantity"`
	OrderCode     string `json:"order_code"`
	ProductName   string `json:"product_name"`
	ProductImage  string `json:"product_image"`
	Price         int64  `json:"price"`
	ProductUnit   string `json:"product_unit"`
	ProductWeight int64  `json:"product_weight"`
}

// PublishOrderItemEntity is sent to product-service for stock updates.
// Optional fields align with product-service DedupKeyForStock.
type PublishOrderItemEntity struct {
	ProductID      int64   `json:"product_id"`
	Quantity       int64   `json:"quantity"`
	OrderID        *int64  `json:"order_id,omitempty"`
	OrderItemID    *int64  `json:"order_item_id,omitempty"`
	IdempotencyKey *string `json:"idempotency_key,omitempty"`
}
