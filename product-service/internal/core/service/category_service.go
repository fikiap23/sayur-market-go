package service

import (
	"context"
	"errors"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"product-service/utils/conv"

	"github.com/labstack/gommon/log"
)

type CategoryServiceInterface interface {
	GetAll(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error)
	GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error)
	GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error)
	CreateCategory(ctx context.Context, req entity.CategoryEntity) error
	EditCategory(ctx context.Context, req entity.CategoryEntity) error
	DeleteCategory(ctx context.Context, categoryID int64) error

	GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error)
}

type categoryService struct {
	repo repository.CategoryRepositoryInterface
}

// GetAllPublished implements CategoryServiceInterface.
func (c *categoryService) GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error) {
	return c.repo.GetAllPublished(ctx)
}

// CreateCategory implements CategoryServiceInterface.
func (c *categoryService) CreateCategory(ctx context.Context, req entity.CategoryEntity) error {
	slug := conv.GenerateSlug(req.Name)
	result, err := c.repo.GetBySlug(ctx, slug)
	if err != nil {
		if err.Error() != "404" {
			log.Errorf("[CategoryService-1] CreateCategory: %v", err)
			return err
		}
	}

	if result != nil {
		err = errors.New("409")
		log.Infof("[CategoryService-2] CreateCategory: Category already exists")
		return err
	}

	req.Slug = slug
	err = c.repo.CreateCategory(ctx, req)
	if err != nil {
		log.Errorf("[CategoryService-3] CreateCategory: %v", err)
		return err
	}
	return nil
}

// DeleteCategory implements CategoryServiceInterface.
func (c *categoryService) DeleteCategory(ctx context.Context, categoryID int64) error {
	return c.repo.DeleteCategory(ctx, categoryID)
}

// EditCategory implements CategoryServiceInterface.
func (c *categoryService) EditCategory(ctx context.Context, req entity.CategoryEntity) error {
	slug := conv.GenerateSlug(req.Name)
	result, err := c.repo.GetByID(ctx, req.ID)
	if err != nil {
		log.Errorf("[CategoryService-1] EditCategory: %v", err)
		return err
	}

	if slug != result.Slug {
		resSlug, err := c.repo.GetBySlug(ctx, slug)
		if err != nil {
			log.Errorf("[CategoryService-2] EditCategory: %v", err)
			return err
		}
		if resSlug != nil {
			err = errors.New("409")
			log.Infof("[CategoryService-3] EditCategory: Category already exists")
			return err
		}
	}

	req.Slug = slug
	err = c.repo.EditCategory(ctx, req)
	if err != nil {
		log.Errorf("[CategoryService-4] EditCategory: %v", err)
		return err
	}

	return nil
}

// GetAll implements CategoryServiceInterface.
func (c *categoryService) GetAll(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error) {
	return c.repo.GetAll(ctx, queryString)
}

// GetByID implements CategoryServiceInterface.
func (c *categoryService) GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error) {
	return c.repo.GetByID(ctx, categoryID)
}

// GetBySlug implements CategoryServiceInterface.
func (c *categoryService) GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error) {
	return c.repo.GetBySlug(ctx, slug)
}

func NewCategoryService(repo repository.CategoryRepositoryInterface) CategoryServiceInterface {
	return &categoryService{repo: repo}
}
