package entity

import "time"

type ProductHttpClientResponse struct {
	Message string                `json:"message"`
	Data    ProductResponseEntity `json:"data"`
}

type ChildProductResponseEntity struct {
	ID           int     `json:"id"`
	Weight       int     `json:"weight"`
	Stock        int     `json:"stock"`
	RegulerPrice float64 `json:"reguler_price"`
	SalePrice    float64 `json:"sale_price"`
	Unit         string  `json:"unit"`
	Image        string  `json:"image"`
}

type ProductResponseEntity struct {
	ID            int                          `json:"id"`
	ProductName   string                       `json:"product_name"`
	ParentID      int                          `json:"parent_id"`
	ProductImage  string                       `json:"product_image"`
	CategoryName  string                       `json:"category_name"`
	ProductStatus string                       `json:"product_status"`
	SalePrice     float64                      `json:"sale_price"`
	RegulerPrice  float64                      `json:"reguler_price"`
	CreatedAt     time.Time                    `json:"created_at"`
	Unit          string                       `json:"unit"`
	Weight        int                          `json:"weight"`
	Stock         int                          `json:"stock"`
	Child         []ChildProductResponseEntity `json:"child"`
}
