package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"order-service/config"
	httpclient "order-service/internal/adapter/http_client"
	"order-service/internal/adapter/repository"
	"order-service/internal/core/domain/entity"
	"order-service/internal/core/domain/model"
	"order-service/utils/conv"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type OrderServiceInterface interface {
	GetAll(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error)
	GetByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error)
	CreateOrder(ctx context.Context, req entity.OrderEntity, accessToken string) (int64, error)
	UpdateStatus(ctx context.Context, req entity.OrderEntity, accessToken string) error
	GetAllCustomer(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error)
	GetDetailCustomer(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error)
	DeleteByID(ctx context.Context, orderID int64) error
	GetOrderByOrderCode(ctx context.Context, orderCode, accessToken string) (*entity.OrderEntity, error)
	GetPublicOrderIDByOrderCode(ctx context.Context, orderCode string) (int64, error)
}

type orderService struct {
	db         *gorm.DB
	repo       repository.OrderRepositoryInterface
	outboxRepo repository.OutboxRepositoryInterface
	cfg        *config.Config
	httpClient httpclient.HttpClient
	elasticRepo repository.ElasticRepositoryInterface
	logger     zerolog.Logger
}

func (o *orderService) insertOutbox(tx *gorm.DB, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	event := &model.OutboxEvent{
		ID:        uuid.New().String(),
		EventType: eventType,
		Payload:   string(data),
		Status:    model.OutboxStatusPending,
	}
	return o.outboxRepo.Insert(tx, event)
}

// GetPublicOrderIDByOrderCode implements OrderServiceInterface.
func (o *orderService) GetPublicOrderIDByOrderCode(ctx context.Context, orderCode string) (int64, error) {
	result, err := o.repo.GetOrderByOrderCode(ctx, orderCode)
	if err != nil {
		log.Errorf("[OrderService-1] GetPublicOrderIDByOrderCode: %v", err)
		return 0, err
	}

	return result.ID, nil
}

// GetOrderByOrderCode implements OrderServiceInterface.
func (o *orderService) GetOrderByOrderCode(ctx context.Context, orderCode string, accessToken string) (*entity.OrderEntity, error) {
	result, err := o.repo.GetOrderByOrderCode(ctx, orderCode)
	if err != nil {
		log.Errorf("[OrderService-1] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-2] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	isCustomer := false
	if token["role_name"].(string) != "Super Admin" {
		isCustomer = true
	}

	userResponse, err := o.httpClientUserService(result.BuyerId, token["token"].(string), isCustomer)
	if err != nil {
		log.Errorf("[OrderService-3] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	result.BuyerName = userResponse.Name
	result.BuyerEmail = userResponse.Email
	result.BuyerPhone = userResponse.Phone
	result.BuyerAddress = userResponse.Address

	for key, val := range result.OrderItems {
		productResponse, err := o.httpClientProductService(val.ProductID, token["token"].(string), isCustomer)
		if err != nil {
			log.Errorf("[OrderService-4] GetOrderByOrderCode: %v", err)
			return nil, err
		}

		result.OrderItems[key].ProductImage = productResponse.ProductImage
		result.OrderItems[key].ProductName = productResponse.ProductName
		result.OrderItems[key].Price = int64(productResponse.SalePrice)
	}

	return result, nil
}

// DeleteByID implements OrderServiceInterface.
func (o *orderService) DeleteByID(ctx context.Context, orderID int64) error {
	return o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := o.repo.WithTx(tx)
		if err := txRepo.DeleteOrder(ctx, orderID); err != nil {
			log.Errorf("[OrderService-1] DeleteByID: %v", err)
			return err
		}
		payload := map[string]string{
			"orderID": fmt.Sprintf("%d", orderID),
		}
		if err := o.insertOutbox(tx, model.EventOrderDeleted, payload); err != nil {
			o.logger.Error().Err(err).Msg("insert outbox delete event failed")
			return err
		}
		return nil
	})
}

// GetDetailCustomer implements OrderServiceInterface.
func (o *orderService) GetDetailCustomer(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error) {
	result, err := o.repo.GetByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService-1] GetByID: %v", err)
		return nil, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-2] GetByID: %v", err)
		return nil, err
	}

	userResponse, err := o.httpClientUserService(result.BuyerId, token["token"].(string), true)
	if err != nil {
		log.Errorf("[OrderService-3] GetByID: %v", err)
		return nil, err
	}

	result.BuyerName = userResponse.Name
	result.BuyerEmail = userResponse.Email
	result.BuyerPhone = userResponse.Phone
	result.BuyerAddress = userResponse.Address

	for key, val := range result.OrderItems {
		productResponse, err := o.httpClientProductService(val.ProductID, token["token"].(string), true)
		if err != nil {
			log.Errorf("[OrderService-3] GetByID: %v", err)
			return nil, err
		}

		result.OrderItems[key].ProductImage = productResponse.ProductImage
		if productResponse.Child != nil {
			result.OrderItems[key].ProductImage = productResponse.Child[0].Image
		}
		result.OrderItems[key].ProductName = productResponse.ProductName
		result.OrderItems[key].Price = int64(productResponse.SalePrice)
		result.OrderItems[key].ProductWeight = int64(productResponse.Weight)
		result.OrderItems[key].ProductUnit = productResponse.Unit
	}

	return result, nil
}

// GetAllCustomer implements OrderServiceInterface.
func (o *orderService) GetAllCustomer(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error) {
	results, count, total, err := o.elasticRepo.SearchOrderElasticByBuyerId(ctx, queryString, queryString.BuyerID)
	if err == nil {
		return results, count, total, nil
	}
	log.Errorf("[OrderService-1] GetAllCustomer: %v", err)

	results, count, total, err = o.repo.GetAll(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService-2] GetAllCustomer: %v", err)
		return nil, 0, 0, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-3] GetAllCustomer: %v", err)
		return nil, 0, 0, err
	}

	for key, val := range results {
		userResponse, err := o.httpClientUserService(val.BuyerId, token["token"].(string), true)
		if err != nil {
			log.Errorf("[OrderService-4] GetAllCustomer: %v", err)
			return nil, 0, 0, err
		}

		results[key].BuyerName = userResponse.Name
		results[key].BuyerEmail = userResponse.Email
		results[key].BuyerPhone = userResponse.Phone
		results[key].BuyerAddress = userResponse.Address

		for key2, res := range val.OrderItems {

			productResponse, err := o.httpClientProductService(res.ProductID, token["token"].(string), true)
			if err != nil {
				log.Errorf("[OrderService-5] GetAllCustomer: %v", err)
				return nil, 0, 0, err
			}

			val.OrderItems[key2].ProductImage = productResponse.ProductImage
			val.OrderItems[key2].ProductName = productResponse.ProductName
			val.OrderItems[key2].Price = int64(productResponse.SalePrice)
			val.OrderItems[key2].Quantity = res.Quantity
			val.OrderItems[key2].ProductUnit = productResponse.Unit
			val.OrderItems[key2].ProductWeight = int64(productResponse.Weight)
		}
	}

	return results, count, total, nil
}

// UpdateStatus implements OrderServiceInterface.
func (o *orderService) UpdateStatus(ctx context.Context, req entity.OrderEntity, accessToken string) error {
	var token map[string]interface{}
	err := json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-1] UpdateStatus: %v", err)
		return err
	}

	existing, err := o.repo.GetByID(ctx, req.ID)
	if err != nil {
		log.Errorf("[OrderService-2] UpdateStatus: %v", err)
		return err
	}

	userResponse, err := o.httpClientUserService(existing.BuyerId, token["token"].(string), false)
	if err != nil {
		log.Errorf("[OrderService-3] UpdateStatus: %v", err)
		return err
	}

	return o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := o.repo.WithTx(tx)
		buyerID, statusOrder, orderCode, err := txRepo.UpdateStatus(ctx, req)
		if err != nil {
			log.Errorf("[OrderService-4] UpdateStatus: %v", err)
			return err
		}

		message := fmt.Sprintf("Hello,\n\nYour order with ID %s has been updated to status: %s.\n\nThank you for shopping with us!", orderCode, statusOrder)

		emailPayload := map[string]interface{}{
			"receiver_email":    userResponse.Email,
			"message":           message,
			"subject":           "Update Status Order",
			"type":              "UPDATE_STATUS",
			"receiver_id":       buyerID,
			"notification_type": "EMAIL",
		}
		if err := o.insertOutbox(tx, model.EventNotificationOrderEmail, emailPayload); err != nil {
			o.logger.Error().Err(err).Msg("insert outbox email notification failed")
			return err
		}

		pushPayload := map[string]interface{}{
			"receiver_email":    "",
			"message":           message,
			"subject":           "Update Status Order",
			"type":              "UPDATE_STATUS",
			"receiver_id":       buyerID,
			"notification_type": "PUSH",
		}
		if err := o.insertOutbox(tx, model.EventNotificationOrderPush, pushPayload); err != nil {
			o.logger.Error().Err(err).Msg("insert outbox push notification failed")
			return err
		}

		esPayload := map[string]string{
			"orderID": fmt.Sprintf("%d", req.ID),
			"status":  statusOrder,
		}
		if err := o.insertOutbox(tx, model.EventOrderStatusES, esPayload); err != nil {
			o.logger.Error().Err(err).Msg("insert outbox ES status failed")
			return err
		}

		return nil
	})
}

// CreateOrder implements OrderServiceInterface.
func (o *orderService) CreateOrder(ctx context.Context, req entity.OrderEntity, accessToken string) (int64, error) {
	req.OrderCode = conv.GenerateOrderCode()
	shippingFee := 0
	if req.ShippingType == "Delivery" {
		shippingFee = 5000
	}
	req.ShippingFee = int64(shippingFee)
	req.Status = "Pending"
	orderID, err := o.repo.CreateOrder(ctx, req)
	if err != nil {
		log.Errorf("[OrderService-1] CreateOrder: %v", err)
		return 0, err
	}

	resultData, err := o.GetByID(ctx, orderID, accessToken)
	if err != nil {
		log.Errorf("[OrderService-2] CreateOrder: enrich order failed: %v", err)
		fallback, ferr := o.repo.GetByID(ctx, orderID)
		if ferr != nil {
			resultData = nil
		} else {
			resultData = fallback
		}
	}

	err = o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if resultData != nil {
			if err := o.insertOutbox(tx, model.EventOrderCreated, resultData); err != nil {
				o.logger.Error().Err(err).Msg("insert outbox order.created failed")
				return err
			}
		}
		for _, item := range req.OrderItems {
			p := entity.PublishOrderItemEntity{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
			if err := o.insertOutbox(tx, model.EventOrderStockUpdate, p); err != nil {
				o.logger.Error().Err(err).Msg("insert outbox stock update failed")
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("[OrderService-3] CreateOrder outbox: %v", err)
		return orderID, err
	}

	return orderID, nil
}

// GetByID implements OrderServiceInterface.
func (o *orderService) GetByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error) {
	result, err := o.repo.GetByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService-1] GetByID: %v", err)
		return nil, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-2] GetByID: %v", err)
		return nil, err
	}

	isCustomer := false
	if token["role_name"].(string) != "Super Admin" {
		isCustomer = true
	}

	userResponse, err := o.httpClientUserService(result.BuyerId, token["token"].(string), isCustomer)
	if err != nil {
		log.Errorf("[OrderService-2] GetByID: %v", err)
		return nil, err
	}

	result.BuyerName = userResponse.Name
	result.BuyerEmail = userResponse.Email
	result.BuyerPhone = userResponse.Phone
	result.BuyerAddress = userResponse.Address

	for key, val := range result.OrderItems {
		productResponse, err := o.httpClientProductService(val.ProductID, token["token"].(string), isCustomer)
		if err != nil {
			log.Errorf("[OrderService-3] GetByID: %v", err)
			return nil, err
		}

		result.OrderItems[key].ProductImage = productResponse.ProductImage
		result.OrderItems[key].ProductName = productResponse.ProductName
		result.OrderItems[key].Price = int64(productResponse.SalePrice)
	}

	return result, nil
}

// GetAll implements OrderServiceInterface.
func (o *orderService) GetAll(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error) {
	results, count, total, err := o.elasticRepo.SearchOrderElastic(ctx, queryString)
	if err == nil {
		return results, count, total, nil
	}
	log.Errorf("[OrderService-1] GetAll: %v", err)

	results, count, total, err = o.repo.GetAll(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[OrderService-3] GetAll: %v", err)
		return nil, 0, 0, err
	}

	isCustomer := false
	if token["role_name"].(string) != "Super Admin" {
		isCustomer = true
	}

	for key, val := range results {

		userResponse, err := o.httpClientUserService(val.BuyerId, token["token"].(string), isCustomer)
		if err != nil {
			log.Errorf("[OrderService-4] GetAll: %v", err)
			return nil, 0, 0, err
		}
		results[key].BuyerName = userResponse.Name

		for key2, res := range val.OrderItems {

			productResponse, err := o.httpClientProductService(res.ProductID, token["token"].(string), isCustomer)
			if err != nil {
				log.Errorf("[OrderService-5] GetAll: %v", err)
				return nil, 0, 0, err
			}

			val.OrderItems[key2].ProductImage = productResponse.ProductImage
		}
	}

	return results, count, total, nil
}

func (o *orderService) httpClientUserService(userID int64, accessToken string, isCustomer bool) (*entity.CustomerResponseEntity, error) {
	baseUrlUser := fmt.Sprintf("%s/%s", o.cfg.App.UserServiceUrl, "admin/customers/"+strconv.FormatInt(userID, 10))
	if isCustomer {
		baseUrlUser = fmt.Sprintf("%s/%s", o.cfg.App.UserServiceUrl, "auth/profile")
	}
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataUser, err := o.httpClient.CallURL("GET", baseUrlUser, header, nil)
	if err != nil {
		log.Errorf("[OrderService-1] httpClientUserService: %v", err)
		return nil, err
	}

	defer dataUser.Body.Close()

	body, err := io.ReadAll(dataUser.Body)
	if err != nil {
		log.Errorf("[OrderService-2] httpClientUserService: %v", err)
		return nil, err
	}

	var userResponse entity.UserHttpClientResponse
	err = json.Unmarshal(body, &userResponse)
	if err != nil {
		log.Errorf("[OrderService-3] httpClientUserService: %v", err)
		return nil, err
	}

	return &userResponse.Data, nil
}

func (o *orderService) httpClientProductService(productID int64, accessToken string, isCustomer bool) (*entity.ProductResponseEntity, error) {
	baseUrlProduct := fmt.Sprintf("%s/%s", o.cfg.App.ProductServiceUrl, "admin/products/"+strconv.FormatInt(productID, 10))
	if isCustomer {
		baseUrlProduct = fmt.Sprintf("%s/%s", o.cfg.App.ProductServiceUrl, "products/home/"+strconv.FormatInt(productID, 10))
	}
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataProduct, err := o.httpClient.CallURL("GET", baseUrlProduct, header, nil)
	if err != nil {
		log.Errorf("[OrderService-1] httpClientProductService: %v", err)
		return nil, err
	}

	defer dataProduct.Body.Close()

	body, err := io.ReadAll(dataProduct.Body)
	if err != nil {
		log.Errorf("[OrderService-2] httpClientProductService: %v", err)
		return nil, err
	}

	var productResponse entity.ProductHttpClientResponse
	err = json.Unmarshal(body, &productResponse)
	if err != nil {
		log.Errorf("[OrderService-3] httpClientProductService: %v", err)
		return nil, err
	}

	return &productResponse.Data, nil
}

func NewOrderService(
	db *gorm.DB,
	repo repository.OrderRepositoryInterface,
	outboxRepo repository.OutboxRepositoryInterface,
	cfg *config.Config,
	httpClient httpclient.HttpClient,
	elasticRepo repository.ElasticRepositoryInterface,
	logger zerolog.Logger,
) OrderServiceInterface {
	return &orderService{
		db:          db,
		repo:        repo,
		outboxRepo:  outboxRepo,
		cfg:         cfg,
		httpClient:  httpClient,
		elasticRepo: elasticRepo,
		logger:      logger.With().Str("component", "order_service").Logger(),
	}
}
