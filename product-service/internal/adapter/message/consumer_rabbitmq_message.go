package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"product-service/internal/core/domain/entity"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog"
)

// ESIndexHandler indexes products into Elasticsearch when consumed from the
// product-publish queue. Implements rabbitmq.MessageHandler.
type ESIndexHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewESIndexHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *ESIndexHandler {
	return &ESIndexHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "es_index_handler").Logger(),
	}
}

func (h *ESIndexHandler) Handle(ctx context.Context, body []byte) error {
	var product entity.ProductEntity
	if err := json.Unmarshal(body, &product); err != nil {
		return fmt.Errorf("unmarshal product: %w", err)
	}

	productJSON, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("marshal product for ES: %w", err)
	}

	res, err := h.esClient.Index(
		"products",
		bytes.NewReader(productJSON),
		h.esClient.Index.WithDocumentID(fmt.Sprintf("%d", product.ID)),
		h.esClient.Index.WithContext(ctx),
		h.esClient.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("index to ES: %w", err)
	}
	defer res.Body.Close()

	resBody, _ := io.ReadAll(res.Body)
	if res.IsError() {
		return fmt.Errorf("ES index error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Int64("product_id", product.ID).Msg("product indexed in ES")
	return nil
}

// ESDeleteHandler removes products from Elasticsearch when consumed from the
// product-delete queue. Implements rabbitmq.MessageHandler.
type ESDeleteHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewESDeleteHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *ESDeleteHandler {
	return &ESDeleteHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "es_delete_handler").Logger(),
	}
}

func (h *ESDeleteHandler) Handle(ctx context.Context, body []byte) error {
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("unmarshal delete msg: %w", err)
	}

	productID := data["ProductID"]
	if productID == "" {
		return fmt.Errorf("missing ProductID in message")
	}

	res, err := h.esClient.Delete("products", productID,
		h.esClient.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete from ES: %w", err)
	}
	defer res.Body.Close()

	h.logger.Debug().Str("product_id", productID).Msg("product deleted from ES")
	return nil
}
