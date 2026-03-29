package handlers

import (
	"net/http"
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handlers/request"
	"product-service/internal/adapter/handlers/response"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/service"
	"product-service/utils/conv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type CategoryHandlerInterface interface {
	GetAllAdmin(c echo.Context) error
	GetByIDAdmin(c echo.Context) error
	GetBySlugAdmin(c echo.Context) error
	Create(c echo.Context) error
	Update(c echo.Context) error
	Delete(c echo.Context) error

	GetAllHome(c echo.Context) error
	GetAllShop(c echo.Context) error
}

type categoryHandler struct {
	categoryService service.CategoryServiceInterface
}

// GetAllHome implements CategoryHandlerInterface.
func (ch *categoryHandler) GetAllShop(c echo.Context) error {
	var (
		resp           = response.DefaultResponse{}
		ctx            = c.Request().Context()
		respCategories = []response.CategoryListShopResponse{}
	)

	results, err := ch.categoryService.GetAllPublished(ctx)
	if err != nil {
		log.Errorf("[CategoryHandler-1] GetAllShop: %v", err)
		if err.Error() == "404" {
			resp.Message = "Data not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respCategories = RekursifCategory(results, nil, 0)

	resp.Message = "success"
	resp.Data = respCategories
	return c.JSON(http.StatusOK, resp)
}

// GetAllHome implements CategoryHandlerInterface.
func (ch *categoryHandler) GetAllHome(c echo.Context) error {
	var (
		resp           = response.DefaultResponse{}
		ctx            = c.Request().Context()
		respCategories = []response.CategoryListHomeResponse{}
	)

	results, err := ch.categoryService.GetAllPublished(ctx)
	if err != nil {
		log.Errorf("[CategoryHandler-1] GetAllHome: %v", err)
		if err.Error() == "404" {
			resp.Message = "Data not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	for _, result := range results {
		if result.ParentID == nil {
			respCategories = append(respCategories, response.CategoryListHomeResponse{
				Name: result.Name,
				Icon: result.Icon,
				Slug: result.Slug,
			})
		}
	}

	resp.Message = "success"
	resp.Data = respCategories
	return c.JSON(http.StatusOK, resp)
}

// Create implements CategoryHandlerInterface.
func (ch *categoryHandler) Create(c echo.Context) error {
	var (
		resp    = response.DefaultResponse{}
		ctx     = c.Request().Context()
		request = request.CreateCategoryRequest{}
	)

	if err := c.Bind(&request); err != nil {
		log.Errorf("[CategoryHandler-1] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(request); err != nil {
		log.Errorf("[CategoryHandler-2] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	reqEntity := entity.CategoryEntity{
		Name:        request.Name,
		Icon:        request.Icon,
		Description: request.Description,
		Status:      request.Status,
		ParentID:    request.ParentID,
	}

	err := ch.categoryService.CreateCategory(ctx, reqEntity)
	if err != nil {
		log.Errorf("[CategoryHandler-3] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// Delete implements CategoryHandlerInterface.
func (ch *categoryHandler) Delete(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	idStr := c.Param("id")
	if idStr == "" {
		log.Errorf("[CategoryHandler-1] Delete: %v", "Invalid id")
		resp.Message = "ID is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Errorf("[CategoryHandler-2] Delete: %v", err.Error())
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = ch.categoryService.DeleteCategory(ctx, id)
	if err != nil {
		log.Errorf("[CategoryHandler-3] Delete: %v", err)
		if err.Error() == "404" {
			resp.Message = "Category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// GetByIDAdmin implements CategoryHandlerInterface.
func (ch *categoryHandler) GetByIDAdmin(c echo.Context) error {
	var (
		resp           = response.DefaultResponse{}
		ctx            = c.Request().Context()
		respCategories = response.CategoryDetailResponse{}
	)

	idStr := c.Param("id")
	if idStr == "" {
		log.Errorf("[CategoryHandler-1] GetByIDAdmin: %v", "invalid id")
		resp.Message = "ID is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Errorf("[CategoryHandler-2] GetByIDAdmin: %v", err.Error())
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	result, err := ch.categoryService.GetByID(ctx, id)
	if err != nil {
		log.Errorf("[CategoryHandler-3] GetByIDAdmin: %v", err)
		if err.Error() == "404" {
			resp.Message = "Data not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	respCategories = response.CategoryDetailResponse{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Icon:        result.Icon,
		Status:      result.Status,
		Description: result.Description,
	}

	resp.Message = "success"
	resp.Data = respCategories
	return c.JSON(http.StatusOK, resp)
}

// GetBySlugAdmin implements CategoryHandlerInterface.
func (ch *categoryHandler) GetBySlugAdmin(c echo.Context) error {
	var (
		resp           = response.DefaultResponse{}
		ctx            = c.Request().Context()
		respCategories = response.CategoryDetailResponse{}
	)

	slug := c.Param("slug")
	if slug == "" {
		log.Errorf("[CategoryHandler-1] GetBySlugAdmin: %v", "invalid slug")
		resp.Message = "Slug is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	result, err := ch.categoryService.GetBySlug(ctx, slug)
	if err != nil {
		log.Errorf("[CategoryHandler-2] GetBySlugAdmin: %v", err)
		if err.Error() == "404" {
			resp.Message = "Data not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	respCategories = response.CategoryDetailResponse{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Icon:        result.Icon,
		Status:      result.Status,
		Description: result.Description,
	}

	resp.Message = "success"
	resp.Data = respCategories
	return c.JSON(http.StatusOK, resp)
}

// Update implements CategoryHandlerInterface.
func (ch *categoryHandler) Update(c echo.Context) error {
	var (
		resp    = response.DefaultResponse{}
		ctx     = c.Request().Context()
		request = request.CreateCategoryRequest{}
	)

	idStr := c.Param("id")
	if idStr == "" {
		log.Errorf("[CategoryHandler-1] Update: %v", "Invalid id")
		resp.Message = "ID is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	id, err := conv.StringToInt64(idStr)
	if err != nil {
		log.Errorf("[CategoryHandler-2] Update: %v", err.Error())
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Bind(&request); err != nil {
		log.Errorf("[CategoryHandler-3] Update: %v", "Invalid id")
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Validate(request); err != nil {
		log.Errorf("[CategoryHandler-4] Update: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	reqEntity := entity.CategoryEntity{
		ID:          id,
		Name:        request.Name,
		Icon:        request.Icon,
		Description: request.Description,
		Status:      request.Status,
		ParentID:    request.ParentID,
	}

	err = ch.categoryService.EditCategory(ctx, reqEntity)
	if err != nil {
		log.Errorf("[CategoryHandler-5] Update: %v", err)
		if err.Error() == "404" {
			resp.Message = "Category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// GetAllAdmin implements CategoryHandlerInterface.
func (ch *categoryHandler) GetAllAdmin(c echo.Context) error {
	var (
		resp           = response.DefaultResponseWithPaginations{}
		ctx            = c.Request().Context()
		respCategories = []response.CategoryListAdminResponse{}
	)

	search := c.QueryParam("search")
	orderBy := "created_at"
	if c.QueryParam("orderBy") != "" {
		orderBy = c.QueryParam("orderBy")
	}

	orderType := "desc"
	if c.QueryParam("orderType") != "" {
		orderType = c.QueryParam("orderType")
	}

	var page int64 = 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		page, _ = conv.StringToInt64(pageStr)
		if page <= 0 {
			page = 1
		}
	}

	var perPage int64 = 10
	if perPageStr := c.QueryParam("perPage"); perPageStr != "" {
		perPage, _ = conv.StringToInt64(perPageStr)
		if perPage <= 0 {
			perPage = 10
		}
	}

	reqEntity := entity.QueryStringEntity{
		Search:    search,
		OrderBy:   orderBy,
		OrderType: orderType,
		Page:      int(page),
		Limit:     int(perPage),
	}

	results, totalData, totalPage, err := ch.categoryService.GetAll(ctx, reqEntity)
	if err != nil {
		log.Errorf("[CategoryHandler-1] Create: %v", err)
		if err.Error() == "404" {
			resp.Message = "Data not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	for _, result := range results {
		respCategories = append(respCategories, response.CategoryListAdminResponse{
			ID:           result.ID,
			Name:         result.Name,
			Icon:         result.Icon,
			Slug:         result.Slug,
			Status:       result.Status,
			TotalProduct: len(result.Products),
		})
	}

	pagination := response.Pagination{
		Page:       page,
		TotalCount: totalData,
		PerPage:    perPage,
		TotalPage:  totalPage,
	}
	resp.Message = "success"
	resp.Data = respCategories
	resp.Pagination = &pagination
	return c.JSON(http.StatusOK, resp)
}

func NewCategoryHandler(e *echo.Echo, categoryService service.CategoryServiceInterface, cfg *config.Config) CategoryHandlerInterface {
	category := &categoryHandler{categoryService: categoryService}

	categoryApp := e.Group("/categories")
	categoryApp.GET("/home", category.GetAllHome)
	categoryApp.GET("/shop", category.GetAllShop)

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg)
	adminGroup := e.Group("/admin", mid.CheckToken())
	adminGroup.GET("/categories", category.GetAllAdmin)
	adminGroup.GET("/categories/:id", category.GetByIDAdmin)
	adminGroup.GET("/categories/:slug/slug", category.GetBySlugAdmin)
	adminGroup.POST("/categories", category.Create)
	adminGroup.PUT("/categories/:id", category.Update)
	adminGroup.DELETE("/categories/:id", category.Delete)

	return category
}

func RekursifCategory(results []entity.CategoryEntity, parentID *int64, level int) []response.CategoryListShopResponse {
	var resps []response.CategoryListShopResponse

	for _, category := range results {
		if category.ParentID == parentID {
			resps = append(resps, response.CategoryListShopResponse{
				Name: category.Name,
				Slug: category.Slug,
			})

			childCategories := RekursifCategory(results, &category.ID, level+1)
			resps = append(resps, childCategories...)
		}
	}

	return resps
}
