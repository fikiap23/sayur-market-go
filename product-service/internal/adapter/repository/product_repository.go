package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type ProductRepositoryInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	Update(ctx context.Context, req entity.ProductEntity) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	// WithTx returns a copy of the repository that uses the given transaction for all DB operations.
	WithTx(tx *gorm.DB) ProductRepositoryInterface
	// GetDB exposes the underlying *gorm.DB for transaction management.
	GetDB() *gorm.DB
}

type productRepository struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
}

// Delete implements ProductRepositoryInterface.
// ES sync is now handled by the outbox worker + consumer, not inline.
func (p *productRepository) Delete(ctx context.Context, productID int64) error {
	modelProduct := model.Product{}

	if err := p.db.WithContext(ctx).Preload("Childs").First(&modelProduct, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
		}
		log.Errorf("[ProductRepository-1] Delete: %v", err)
		return err
	}

	if err := p.db.WithContext(ctx).Select("Childs").Delete(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository-2] Delete: %v", err)
		return err
	}

	return nil
}

// Update implements ProductRepositoryInterface.
func (p *productRepository) Update(ctx context.Context, req entity.ProductEntity) error {
	modelProduct := model.Product{}

	if err := p.db.Where("id = ?", req.ID).First(&modelProduct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
		}
		log.Errorf("[ProductRepository-1] Update: %v", err)
		return err
	}

	modelProduct.CategorySlug = req.CategorySlug
	modelProduct.ParentID = req.ParentID
	modelProduct.Name = req.Name
	modelProduct.Image = req.Image
	modelProduct.Description = req.Description
	modelProduct.RegulerPrice = req.RegulerPrice
	modelProduct.SalePrice = req.SalePrice
	modelProduct.Unit = req.Unit
	modelProduct.Weight = req.Weight
	modelProduct.Stock = req.Stock
	modelProduct.Variant = req.Variant
	modelProduct.Status = req.Status

	if err := p.db.Save(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository-2] Update: %v", err)
		return err
	}

	if len(req.Child) > 0 {
		if err := p.db.Where("parent_id = ?", modelProduct.ID).Delete(&model.Product{}).Error; err != nil {
			log.Errorf("[ProductRepository-3] Update: %v", err)
			return err
		}

		modelProductChild := []model.Product{}
		for _, val := range req.Child {
			modelProductChild = append(modelProductChild, model.Product{
				CategorySlug: req.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         req.Name,
				Image:        val.Image,
				Description:  req.Description,
				RegulerPrice: val.RegulerPrice,
				SalePrice:    val.SalePrice,
				Unit:         req.Unit,
				Weight:       val.Weight,
				Stock:        val.Stock,
				Variant:      req.Variant,
				Status:       req.Status,
			})
		}

		if err := p.db.Create(&modelProductChild).Error; err != nil {
			log.Errorf("[ProductRepository-3] Update: %v", err)
			return err
		}
	}

	return nil
}

// Create implements ProductRepositoryInterface.
func (p *productRepository) Create(ctx context.Context, req entity.ProductEntity) (int64, error) {
	modelProduct := model.Product{
		CategorySlug: req.CategorySlug,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Image:        req.Image,
		Description:  req.Description,
		RegulerPrice: req.RegulerPrice,
		SalePrice:    req.SalePrice,
		Unit:         req.Unit,
		Weight:       req.Weight,
		Stock:        req.Stock,
		Variant:      req.Variant,
		Status:       req.Status,
	}

	if err := p.db.Create(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository-1] Create: %v", err)
		return 0, err
	}

	if len(req.Child) > 0 {
		modelProductChild := []model.Product{}
		for _, val := range req.Child {
			modelProductChild = append(modelProductChild, model.Product{
				CategorySlug: req.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         req.Name,
				Image:        val.Image,
				Description:  req.Description,
				RegulerPrice: val.RegulerPrice,
				SalePrice:    val.SalePrice,
				Unit:         req.Unit,
				Weight:       val.Weight,
				Stock:        val.Stock,
				Variant:      req.Variant,
				Status:       req.Status,
			})
		}

		if err := p.db.Create(&modelProductChild).Error; err != nil {
			log.Errorf("[ProductRepository-2] Create: %v", err)
			return 0, err
		}
	}

	return modelProduct.ID, nil
}

// GetByID implements ProductRepositoryInterface.
func (p *productRepository) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	modelProduct := model.Product{}

	if err := p.db.WithContext(ctx).Preload("Category").First(&modelProduct, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
		}
		log.Errorf("[ProductRepository-1] GetByID: %v", err)
		return nil, err
	}

	modelParent := []model.Product{}
	err := p.db.WithContext(ctx).Preload("Category").Where("parent_id = ?", modelProduct.ID).Find(&modelParent).Error
	if err != nil {
		log.Errorf("[ProductRepository-2] GetByID: %v", err)
		return nil, err
	}

	childEntities := []entity.ProductEntity{}
	for _, val := range modelParent {
		childEntities = append(childEntities, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: &val.Category.Name,
			Child:        childEntities,
			CreatedAt:    val.CreatedAt,
			UpdatedAt:    val.UpdatedAt,
		})
	}

	return &entity.ProductEntity{
		ID:           modelProduct.ID,
		CategorySlug: modelProduct.CategorySlug,
		ParentID:     modelProduct.ParentID,
		Name:         modelProduct.Name,
		Image:        modelProduct.Image,
		Description:  modelProduct.Description,
		RegulerPrice: modelProduct.RegulerPrice,
		SalePrice:    modelProduct.SalePrice,
		Unit:         modelProduct.Unit,
		Weight:       modelProduct.Weight,
		Stock:        modelProduct.Stock,
		Variant:      modelProduct.Variant,
		Status:       modelProduct.Status,
		CategoryName: &modelProduct.Category.Name,
		Child:        childEntities,
		CreatedAt:    modelProduct.CreatedAt,
		UpdatedAt:    modelProduct.UpdatedAt,
	}, nil
}

// SearchProducts queries Elasticsearch first. If ES is unavailable or returns
// an error, it automatically falls back to PostgreSQL so the caller never panics.
func (p *productRepository) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	products, totalData, totalPage, err := p.searchES(ctx, query)
	if err != nil {
		log.Printf("[SearchProducts] ES failed (%v), falling back to PostgreSQL", err)
		return p.GetAll(ctx, query)
	}
	return products, totalData, totalPage, nil
}

func (p *productRepository) searchES(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	var mainQueries []string
	var filterQueries []string
	from := (query.Page - 1) * query.Limit

	sortField := "id"
	if query.OrderBy != "" {
		sortField = query.OrderBy
	}

	sortOrder := "asc"
	if query.OrderType == "desc" {
		sortOrder = "desc"
	}

	sortQuery := fmt.Sprintf(`{ "%s": "%s" }`, sortField, sortOrder)

	if query.CategorySlug != "" {
		filterQueries = append(filterQueries, fmt.Sprintf(`{ "term": { "category_slug.keyword": "%s" } }`, query.CategorySlug))
	}

	if query.StartPrice > 0 && query.EndPrice > 0 {
		filterQueries = append(filterQueries, fmt.Sprintf(`{ "range": { "reguler_price": { "gte": %d, "lte": %d } } }`, query.StartPrice, query.EndPrice))
	}

	if query.Search != "" {
		mainQueries = append(mainQueries, fmt.Sprintf(`{ "multi_match": { "query": "%s", "fields": ["name", "description", "category_name"] } }`, query.Search))
	}

	mainQuery := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"bool": {
				"must": [
					%s
				],
				"filter": [
					%s
				]
			}
		},
		"sort": [
			%s
		]
	}`, from, query.Limit, strings.Join(mainQueries, ","), strings.Join(filterQueries, ","), sortQuery)

	res, err := p.esClient.Search(
		p.esClient.Search.WithContext(ctx),
		p.esClient.Search.WithIndex("products"),
		p.esClient.Search.WithBody(strings.NewReader(mainQuery)),
		p.esClient.Search.WithPretty(),
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("es search request: %w", err)
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, 0, fmt.Errorf("es decode response: %w", err)
	}
	if res.IsError() {
		errDetail, _ := result["error"]
		return nil, 0, 0, fmt.Errorf("es search error: %v", errDetail)
	}

	hitsRoot, ok := result["hits"].(map[string]interface{})
	if !ok || hitsRoot == nil {
		return nil, 0, 0, fmt.Errorf("es: missing hits in response")
	}

	totalData := esHitsTotal(hitsRoot)

	totalPage := 0
	if query.Limit > 0 {
		totalPage = int(math.Ceil(float64(totalData) / float64(query.Limit)))
	}

	products := []entity.ProductEntity{}
	hits, ok := hitsRoot["hits"].([]interface{})
	if ok {
		for _, hit := range hits {
			hm, ok := hit.(map[string]interface{})
			if !ok {
				continue
			}
			source, ok := hm["_source"]
			if !ok {
				continue
			}
			data, err := json.Marshal(source)
			if err != nil {
				continue
			}
			var product entity.ProductEntity
			if err := json.Unmarshal(data, &product); err != nil {
				continue
			}
			products = append(products, product)
		}
	}

	return products, int64(totalData), int64(totalPage), nil
}

// esHitsTotal reads hit count from ES search response (ES 6: number; ES 7+: { "value", "relation" }).
func esHitsTotal(hits map[string]interface{}) int {
	if hits == nil {
		return 0
	}
	t, ok := hits["total"]
	if !ok || t == nil {
		return 0
	}
	switch v := t.(type) {
	case float64:
		return int(v)
	case map[string]interface{}:
		if val, ok := v["value"].(float64); ok {
			return int(val)
		}
	}
	return 0
}

// GetAll implements ProductRepositoryInterface.
func (p *productRepository) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	modelProducts := []model.Product{}
	var countData int64

	order := fmt.Sprintf("%s %s", query.OrderBy, query.OrderType)
	offset := (query.Page - 1) * query.Limit
	defaultStatus := "ACTIVE"
	if query.Status != "" {
		defaultStatus = query.Status
	}
	sqlMain := p.db.Preload("Category").
		Where("parent_id IS NULL AND status = ?", defaultStatus).
		Where("name ILIKE ? OR description ILIKE ? OR category_slug ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%")
	if query.CategorySlug != "" {
		sqlMain = sqlMain.Where("category_slug = ?", query.CategorySlug)
	}

	if query.StartPrice > 0 {
		sqlMain = sqlMain.Where("sale_price >= ?", query.StartPrice)
	}

	if query.EndPrice > 0 {
		sqlMain = sqlMain.Where("sale_price <= ?", query.EndPrice)
	}

	if err := sqlMain.Model(&modelProducts).Count(&countData).Error; err != nil {
		log.Errorf("[ProductRepository-1] GetAll: %v", err)
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(query.Limit)))
	if err := sqlMain.Order(order).Limit(int(query.Limit)).Offset(int(offset)).Find(&modelProducts).Error; err != nil {
		log.Errorf("[ProductRepository-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelProducts) == 0 {
		log.Errorf("[ProductRepository-3] GetAll: %v", "Data not found")
		return nil, 0, 0, errors.New("404")
	}

	respProducts := []entity.ProductEntity{}
	for _, val := range modelProducts {
		respProducts = append(respProducts, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: &val.Category.Name,
			CreatedAt:    val.CreatedAt,
			UpdatedAt:    val.UpdatedAt,
		})
	}

	return respProducts, countData, int64(totalPage), nil
}

func (p *productRepository) WithTx(tx *gorm.DB) ProductRepositoryInterface {
	return &productRepository{db: tx, esClient: p.esClient}
}

func (p *productRepository) GetDB() *gorm.DB {
	return p.db
}

func NewProductRepository(db *gorm.DB, es *elasticsearch.Client) ProductRepositoryInterface {
	return &productRepository{db: db, esClient: es}
}
