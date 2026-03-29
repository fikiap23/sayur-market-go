package response

type CategoryListAdminResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	TotalProduct int    `json:"total_product"`
}

type CategoryDetailResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Icon        string  `json:"icon"`
	Slug        string  `json:"slug"`
	Status      string  `json:"status"`
	Description *string `json:"description"`
}

type CategoryListHomeResponse struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	Slug string `json:"slug"`
}

type CategoryListShopResponse struct {
	Name  string                     `json:"name"`
	Slug  string                     `json:"slug"`
	Child []CategoryListShopResponse `json:"child"`
}
