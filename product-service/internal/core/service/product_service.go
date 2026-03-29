package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type ProductServiceInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) error
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	// ReindexAll fetches every product from PG and bulk-indexes them into ES.
	ReindexAll(ctx context.Context) (int, error)
}

type productService struct {
	db        *gorm.DB
	repo      repository.ProductRepositoryInterface
	outboxRepo repository.OutboxRepositoryInterface
	repoCat   repository.CategoryRepositoryInterface
	logger    zerolog.Logger
}

func (p *productService) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.SearchProducts(ctx, query)
}

// Create inserts the product and an outbox event inside a single DB transaction.
// The outbox worker will later pick up the event and publish it to RabbitMQ.
func (p *productService) Create(ctx context.Context, req entity.ProductEntity) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := p.repo.WithTx(tx)

		productID, err := txRepo.Create(ctx, req)
		if err != nil {
			p.logger.Error().Err(err).Msg("create product failed")
			return err
		}

		product, err := txRepo.GetByID(ctx, productID)
		if err != nil {
			p.logger.Error().Err(err).Int64("product_id", productID).Msg("fetch created product failed")
			return err
		}
		product, err = p.productWithCategoryName(ctx, product)
		if err != nil {
			p.logger.Error().Err(err).Int64("product_id", productID).Msg("resolve category for outbox failed")
			return err
		}

		if err := p.insertOutboxEvent(tx, model.EventProductCreated, *product); err != nil {
			p.logger.Error().Err(err).Int64("product_id", productID).Msg("insert outbox event failed")
			return err
		}

		return nil
	})
}

// Update modifies the product and writes an outbox event atomically.
func (p *productService) Update(ctx context.Context, req entity.ProductEntity) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := p.repo.WithTx(tx)

		if err := txRepo.Update(ctx, req); err != nil {
			p.logger.Error().Err(err).Int64("product_id", req.ID).Msg("update product failed")
			return err
		}

		product, err := txRepo.GetByID(ctx, req.ID)
		if err != nil {
			p.logger.Error().Err(err).Int64("product_id", req.ID).Msg("fetch updated product failed")
			return err
		}
		product, err = p.productWithCategoryName(ctx, product)
		if err != nil {
			p.logger.Error().Err(err).Int64("product_id", req.ID).Msg("resolve category for outbox failed")
			return err
		}

		if err := p.insertOutboxEvent(tx, model.EventProductUpdated, *product); err != nil {
			p.logger.Error().Err(err).Int64("product_id", req.ID).Msg("insert outbox event failed")
			return err
		}

		return nil
	})
}

// Delete removes the product and writes an outbox event atomically.
func (p *productService) Delete(ctx context.Context, productID int64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := p.repo.WithTx(tx)

		if err := txRepo.Delete(ctx, productID); err != nil {
			p.logger.Error().Err(err).Int64("product_id", productID).Msg("delete product failed")
			return err
		}

		payload := map[string]interface{}{
			"product_id": productID,
		}

		if err := p.insertOutboxEventRaw(tx, model.EventProductDeleted, payload); err != nil {
			p.logger.Error().Err(err).Int64("product_id", productID).Msg("insert outbox event failed")
			return err
		}

		return nil
	})
}

func (p *productService) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.GetAll(ctx, query)
}

func (p *productService) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	result, err := p.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	return p.productWithCategoryName(ctx, result)
}

// productWithCategoryName sets CategoryName from categories table (by slug).
// Category rows are expected to be committed already; product may be read inside an open transaction.
func (p *productService) productWithCategoryName(ctx context.Context, product *entity.ProductEntity) (*entity.ProductEntity, error) {
	if product.CategorySlug == nil {
		return nil, errors.New("category slug is null")
	}
	resultCat, err := p.repoCat.GetBySlug(ctx, *product.CategorySlug)
	if err != nil {
		return nil, err
	}
	if resultCat == nil {
		return nil, errors.New("category not found")
	}
	product.CategoryName = &resultCat.Name
	return product, nil
}

// ReindexAll fetches all active products from PostgreSQL and returns them
// for bulk-indexing into Elasticsearch by the handler.
func (p *productService) ReindexAll(ctx context.Context) (int, error) {
	// Fetch a large batch; in production use cursor-based pagination.
	query := entity.QueryStringProduct{
		Page:      1,
		Limit:     10000,
		OrderBy:   "id",
		OrderType: "asc",
		Status:    "ACTIVE",
	}

	products, _, _, err := p.repo.GetAll(ctx, query)
	if err != nil && err.Error() != "404" {
		return 0, fmt.Errorf("fetch all products: %w", err)
	}
	return len(products), nil
}

// insertOutboxEvent marshals any struct and writes it to the outbox table.
func (p *productService) insertOutboxEvent(tx *gorm.DB, eventType string, payload interface{}) error {
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
	return p.outboxRepo.Insert(tx, event)
}

func (p *productService) insertOutboxEventRaw(tx *gorm.DB, eventType string, payload interface{}) error {
	return p.insertOutboxEvent(tx, eventType, payload)
}

func NewProductService(
	db *gorm.DB,
	repo repository.ProductRepositoryInterface,
	outboxRepo repository.OutboxRepositoryInterface,
	repoCat repository.CategoryRepositoryInterface,
	logger zerolog.Logger,
) ProductServiceInterface {
	return &productService{
		db:         db,
		repo:       repo,
		outboxRepo: outboxRepo,
		repoCat:    repoCat,
		logger:     logger.With().Str("component", "product_service").Logger(),
	}
}
