package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"

	"gorm.io/gorm"
)

type CategoryRepositoryInterface interface {
	GetAll(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error)
	GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error)
	GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error)
	CreateCategory(ctx context.Context, req entity.CategoryEntity) error
	EditCategory(ctx context.Context, req entity.CategoryEntity) error
	DeleteCategory(ctx context.Context, categoryID int64) error

	GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error)
}

type categoryRepository struct {
	db *gorm.DB
}

// GetAllPublished implements CategoryRepositoryInterface.
func (c *categoryRepository) GetAllPublished(ctx context.Context) ([]entity.CategoryEntity, error) {
	modelCategories := []model.Category{}

	if err := c.db.Select("id, parent_id, name, icon, slug").Where("status = ?", true).Find(&modelCategories).Error; err != nil {
		log.Errorf("[CategoryRepository-1] GetAllPublished: %v", err)
		return nil, err
	}

	if len(modelCategories) == 0 {
		err := errors.New("404")
		log.Infof("[CategoryRepository-2] GetAllPublished: No category found")
		return nil, err
	}

	entities := []entity.CategoryEntity{}
	for _, val := range modelCategories {
		entities = append(entities, entity.CategoryEntity{
			ID:       val.ID,
			ParentID: val.ParentID,
			Name:     val.Name,
			Icon:     val.Icon,
			Status:   "Published",
			Slug:     val.Slug,
		})
	}

	return entities, nil
}

// DeleteCategory implements CategoryRepositoryInterface.
func (c *categoryRepository) DeleteCategory(ctx context.Context, categoryID int64) error {
	modelCategory := model.Category{}

	if err := c.db.Preload("Products").Where("id = ?", categoryID).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[CategoryRepository-1] DeleteCategory: Category not found")
			return err
		}
		log.Errorf("[CategoryRepository-2] DeleteCategory: %v", err)
		return err
	}

	if len(modelCategory.Products) > 0 {
		err := errors.New("304")
		log.Errorf("[CategoryRepository-3] DeleteCategory: %v", "Category using other products")
		return err
	}

	if err := c.db.Delete(&modelCategory).Error; err != nil {
		log.Errorf("[CategoryRepository-4] DeleteCategory: %v", err)
		return err
	}

	return nil
}

// EditCategory implements CategoryRepositoryInterface.
func (c *categoryRepository) EditCategory(ctx context.Context, req entity.CategoryEntity) error {
	modelCategory := model.Category{}

	if err := c.db.Where("id = ?", req.ID).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[CategoryRepository-1] EditCategory: Category not found")
			return err
		}
		log.Errorf("[CategoryRepository-2] EditCategory: %v", err)
		return err
	}

	status := true
	if req.Status == "Unpublished" {
		status = false
	}
	modelCategory.ParentID = req.ParentID
	modelCategory.Name = req.Name
	modelCategory.Icon = req.Icon
	modelCategory.Status = status
	modelCategory.Slug = req.Slug
	modelCategory.Description = req.Description
	if err := c.db.Save(&modelCategory).Error; err != nil {
		log.Errorf("[CategoryRepository-3] EditCategory: %v", err)
		return err
	}

	return nil
}

// CreateCategory implements CategoryRepositoryInterface.
func (c *categoryRepository) CreateCategory(ctx context.Context, req entity.CategoryEntity) error {
	status := true
	if req.Status == "Unpublished" {
		status = false
	}
	modelCategory := model.Category{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Icon:        req.Icon,
		Status:      status,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := c.db.Create(&modelCategory).Error; err != nil {
		log.Errorf("[CategoryRepository-1] CreateCategory: %v", err)
		return err
	}
	return nil
}

// GetBySlug implements CategoryRepositoryInterface.
func (c *categoryRepository) GetBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error) {
	modelCategory := model.Category{}
	if err := c.db.Where("slug =?", slug).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[CategoryRepository-1] GetBySlug: Category not found")
			return nil, err
		}
		log.Errorf("[CategoryRepository-2] GetBySlug: %v", err)
		return nil, err
	}

	status := "Published"
	if modelCategory.Status == false {
		status = "Unpublished"
	}

	return &entity.CategoryEntity{
		ID:          modelCategory.ID,
		ParentID:    modelCategory.ParentID,
		Name:        modelCategory.Name,
		Icon:        modelCategory.Icon,
		Status:      status,
		Slug:        modelCategory.Slug,
		Description: modelCategory.Description,
	}, nil
}

// GetByID implements CategoryRepositoryInterface.
func (c *categoryRepository) GetByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error) {
	modelCategory := model.Category{}
	if err := c.db.Where("id = ?", categoryID).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[CategoryRepository-1] GetByID: Category not found")
			return nil, err
		}
		log.Errorf("[CategoryRepository-2] GetByID: %v", err)
		return nil, err
	}

	status := "Published"
	if modelCategory.Status == false {
		status = "Unpublished"
	}

	return &entity.CategoryEntity{
		ID:          categoryID,
		ParentID:    modelCategory.ParentID,
		Name:        modelCategory.Name,
		Icon:        modelCategory.Icon,
		Status:      status,
		Slug:        modelCategory.Slug,
		Description: modelCategory.Description,
	}, nil
}

// GetAll implements CategoryRepositoryInterface.
func (c *categoryRepository) GetAll(ctx context.Context, query entity.QueryStringEntity) ([]entity.CategoryEntity, int64, int64, error) {
	modelCategories := []model.Category{}
	var countData int64

	order := fmt.Sprintf("%s %s", query.OrderBy, query.OrderType)
	offset := (query.Page - 1) * query.Limit

	sqlMain := c.db.Preload("Products").
		Where("name ILIKE ? OR slug ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	if err := sqlMain.Model(&modelCategories).Count(&countData).Error; err != nil {
		log.Errorf("[CategoryRepository-1] GetAll: %v", err)
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(query.Limit)))
	if err := sqlMain.Order(order).Limit(int(query.Limit)).Offset(int(offset)).Find(&modelCategories).Error; err != nil {
		log.Errorf("[CategoryRepository-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelCategories) == 0 {
		err := errors.New("404")
		log.Infof("[CategoryRepository-3] GetAll: No category found")
		return nil, 0, 0, err
	}

	entities := []entity.CategoryEntity{}
	for _, val := range modelCategories {
		productEntities := []entity.ProductEntity{}
		for _, prd := range val.Products {
			productEntities = append(productEntities, entity.ProductEntity{
				ID:           prd.ID,
				CategorySlug: &val.Slug,
				ParentID:     prd.ParentID,
				Name:         prd.Name,
				Image:        prd.Image,
			})
		}
		status := "Published"
		if val.Status == false {
			status = "Unpublished"
		}

		entities = append(entities, entity.CategoryEntity{
			ID:          val.ID,
			ParentID:    val.ParentID,
			Name:        val.Name,
			Icon:        val.Icon,
			Status:      status,
			Slug:        val.Slug,
			Description: val.Description,
			Products:    productEntities,
		})
	}

	return entities, countData, int64(totalPage), nil
}

func NewCategoryRepository(db *gorm.DB) CategoryRepositoryInterface {
	return &categoryRepository{db: db}
}
