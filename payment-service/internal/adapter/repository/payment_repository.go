package repository

import (
	"context"
	"errors"
	"math"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type PaymentRepositoryInterface interface {
	CreatePayment(ctx context.Context, payment entity.PaymentEntity) error
	CreatePaymentInTx(ctx context.Context, tx *gorm.DB, payment entity.PaymentEntity) error
	LogPayment(ctx context.Context, paymentID uint, status string) error
	UpdateStatusByOrderCode(ctx context.Context, orderID uint, status string) error
	GetAll(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error)
	GetDetail(ctx context.Context, paymentID uint) (*entity.PaymentEntity, error)
	GetByOrderID(ctx context.Context, orderID uint) error
}

type paymentRepository struct {
	db *gorm.DB
}

// GetByOrderID implements PaymentRepositoryInterface.
func (p *paymentRepository) GetByOrderID(ctx context.Context, orderID uint) error {
	modelPayment := model.Payment{}

	if err := p.db.Where("order_id = ?", orderID).First(&modelPayment).Error; err != nil {
		log.Errorf("[PaymentRepository-1] GetByOrderID: %v", err)
		return err
	}

	return nil
}

// GetDetail implements PaymentRepositoryInterface.
func (p *paymentRepository) GetDetail(ctx context.Context, paymentID uint) (*entity.PaymentEntity, error) {
	modelPayment := model.Payment{}

	if err := p.db.Where("id = ?", paymentID).First(&modelPayment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[PaymentRepository-1] GetDetail: No payment found")
			return nil, err
		}
		log.Errorf("[PaymentRepository-1] GetDetail: %v", err)
		return nil, err
	}

	return &entity.PaymentEntity{
		ID:               modelPayment.ID,
		OrderID:          modelPayment.OrderID,
		UserID:           modelPayment.UserID,
		PaymentMethod:    modelPayment.PaymentMethod,
		PaymentStatus:    modelPayment.PaymentStatus,
		PaymentGatewayID: *modelPayment.PaymentGatewayID,
		GrossAmount:      modelPayment.GrossAmount,
		PaymentURL:       *modelPayment.PaymentURL,
		PaymentAt:        modelPayment.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GetAll implements PaymentRepositoryInterface.
func (p *paymentRepository) GetAll(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error) {
	modelPayments := []model.Payment{}
	var countData int64
	offset := (req.Page - 1) * req.Limit

	sqlMain := p.db.
		Where("payment_method ILIKE ? OR payment_status ILIKE ?", "%"+req.Search+"%", "%"+req.Status+"%")

	if req.UserID != 0 {
		sqlMain = sqlMain.Where("user_id = ?", req.UserID)
	}

	if err := sqlMain.Model(&modelPayments).Count(&countData).Error; err != nil {
		log.Errorf("[PaymentRepository-1] GetAll: %v", err)
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(req.Limit)))
	if err := sqlMain.Order("created_at DESC").Limit(int(req.Limit)).Offset(int(offset)).Find(&modelPayments).Error; err != nil {
		log.Errorf("[PaymentRepository-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelPayments) == 0 {
		err := errors.New("404")
		log.Infof("[PaymentRepository-3] GetAll: No payment found")
		return nil, 0, 0, err
	}

	entities := []entity.PaymentEntity{}
	for _, val := range modelPayments {
		entities = append(entities, entity.PaymentEntity{
			ID:               val.ID,
			OrderID:          val.OrderID,
			UserID:           val.UserID,
			PaymentMethod:    val.PaymentMethod,
			PaymentStatus:    val.PaymentStatus,
			PaymentGatewayID: *val.PaymentGatewayID,
			GrossAmount:      val.GrossAmount,
			PaymentURL:       *val.PaymentURL,
		})
	}

	return entities, countData, int64(totalPage), nil
}

// UpdateStatusByOrderCode implements PaymentRepositoryInterface.
func (p *paymentRepository) UpdateStatusByOrderCode(ctx context.Context, orderID uint, status string) error {
	modelPayment := model.Payment{}

	if err := p.db.Where("order_id = ?", orderID).First(&modelPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] UpdateStatusByOrderCode-1: %v", err)
		return err
	}

	modelPayment.PaymentStatus = status

	if err := p.db.Save(&modelPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] UpdateStatusByOrderCode-2: %v", err)
		return err
	}

	return nil
}

// LogPayment implements PaymentRepositoryInterface.
func (p *paymentRepository) LogPayment(ctx context.Context, paymentID uint, status string) error {
	logPayment := model.PaymentLog{
		PaymentID: paymentID,
		Status:    status,
	}

	if err := p.db.Create(&logPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] LogPayment-1: %v", err)
		return err
	}

	return nil
}

// CreatePayment implements PaymentRepositoryInterface.
func (p *paymentRepository) CreatePayment(ctx context.Context, payment entity.PaymentEntity) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return p.CreatePaymentInTx(ctx, tx, payment)
	})
}

// CreatePaymentInTx creates payment and audit log on the given transaction.
func (p *paymentRepository) CreatePaymentInTx(ctx context.Context, tx *gorm.DB, payment entity.PaymentEntity) error {
	modelPayment := model.Payment{
		OrderID:          payment.OrderID,
		UserID:           payment.UserID,
		PaymentMethod:    payment.PaymentMethod,
		PaymentStatus:    payment.PaymentStatus,
		PaymentGatewayID: &payment.PaymentGatewayID,
		GrossAmount:      payment.GrossAmount,
		PaymentURL:       &payment.PaymentURL,
	}

	if err := tx.WithContext(ctx).Create(&modelPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] CreatePaymentInTx-1: %v", err)
		return err
	}

	logPayment := model.PaymentLog{
		PaymentID: modelPayment.ID,
		Status:    modelPayment.PaymentStatus,
	}
	if err := tx.WithContext(ctx).Create(&logPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] CreatePaymentInTx-2: %v", err)
		return err
	}

	return nil
}

func NewPaymentRepository(db *gorm.DB) PaymentRepositoryInterface {
	return &paymentRepository{db: db}
}
