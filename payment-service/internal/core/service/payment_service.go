package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"payment-service/config"
	httpclient "payment-service/internal/adapter/http_client"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"
	"strconv"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type PaymentServiceInterface interface {
	ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error)
	UpdateStatusByOrderCode(ctx context.Context, orderCode, status string) error
	GetAll(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error)
	GetDetail(ctx context.Context, paymentID uint, accessToken string) (*entity.PaymentEntity, error)
}

type paymentService struct {
	repo                repository.PaymentRepositoryInterface
	db                  *gorm.DB
	outboxRepo          repository.OutboxRepositoryInterface
	httpClientToService httpclient.HttpClientToService
	midtrans            httpclient.MidtransClientInterface
	cfg                 *config.Config
}

// GetDetail implements PaymentServiceInterface.
func (p *paymentService) GetDetail(ctx context.Context, paymentID uint, accessToken string) (*entity.PaymentEntity, error) {
	result, err := p.repo.GetDetail(ctx, paymentID)
	if err != nil {
		log.Errorf("[PaymentService] GetDetail-1: %v", err)
		return nil, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[PaymentService] GetDetail-2: %v", err)
		return nil, err
	}

	userID := int64(result.UserID)
	if token["role_name"].(string) == "Super Admin" {
		userID = 0
	}

	orderDetail, err := p.httpClientOrderService(int64(result.OrderID), token["token"].(string))
	if err != nil {
		log.Errorf("[PaymentService] GetDetail-3: %v", err)
		return nil, err
	}

	isAdmin := false
	if token["role_name"].(string) == "Super Admin" {
		isAdmin = true
	}

	userDetail, err := p.httpClientUserService(token["token"].(string), userID, isAdmin)
	if err != nil {
		log.Errorf("[PaymentService] GetDetail-4: %v", err)
		return nil, err
	}

	result.CustomerName = userDetail.Name
	result.CustomerEmail = userDetail.Email
	result.CustomerAddress = userDetail.Address

	result.OrderCode = orderDetail.OrderCode
	result.OrderShippingType = orderDetail.ShippingType
	result.OrderAt = orderDetail.OrderDatetime
	result.OrderRemarks = orderDetail.Remarks

	return result, nil
}

// GetAll implements PaymentServiceInterface.
func (p *paymentService) GetAll(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error) {
	results, count, total, err := p.repo.GetAll(ctx, req)
	if err != nil {
		log.Errorf("[PaymentService] GetAll-1: %v", err)
		return nil, 0, 0, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Errorf("[PaymentService] GetAll-2: %v", err)
		return nil, 0, 0, err
	}
	for key, val := range results {
		orderDetail, err := p.httpClientOrderService(int64(val.OrderID), token["token"].(string))
		if err != nil {
			log.Errorf("[PaymentService] GetAll-3: %v", err)
			return nil, 0, 0, err
		}
		results[key].OrderCode = orderDetail.OrderCode
		results[key].OrderShippingType = orderDetail.ShippingType
	}

	return results, count, total, nil
}

// UpdateStatusByOrderCode implements PaymentServiceInterface.
func (p *paymentService) UpdateStatusByOrderCode(ctx context.Context, orderCode string, status string) error {
	orderDetailID, err := p.httpClientPublicOrderIDByCodeService(orderCode)
	if err != nil {
		log.Errorf("[PaymentService] UpdateStatusByOrderCode-1: %v", err)
		return err
	}

	if err = p.repo.UpdateStatusByOrderCode(ctx, uint(orderDetailID), status); err != nil {
		log.Errorf("[PaymentService] UpdateStatusByOrderCode-2: %v", err)
		return err
	}

	return nil
}

// ProcessPayment implements PaymentServiceInterface.
func (p *paymentService) ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error) {
	err := p.repo.GetByOrderID(ctx, uint(payment.OrderID))
	if err == nil {
		log.Infof("[PaymentService] ProcessPayment-1: Payment already exists")
		return nil, errors.New("Payment already exists")
	}

	if payment.PaymentMethod == "cod" {
		payment.PaymentStatus = "Success"

		if err := p.createPaymentWithOutbox(ctx, payment); err != nil {
			log.Errorf("[PaymentService] ProcessPayment-2: %v", err)
			return nil, err
		}

		return &payment, nil
	}

	if payment.PaymentMethod == "midtrans" {
		var token map[string]interface{}
		err := json.Unmarshal([]byte(accessToken), &token)
		if err != nil {
			log.Errorf("[PaymentService] ProcessPayment-4: %v", err)
			return nil, err
		}

		isAdmin := false
		if token["role_name"].(string) == "Super Admin" {
			isAdmin = true
		}

		userResponse, err := p.httpClientUserService(token["token"].(string), int64(payment.UserID), isAdmin)
		if err != nil {
			log.Errorf("[PaymentService] ProcessPayment-5: %v", err)
			return nil, err
		}

		orderDetail, err := p.httpClientOrderService(int64(payment.OrderID), token["token"].(string))
		if err != nil {
			log.Errorf("[PaymentService] ProcessPayment-6: %v", err)
			return nil, err
		}

		transactionID, err := p.midtrans.CreateTransaction(orderDetail.OrderCode, int64(payment.GrossAmount), userResponse.Name, userResponse.Email)
		if err != nil {
			log.Errorf("[PaymentService] ProcessPayment-7: %v", err)
			return nil, err
		}
		payment.PaymentStatus = "Pending"
		payment.PaymentGatewayID = transactionID

		if err := p.createPaymentWithOutbox(ctx, payment); err != nil {
			log.Errorf("[PaymentService] ProcessPayment-8: %v", err)
			return nil, err
		}

		return &payment, nil
	}

	return nil, errors.New("Invalid payment method")
}

// createPaymentWithOutbox persists payment and enqueues a RabbitMQ message via outbox in one transaction.
func (p *paymentService) createPaymentWithOutbox(ctx context.Context, payment entity.PaymentEntity) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := p.repo.CreatePaymentInTx(ctx, tx, payment); err != nil {
			return err
		}
		return p.insertPaymentSuccessOutbox(tx, payment)
	})
}

func (p *paymentService) insertPaymentSuccessOutbox(tx *gorm.DB, payment entity.PaymentEntity) error {
	if p.cfg.PublisherName.PaymentSuccess == "" {
		return nil
	}

	payload := map[string]string{
		"orderID":       fmt.Sprintf("%d", payment.OrderID),
		"paymentMethod": payment.PaymentMethod,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	event := &model.OutboxEvent{
		EventType: p.cfg.PublisherName.PaymentSuccess,
		Payload:   string(body),
		Status:    model.OutboxStatusPending,
	}
	return p.outboxRepo.Insert(tx, event)
}

func (p *paymentService) httpClientOrderService(orderId int64, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+strconv.FormatInt(orderId, 10))
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataOrder, err := p.httpClientToService.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderService-1: %v", err)
		return nil, err
	}

	defer dataOrder.Body.Close()

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderService-2: %v", err)
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderService-3: %v", err)
		return nil, err
	}

	return &orderDetail.Data, nil
}

func (p *paymentService) httpClientUserService(accessToken string, userID int64, isAdmin bool) (*entity.ProfileHttpResponse, error) {
	baseUrlUser := fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "auth/profile")
	if isAdmin {
		baseUrlUser = fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "admin/customers/"+strconv.FormatInt(userID, 10))
	}
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataUser, err := p.httpClientToService.CallURL("GET", baseUrlUser, header, nil)
	if err != nil {
		log.Errorf("[PaymentService] httpClientUserService-1: %v", err)
		return nil, err
	}

	defer dataUser.Body.Close()

	body, err := io.ReadAll(dataUser.Body)
	if err != nil {
		log.Errorf("[PaymentService] httpClientUserService-2: %v", err)
		return nil, err
	}

	var userResponse entity.UserHttpClientResponse
	err = json.Unmarshal([]byte(body), &userResponse)
	if err != nil {
		log.Errorf("[PaymentService] httpClientUserService-3: %v", err)
		return nil, err
	}

	return &userResponse.Data, nil
}

func (p *paymentService) httpClientOrderByCodeService(orderCode string, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+orderCode+"/code")
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataOrder, err := p.httpClientToService.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-1: %v", err)
		return nil, err
	}

	defer dataOrder.Body.Close()

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-2: %v", err)
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-3: %v", err)
		return nil, err
	}

	return &orderDetail.Data, nil
}

func (p *paymentService) httpClientPublicOrderIDByCodeService(orderCode string) (int64, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "public/orders/"+orderCode+"/code")
	header := map[string]string{
		"Accept": "application/json",
	}
	dataOrder, err := p.httpClientToService.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-1: %v", err)
		return 0, err
	}

	defer dataOrder.Body.Close()

	if dataOrder.StatusCode != 200 {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-3: %v", err)
		return 0, errors.New("Order not found")
	}

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-2: %v", err)
		return 0, err
	}

	var orderDetail entity.GetOrderIDByCodeResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Errorf("[PaymentService] httpClientOrderByCodeService-4: %v", err)
		return 0, err
	}

	return int64(orderDetail.Data.OrderID), nil
}

func NewPaymentService(
	db *gorm.DB,
	repo repository.PaymentRepositoryInterface,
	outboxRepo repository.OutboxRepositoryInterface,
	cfg *config.Config,
	httpClientToService httpclient.HttpClientToService,
	midtrans httpclient.MidtransClientInterface,
) PaymentServiceInterface {
	return &paymentService{
		repo:                repo,
		db:                  db,
		outboxRepo:          outboxRepo,
		httpClientToService: httpClientToService,
		midtrans:            midtrans,
		cfg:                 cfg,
	}
}
