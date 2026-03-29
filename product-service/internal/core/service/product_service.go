package service

import (
	"context"
	"errors"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

type ProductServiceInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) error
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

type productService struct {
	repo              repository.ProductRepositoryInterface
	publisherRabbitMQ message.PublishRabbitMQInterface
	repoCat           repository.CategoryRepositoryInterface
}

// SearchProducts implements ProductServiceInterface.
func (p *productService) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.SearchProducts(ctx, query)
}

// Create implements ProductServiceInterface.
func (p *productService) Create(ctx context.Context, req entity.ProductEntity) error {
	productID, err := p.repo.Create(ctx, req)
	if err != nil {
		log.Errorf("[ProductService-1] Create: %v", err)
		return err
	}

	getProductByID, err := p.GetByID(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-2] Create: %v", err)
	}

	if err := p.publisherRabbitMQ.PublishProductToQueue(*getProductByID); err != nil {
		log.Errorf("[ProductService-3] Create: %v", err)
	}

	return nil
}

// Delete implements ProductServiceInterface.
func (p *productService) Delete(ctx context.Context, productID int64) error {
	err := p.repo.Delete(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-1] Delete: %v", err)
		return err
	}

	if err := p.publisherRabbitMQ.DeleteProductFromQueue(productID); err != nil {
		log.Errorf("[ProductService-2] Delete: %v", err)
	}

	return nil
}

// GetAll implements ProductServiceInterface.
func (p *productService) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.GetAll(ctx, query)
}

// GetByID implements ProductServiceInterface.
func (p *productService) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	result, err := p.repo.GetByID(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-1] GetByID: %v", err)
		return nil, err
	}

	if result.CategorySlug == nil {
		return nil, errors.New("category slug is null")
	}

	resultCat, err := p.repoCat.GetBySlug(ctx, *result.CategorySlug)
	if err != nil {
		log.Errorf("[ProductService-2] GetByID: %v", err)
		return nil, err
	}
	if resultCat == nil {
		return nil, errors.New("category not found")
	}
	result.CategoryName = &resultCat.Name
	return result, nil
}

// Update implements ProductServiceInterface.
func (p *productService) Update(ctx context.Context, req entity.ProductEntity) error {
	err := p.repo.Update(ctx, req)
	if err != nil {
		log.Errorf("[ProductService-1] Update: %v", err)
		return err
	}

	getProductByID, err := p.GetByID(ctx, req.ID)
	if err != nil {
		log.Errorf("[ProductService-2] Update: %v", err)
	}

	if err := p.publisherRabbitMQ.PublishProductToQueue(*getProductByID); err != nil {
		log.Errorf("[ProductService-3] Update: %v", err)
	}

	return nil
}

func NewProductService(repo repository.ProductRepositoryInterface, publisherRabbitMQ message.PublishRabbitMQInterface, repoCat repository.CategoryRepositoryInterface) ProductServiceInterface {
	return &productService{repo: repo, publisherRabbitMQ: publisherRabbitMQ, repoCat: repoCat}
}
