package response

import "time"

type ProductListResponse struct {
	ID            int64     `json:"id"`
	ProductName   string    `json:"product_name"`
	ParentID      int64     `json:"parent_id"`
	ProductImage  string    `json:"product_image"`
	CategoryName  string    `json:"category_name"`
	ProductStatus string    `json:"product_status"`
	SalePrice     int64     `json:"sale_price"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProductDetailResponse struct {
	ID                 int64                  `json:"id"`
	ProductName        string                 `json:"product_name"`
	ParentID           int64                  `json:"parent_id"`
	ProductImage       string                 `json:"product_image"`
	CategoryName       string                 `json:"category_name"`
	CategorySlug       string                 `json:"category_slug"`
	ProductStatus      string                 `json:"product_status"`
	ProductDescription string                 `json:"product_description"`
	SalePrice          int64                  `json:"sale_price"`
	RegulerPrice       int64                  `json:"reguler_price"`
	CreatedAt          time.Time              `json:"created_at"`
	Unit               string                 `json:"unit"`
	Weight             int                    `json:"weight"`
	Stock              int                    `json:"stock"`
	Child              []ProductChildResponse `json:"child"`
}

type ProductChildResponse struct {
	ID           int64 `json:"id"`
	Weight       int   `json:"weight"`
	Stock        int   `json:"stock"`
	RegulerPrice int64 `json:"reguler_price"`
	SalePrice    int64 `json:"sale_price"`
}

type ProductHomeListResponse struct {
	ID           int64  `json:"id"`
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	CategoryName string `json:"category_name"`
	SalePrice    int64  `json:"sale_price"`
	RegulerPrice int64  `json:"reguler_price"`
}

type ProductHomeDetailResponse struct {
	ID           int64                      `json:"id"`
	ProductName  string                     `json:"product_name"`
	CategoryName string                     `json:"category_name"`
	Description  string                     `json:"description"`
	Unit         string                     `json:"unit"`
	ProductImage string                     `json:"image"`
	SalePrice    int64                      `json:"sale_price"`
	RegulerPrice int64                      `json:"reguler_price"`
	Stock        int                        `json:"stock"`
	Weight       int                        `json:"weight"`
	Child        []ProductChildHomeResponse `json:"child"`
}

type ProductChildHomeResponse struct {
	ID           int64  `json:"id"`
	Weight       int    `json:"weight"`
	Stock        int    `json:"stock"`
	RegulerPrice int64  `json:"reguler_price"`
	SalePrice    int64  `json:"sale_price"`
	Image        string `json:"image"`
}
