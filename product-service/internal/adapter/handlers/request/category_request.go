package request

type CreateCategoryRequest struct {
	Name        string  `json:"name" validate:"required"`
	Icon        string  `json:"icon" validate:"required"`
	Description *string `json:"description"`
	Status      string  `json:"status" validate:"required"`
	ParentID    *int64  `json:"parent_id"`
}
