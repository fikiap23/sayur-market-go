package response

type DefaultResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type DefaultResponseWithPaginations struct {
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Page       int64 `json:"page"`
	TotalCount int64 `json:"total_count"`
	PerPage    int64 `json:"per_page"`
	TotalPage  int64 `json:"total_page"`
}

func ResponseSuccess(message string, data interface{}) DefaultResponse {
	return DefaultResponse{
		Message: message,
		Data:    data,
	}
}

func ResponseSuccessWithPagination(message string, data interface{}, page, totalData, totalPage, limit int64) DefaultResponseWithPaginations {
	return DefaultResponseWithPaginations{
		Message: message,
		Data:    data,
		Pagination: &Pagination{
			Page:       page,
			TotalCount: totalData,
			PerPage:    limit,
			TotalPage:  totalPage,
		},
	}
}

func ResponseError(message string) DefaultResponse {
	return DefaultResponse{
		Message: message,
		Data:    nil,
	}
}
