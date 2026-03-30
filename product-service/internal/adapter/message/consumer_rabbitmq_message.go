package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"product-service/internal/core/domain/entity"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog"
)

// ESIndexHandler indexes products into Elasticsearch. It is idempotent:
// if an existing document has a newer "version" (max of created_at / updated_at)
// than the incoming event, the update is skipped to handle out-of-order events.
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

	docID := fmt.Sprintf("%d", product.ID)

	incomingVer := product.EffectiveVersionTime()
	// Check for out-of-order: if an existing doc has a newer version time, skip.
	existingVer, err := h.getExistingVersionTime(ctx, docID)
	if err == nil && existingVer != nil && !existingVer.IsZero() && !incomingVer.IsZero() {
		if incomingVer.Before(*existingVer) {
			h.logger.Info().
				Int64("product_id", product.ID).
				Time("existing_version_ts", *existingVer).
				Time("incoming_version_ts", incomingVer).
				Msg("skipping out-of-order event, existing doc is newer")
			return nil
		}
	}

	productJSON, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("marshal product for ES: %w", err)
	}

	// Use Index (upsert semantics) — idempotent by design since the same
	// document ID always maps to the same ES doc.
	res, err := h.esClient.Index(
		"products",
		bytes.NewReader(productJSON),
		h.esClient.Index.WithDocumentID(docID),
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

	h.logger.Debug().Int64("product_id", product.ID).Msg("product indexed in ES (idempotent)")
	return nil
}

// getExistingVersionTime returns max(created_at, updated_at) from an existing ES document.
// Returns nil if the document does not exist.
func (h *ESIndexHandler) getExistingVersionTime(ctx context.Context, docID string) (*time.Time, error) {
	res, err := h.esClient.Get("products", docID,
		h.esClient.Get.WithContext(ctx),
		h.esClient.Get.WithSourceIncludes("created_at", "updated_at"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, nil
	}
	if res.IsError() {
		return nil, fmt.Errorf("es get error: %s", res.Status())
	}

	var result struct {
		Source struct {
			CreatedAt time.Time  `json:"created_at"`
			UpdatedAt *time.Time `json:"updated_at"`
		} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	t := result.Source.CreatedAt
	if result.Source.UpdatedAt != nil && !result.Source.UpdatedAt.IsZero() && result.Source.UpdatedAt.After(t) {
		t = *result.Source.UpdatedAt
	}
	return &t, nil
}

// ESDeleteHandler removes products from Elasticsearch. It is idempotent:
// deleting a non-existent document is treated as success.
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
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("unmarshal delete msg: %w", err)
	}

	// Support both string and numeric product_id from the outbox payload.
	productID := ""
	if v, ok := data["ProductID"]; ok {
		productID = fmt.Sprintf("%v", v)
	} else if v, ok := data["product_id"]; ok {
		switch id := v.(type) {
		case float64:
			productID = strconv.FormatInt(int64(id), 10)
		case string:
			productID = id
		default:
			productID = fmt.Sprintf("%v", id)
		}
	}

	if productID == "" || productID == "0" {
		return fmt.Errorf("missing product_id in delete message")
	}

	res, err := h.esClient.Delete("products", productID,
		h.esClient.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete from ES: %w", err)
	}
	defer res.Body.Close()

	// 404 means the document was already deleted — idempotent success.
	if res.StatusCode == 404 {
		h.logger.Debug().Str("product_id", productID).Msg("product already absent from ES, idempotent OK")
		return nil
	}

	if res.IsError() {
		resBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES delete error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Str("product_id", productID).Msg("product deleted from ES (idempotent)")
	return nil
}
