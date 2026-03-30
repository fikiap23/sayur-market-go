package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"order-service/internal/core/domain/entity"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
)

type ElasticRepositoryInterface interface {
	SearchOrderElastic(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error)
	SearchOrderElasticByBuyerId(ctx context.Context, queryString entity.QueryStringEntity, buyerId int64) ([]entity.OrderEntity, int64, int64, error)
}

type elasticRepository struct {
	esClient *elasticsearch.Client
}

func NewElasticRepository(es *elasticsearch.Client) ElasticRepositoryInterface {
	return &elasticRepository{esClient: es}
}

// SearchOrderElasticByBuyerId implements ElasticRepositoryInterface.
func (e *elasticRepository) SearchOrderElasticByBuyerId(ctx context.Context, query entity.QueryStringEntity, buyerId int64) ([]entity.OrderEntity, int64, int64, error) {
	from := (query.Page - 1) * query.Limit

	statusFilter := ""
	if query.Status != "" {
		statusFilter = fmt.Sprintf(`{ "match": { "status": "%s" } },`, query.Status)
	}

	searchFilter := `{"match_all": {}}`
	if query.Search != "" {
		searchFilter = fmt.Sprintf(`{ "multi_match": { "query": "%s", "fields": ["order_code", "status", "buyer_name"] } }`, query.Search)
	}

	idFilter := ""
	if buyerId != 0 {
		idFilter = fmt.Sprintf(`{ "term": { "buyer_id": %d } },`, buyerId)
	}
	// Query Elasticsearch dengan filtering dan pagination
	mainQuery := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"bool": {
				"must": [
					%s
					%s
					%s
				]
			}
		},
		"sort": [
			{ "id": "desc" }
		]
	}`, from, query.Limit, idFilter, statusFilter, searchFilter)

	// Kirim query ke Elasticsearch
	res, err := e.esClient.Search(
		e.esClient.Search.WithContext(ctx),
		e.esClient.Search.WithIndex("orders"),
		e.esClient.Search.WithBody(strings.NewReader(mainQuery)),
		e.esClient.Search.WithPretty(),
	)

	if err != nil {
		log.Printf("Error searching Elasticsearch: %s", err)
		return nil, 0, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Printf("Error decoding response: %s", err)
			return nil, 0, 0, err
		}

		errType := e["error"].(map[string]interface{})["type"]
		if errType == "index_not_found_exception" {
			log.Printf("Index Not Found: %s", err)
			return nil, 0, 0, errors.New("index not found")
		}

		return nil, 0, 0, errors.New(e["error"].(string))
	}

	// Decode response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response: %s", err)
		return nil, 0, 0, err
	}

	// Ambil total data
	totalData := 0
	if hitsTotal, found := result["hits"].(map[string]interface{})["total"].(map[string]interface{}); found {
		totalData = int(hitsTotal["value"].(float64))
	}

	// Hitung total halaman
	totalPage := 0
	if query.Limit > 0 {
		totalPage = int(math.Ceil(float64(totalData) / float64(query.Limit)))
	}

	// Parsing hasil pencarian ke struct domain.Product
	orders := []entity.OrderEntity{}
	hits, found := result["hits"].(map[string]interface{})["hits"].([]interface{})
	if found {
		for _, hit := range hits {
			source := hit.(map[string]interface{})["_source"]
			data, _ := json.Marshal(source)
			var order entity.OrderEntity
			json.Unmarshal(data, &order)
			orders = append(orders, order)
		}
	}

	return orders, int64(totalData), int64(totalPage), nil
}

// SearchOrderElastic implements ElasticRepositoryInterface.
func (e *elasticRepository) SearchOrderElastic(ctx context.Context, query entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error) {
	from := (query.Page - 1) * query.Limit

	statusFilter := ""
	if query.Status != "" {
		statusFilter = fmt.Sprintf(`{ "match": { "status": "%s" } },`, query.Status)
	}

	searchFilter := `{"match_all": {}}`
	if query.Search != "" {
		searchFilter = fmt.Sprintf(`{ "multi_match": { "query": "%s", "fields": ["order_code", "status", "buyer_name"] } }`, query.Search)
	}
	// Query Elasticsearch dengan filtering dan pagination
	mainQuery := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"bool": {
				"must": [
					%s
					%s
				]
			}
		},
		"sort": [
			{ "id": "asc" }
		]
	}`, from, query.Limit, statusFilter, searchFilter)

	// Kirim query ke Elasticsearch
	res, err := e.esClient.Search(
		e.esClient.Search.WithContext(ctx),
		e.esClient.Search.WithIndex("orders"),
		e.esClient.Search.WithBody(strings.NewReader(mainQuery)),
		e.esClient.Search.WithPretty(),
	)

	if err != nil {
		log.Printf("Error searching Elasticsearch: %s", err)
		return nil, 0, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Printf("Error decoding response: %s", err)
			return nil, 0, 0, err
		}

		errType := e["error"].(map[string]interface{})["type"]
		if errType == "index_not_found_exception" {
			log.Printf("Index Not Found: %s", err)
			return nil, 0, 0, errors.New("index not found")
		}

		return nil, 0, 0, errors.New(e["error"].(string))
	}

	// Decode response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response: %s", err)
		return nil, 0, 0, err
	}

	// Ambil total data
	totalData := 0
	if hitsTotal, found := result["hits"].(map[string]interface{})["total"].(map[string]interface{}); found {
		totalData = int(hitsTotal["value"].(float64))
	}

	// Hitung total halaman
	totalPage := 0
	if query.Limit > 0 {
		totalPage = int(math.Ceil(float64(totalData) / float64(query.Limit)))
	}

	// Parsing hasil pencarian ke struct domain.Product
	orders := []entity.OrderEntity{}
	hits, found := result["hits"].(map[string]interface{})["hits"].([]interface{})
	if found {
		for _, hit := range hits {
			source := hit.(map[string]interface{})["_source"]
			data, _ := json.Marshal(source)
			var order entity.OrderEntity
			json.Unmarshal(data, &order)
			orders = append(orders, order)
		}
	}

	return orders, int64(totalData), int64(totalPage), nil
}
