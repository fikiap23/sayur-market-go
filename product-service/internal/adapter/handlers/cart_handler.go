package handlers

import (
	"encoding/json"
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

type CartHandlerInterface interface {
	AddToCart(c echo.Context) error
	GetCart(c echo.Context) error
	RemoveFromCart(c echo.Context) error
	RemoveAllCart(c echo.Context) error
}

type CartHandler struct {
	CartService    service.CartServiceInterface
	ProductService service.ProductServiceInterface
}

func (ch *CartHandler) RemoveAllCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] RemoveFromCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] RemoveFromCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID

	err = ch.CartService.RemoveAllCart(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler-1] RemoveAllCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// AddToCart implements CartHandlerInterface.
func (ch *CartHandler) AddToCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		request     = request.CartRequest{}
		jwtUserData = entity.JwtUserData{}
	)

	if err := c.Bind(&request); err != nil {
		log.Errorf("[CartHandler-1] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(request); err != nil {
		log.Errorf("[CartHandler-2] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-3] AddToCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-4] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	userID := jwtUserData.UserID

	reqEntity := entity.CartItem{
		ProductID: request.ProductID,
		Quantity:  request.Quantity,
	}

	err = ch.CartService.AddToCart(ctx, userID, reqEntity)
	if err != nil {
		log.Errorf("[CartHandler-5] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// GetCart implements CartHandlerInterface.
func (ch *CartHandler) GetCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		respList    = []response.CartResponse{}
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] GetCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] GetCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID
	items, err := ch.CartService.GetCartByUserID(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler-3] GetCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	for _, item := range items {
		product, err := ch.ProductService.GetByID(ctx, item.ProductID)
		if err != nil {
			log.Errorf("[CartHandler-4] GetCart: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}

		respList = append(respList, response.CartResponse{
			ID:            item.ProductID,
			ProductName:   product.Name,
			ProductImage:  product.Image,
			ProductStatus: product.Status,
			SalePrice:     int64(product.SalePrice),
			Quantity:      item.Quantity,
			Unit:          product.Unit,
			Weight:        int64(product.Weight),
		})
	}

	resp.Message = "success"
	resp.Data = respList
	return c.JSON(http.StatusOK, resp)
}

// RemoveFromCart implements CartHandlerInterface.
func (ch *CartHandler) RemoveFromCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] RemoveFromCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] RemoveFromCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID
	productID := c.QueryParam("product_id")
	if productID == "" {
		log.Errorf("[CartHandler-3] RemoveFromCart: %s", "product_id is required")
		resp.Message = "product_id is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	prodID, err := conv.StringToInt64(productID)

	err = ch.CartService.RemoveFromCart(ctx, userID, prodID)
	if err != nil {
		log.Errorf("[CartHandler-4] RemoveFromCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

func NewCartHandler(e *echo.Echo, cfg *config.Config, cartService service.CartServiceInterface, productService service.ProductServiceInterface) CartHandlerInterface {
	cartHandler := &CartHandler{
		CartService:    cartService,
		ProductService: productService,
	}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg)
	authGroup := e.Group("/auth", mid.CheckToken())
	authGroup.POST("/cart", cartHandler.AddToCart)
	authGroup.GET("/cart", cartHandler.GetCart)
	authGroup.DELETE("/cart", cartHandler.RemoveFromCart)
	authGroup.DELETE("/cart/all", cartHandler.RemoveAllCart)
	return cartHandler
}
