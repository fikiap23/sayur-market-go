package handlers

import (
	"encoding/json"
	"net/http"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/handlers/request"
	"payment-service/internal/adapter/handlers/response"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/service"
	"payment-service/utils/conv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type PaymentHandlerInterface interface {
	Create(c echo.Context) error
	MidtranswebHookHandler(c echo.Context) error
	GetAllAdmin(c echo.Context) error
	GetAllCustomer(c echo.Context) error
	GetDetail(c echo.Context) error
}

type paymentHandler struct {
	paymentService service.PaymentServiceInterface
}

func NewPaymentHandler(paymentService service.PaymentServiceInterface, e *echo.Echo, cfg *config.Config) PaymentHandlerInterface {
	paymentHandler := &paymentHandler{
		paymentService: paymentService,
	}
	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg)
	e.POST("/payments/webhook", paymentHandler.MidtranswebHookHandler)
	authGroup := e.Group("auth", mid.CheckToken())
	authGroup.GET("/payments", paymentHandler.GetAllCustomer)
	authGroup.GET("/payments/:id", paymentHandler.GetDetail)
	authGroup.POST("/payments", paymentHandler.Create)

	adminGroup := e.Group("/admin", mid.CheckToken())
	adminGroup.GET("/payments", paymentHandler.GetAllAdmin)
	adminGroup.GET("/payments/:id", paymentHandler.GetDetail)

	return paymentHandler

}

func (ph *paymentHandler) GetDetail(c echo.Context) error {
	var (
		ctx   = c.Request().Context()
		resps = response.PaymentDetailResponse{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[PaymentHandler-1] GetAll: %s", "data token not found")
		return c.JSON(http.StatusNotFound, response.ResponseDefault("data token not found", nil))
	}

	paymentID := c.Param("id")
	if paymentID == "" {
		log.Errorf("[PaymentHandler-2] GetDetail: %s", "Payment ID is required")
		return c.JSON(http.StatusBadRequest, response.ResponseDefault("Payment ID is required", nil))
	}

	paymentIDInt, err := conv.StringToInt64(paymentID)
	if err != nil {
		log.Errorf("[PaymentHandler-3] GetDetail: %v", err)
		return c.JSON(http.StatusBadRequest, response.ResponseDefault(err.Error(), nil))
	}

	result, err := ph.paymentService.GetDetail(ctx, uint(paymentIDInt), user)
	if err != nil {
		log.Errorf("[PaymentHandler-4] GetDetail: %v", err)
		return c.JSON(http.StatusInternalServerError, response.ResponseDefault(err.Error(), nil))
	}

	resps.ID = int64(result.ID)
	resps.OrderCode = result.OrderCode
	resps.PaymentMethod = result.PaymentMethod
	resps.PaymentStatus = result.PaymentStatus
	resps.GrossAmount = result.GrossAmount
	resps.ShippingType = result.OrderShippingType
	resps.PaymentAt = result.PaymentAt
	resps.OrderAt = result.OrderAt
	resps.OrderRemarks = result.OrderRemarks
	resps.CustomerName = result.CustomerName
	resps.CustomerAddress = result.CustomerAddress

	return c.JSON(http.StatusOK, response.ResponseDefault("success", resps))
}

func (ph *paymentHandler) GetAllCustomer(c echo.Context) error {
	var (
		ctx         = c.Request().Context()
		resps       = []response.PaymentListResponse{}
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[PaymentHandler-1] GetAll: %s", "data token not found")
		return c.JSON(http.StatusNotFound, response.ResponseDefault("data token not found", nil))
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[PaymentHandler-2] GetAllCustomer: %v", err)
		return c.JSON(http.StatusBadRequest, response.ResponseDefault(err.Error(), nil))
	}

	userID := jwtUserData.UserID
	search := c.QueryParam("search")
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

	status := ""
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status = statusStr
	}

	orderBy := "created_at"
	if orderByStr := c.QueryParam("orderBy"); orderByStr != "" {
		orderBy = orderByStr
	}

	orderType := "desc"
	if orderTypeStr := c.QueryParam("orderType"); orderTypeStr != "" {
		orderType = orderTypeStr
	}

	reqEntity := entity.PaymentQueryStringRequest{
		Search:    search,
		Status:    status,
		Page:      page,
		Limit:     perPage,
		OrderBy:   orderBy,
		OrderType: orderType,
		UserID:    int64(userID),
	}

	results, count, total, err := ph.paymentService.GetAll(ctx, reqEntity, user)
	if err != nil {
		log.Errorf("[PaymentHandler-3] GetAll: %v", err)
		return c.JSON(http.StatusInternalServerError, response.ResponseDefault(err.Error(), nil))
	}

	for _, val := range results {
		resps = append(resps, response.PaymentListResponse{
			ID:            uint64(val.ID),
			OrderCode:     val.OrderCode,
			PaymentStatus: val.PaymentStatus,
			PaymentMethod: val.PaymentMethod,
			GrossAmount:   val.GrossAmount,
			ShippingType:  val.OrderShippingType,
		})
	}

	return c.JSON(http.StatusOK, response.ResponseSuccessWithPagination("success", resps, page, count, total, perPage))
}

func (ph *paymentHandler) GetAllAdmin(c echo.Context) error {
	var (
		ctx   = c.Request().Context()
		resps = []response.PaymentListResponse{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[PaymentHandler-1] GetAll: %s", "data token not found")
		return c.JSON(http.StatusNotFound, response.ResponseDefault("data token not found", nil))
	}

	userID := 0
	search := c.QueryParam("search")
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

	status := ""
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status = statusStr
	}

	orderBy := "created_at"
	if orderByStr := c.QueryParam("orderBy"); orderByStr != "" {
		orderBy = orderByStr
	}

	orderType := "desc"
	if orderTypeStr := c.QueryParam("orderType"); orderTypeStr != "" {
		orderType = orderTypeStr
	}

	reqEntity := entity.PaymentQueryStringRequest{
		Search:    search,
		Status:    status,
		Page:      page,
		Limit:     perPage,
		OrderBy:   orderBy,
		OrderType: orderType,
		UserID:    int64(userID),
	}

	results, count, total, err := ph.paymentService.GetAll(ctx, reqEntity, user)
	if err != nil {
		log.Errorf("[PaymentHandler-3] GetAll: %v", err)
		return c.JSON(http.StatusInternalServerError, response.ResponseDefault(err.Error(), nil))
	}

	for _, val := range results {
		resps = append(resps, response.PaymentListResponse{
			ID:            uint64(val.ID),
			OrderCode:     val.OrderCode,
			PaymentStatus: val.PaymentStatus,
			PaymentMethod: val.PaymentMethod,
			GrossAmount:   val.GrossAmount,
			ShippingType:  val.OrderShippingType,
		})
	}

	return c.JSON(http.StatusOK, response.ResponseSuccessWithPagination("success", resps, page, count, total, perPage))
}

func (ph *paymentHandler) MidtranswebHookHandler(c echo.Context) error {
	var notificationPayload map[string]interface{}
	if err := c.Bind(&notificationPayload); err != nil {
		log.Errorf("[PaymentHandler-1] MidtranswebHookHandler: %v", err)
		return c.JSON(http.StatusBadRequest, response.ResponseDefault(err.Error(), nil))
	}

	transactionStatus := notificationPayload["transaction_status"].(string)
	orderID := notificationPayload["order_id"].(string)

	newStatus := ""
	switch transactionStatus {
	case "capture", "settlement":
		newStatus = "success"
	case "deny", "cancel", "expire":
		newStatus = "failed"
	case "pending":
		newStatus = "pending"
	default:
		newStatus = "unknown"
	}

	if err := ph.paymentService.UpdateStatusByOrderCode(c.Request().Context(), orderID, newStatus); err != nil {
		log.Errorf("[PaymentHandler-3] MidtranswebHookHandler: %v", err)
		return c.JSON(http.StatusInternalServerError, response.ResponseDefault(err.Error(), nil))
	}

	return c.JSON(http.StatusOK, response.ResponseDefault("success", nil))
}

func (p *paymentHandler) Create(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req = request.PaymentRequest{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[PaymentHandler-1] Create: %s", "data token not found")
		return c.JSON(http.StatusUnauthorized, response.ResponseDefault("data token not found", nil))
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[PaymentHandler-2] Create: %v", err)
		return c.JSON(http.StatusBadRequest, response.ResponseDefault(err.Error(), nil))
	}

	if err := c.Validate(&req); err != nil {
		log.Errorf("[PaymentHandler-3] Create: %v", err)
		return c.JSON(http.StatusUnprocessableEntity, response.ResponseDefault(err.Error(), nil))
	}

	paymentEntity := entity.PaymentEntity{
		OrderID:       req.OrderID,
		PaymentMethod: req.PaymentMethod,
		GrossAmount:   float64(req.GrossAmount),
		UserID:        req.UserID,
		Remarks:       req.Remarks,
	}

	result, err := p.paymentService.ProcessPayment(ctx, paymentEntity, user)
	if err != nil {
		log.Errorf("[PaymentHandler-4] Create: %v", err)
		return c.JSON(http.StatusInternalServerError, response.ResponseDefault(err.Error(), nil))
	}

	responPayment := map[string]interface{}{
		"payment_token": result.PaymentGatewayID,
	}

	return c.JSON(http.StatusCreated, response.ResponseDefault("success", responPayment))
}
