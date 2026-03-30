package response

type OrderAdminList struct {
	ID            int64  `json:"id"`
	OrderCode     string `json:"order_code"`
	ProductImage  string `json:"product_image"`
	CustomerName  string `json:"customer_name"`
	Status        string `json:"status"`
	PaymentType string `json:"payment_method"`
	TotalAmount   int64  `json:"total_amount"`
}

type OrderAdminDetail struct {
	ID            int64         `json:"id"`
	OrderCode     string        `json:"order_code"`
	ProductImage  string        `json:"product_image"`
	OrderDatetime string        `json:"order_datetime"`
	Status        string        `json:"order_status"`
	PaymentType string        `json:"payment_method"`
	ShippingFee   int64         `json:"shipping_fee"`
	ShippingType  string        `json:"shipping_type"`
	Remarks       string        `json:"remarks"`
	TotalAmount   int64         `json:"total_amount"`
	Customer      CustomerOrder `json:"customer"`
	OrderDetail   []OrderDetail `json:"order_detail"`
}

type OrderCustomerList struct {
	ID            int64  `json:"id"`
	OrderCode     string `json:"order_code"`
	ProductImage  string `json:"product_image"`
	ProductName   string `json:"product_name"`
	Status        string `json:"status"`
	PaymentType string `json:"payment_method"`
	TotalAmount   int64  `json:"total_amount"`
	Weight        int64  `json:"weight"`
	Unit          string `json:"unit"`
	Quantity      int64  `json:"quantity"`
	OrderDateTime string `json:"order_datetime"`
}

type CustomerOrder struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email"`
	CustomerID      int64  `json:"customer_id"`
}

type OrderDetail struct {
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	ProductPrice int64  `json:"product_price"`
	Quantity     int64  `json:"quantity"`
}
