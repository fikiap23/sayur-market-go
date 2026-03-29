package entity

import "time"

type ProductEntity struct {
	ID           int64           `json:"id"`
	CategorySlug *string         `json:"category_slug"`
	ParentID     *int64          `json:"parent_id"`
	Name         string          `json:"name"`
	Image        string          `json:"image"`
	Description  string          `json:"description"`
	RegulerPrice int64           `json:"reguler_price"`
	SalePrice    int64           `json:"sale_price"`
	Unit         string          `json:"unit"`
	Weight       int64           `json:"weight"`
	Stock        int             `json:"stock"`
	Variant      int             `json:"variant"`
	Status       string          `json:"status"`
	CategoryName *string         `json:"category_name"`
	Children     []ProductEntity `json:"children,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
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
}
