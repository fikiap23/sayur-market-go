package entity

type CategoryEntity struct {
	ID          int64            `json:"id"`
	ParentID    *int64           `json:"parent_id"`
	Name        string           `json:"name"`
	Icon        string           `json:"icon"`
	Status      bool             `json:"status"`
	Slug        string           `json:"slug"`
	Description string           `json:"description"`
	Products    []ProductEntity  `json:"products,omitempty"`
	Children    []CategoryEntity `json:"children,omitempty"`
}

type QueryStringEntity struct {
	Search    string `json:"search"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	OrderBy   string `json:"order_by"`
	OrderType string `json:"order_type"`
}
