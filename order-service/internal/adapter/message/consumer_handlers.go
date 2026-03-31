package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"order-service/internal/core/domain/entity"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/rs/zerolog"
)

type OrderESIndexHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewOrderESIndexHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *OrderESIndexHandler {
	return &OrderESIndexHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "order_es_index_handler").Logger(),
	}
}

func (h *OrderESIndexHandler) Handle(ctx context.Context, body []byte) error {
	var order entity.OrderEntity
	if err := json.Unmarshal(body, &order); err != nil {
		return fmt.Errorf("unmarshal order: %w", err)
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order for ES: %w", err)
	}

	res, err := h.esClient.Index(
		"orders",
		bytes.NewReader(orderJSON),
		h.esClient.Index.WithDocumentID(fmt.Sprintf("%d", order.ID)),
		h.esClient.Index.WithContext(ctx),
		h.esClient.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("index to ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		resBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES index error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Int64("order_id", order.ID).Msg("order indexed in ES")
	return nil
}

type OrderPaymentSuccessHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewOrderPaymentSuccessHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *OrderPaymentSuccessHandler {
	return &OrderPaymentSuccessHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "order_payment_success_handler").Logger(),
	}
}

func (h *OrderPaymentSuccessHandler) Handle(ctx context.Context, body []byte) error {
	var payment map[string]interface{}
	if err := json.Unmarshal(body, &payment); err != nil {
		return fmt.Errorf("unmarshal payment success message: %w", err)
	}

	pm, ok := payment["paymentMethod"].(string)
	if !ok || pm == "" {
		return fmt.Errorf("invalid or missing paymentMethod")
	}

	orderID, err := extractStringID(payment, "orderID")
	if err != nil {
		return err
	}

	updateScript := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.PaymentMethod = params.payment_method",
			"lang":   "painless",
			"params": map[string]interface{}{
				"payment_method": pm,
			},
		},
	}

	payload, err := json.Marshal(updateScript)
	if err != nil {
		return fmt.Errorf("marshal payment update script: %w", err)
	}

	res, err := h.esClient.Update("orders", orderID, bytes.NewReader(payload), h.esClient.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("update payment method in ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		resBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES update error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Str("order_id", orderID).Str("payment_method", pm).Msg("order payment method updated in ES")
	return nil
}

type OrderUpdateStatusHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewOrderUpdateStatusHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *OrderUpdateStatusHandler {
	return &OrderUpdateStatusHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "order_update_status_handler").Logger(),
	}
}

func (h *OrderUpdateStatusHandler) Handle(ctx context.Context, body []byte) error {
	var order map[string]interface{}
	if err := json.Unmarshal(body, &order); err != nil {
		return fmt.Errorf("unmarshal update status message: %w", err)
	}

	status, ok := order["status"].(string)
	if !ok || status == "" {
		return fmt.Errorf("invalid or missing status")
	}

	orderID, err := extractStringID(order, "orderID")
	if err != nil {
		return err
	}

	updateScript := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.status = params.status",
			"lang":   "painless",
			"params": map[string]interface{}{
				"status": status,
			},
		},
	}

	payload, err := json.Marshal(updateScript)
	if err != nil {
		return fmt.Errorf("marshal status update script: %w", err)
	}

	res, err := h.esClient.Update("orders", orderID, bytes.NewReader(payload), h.esClient.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("update status in ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		resBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES update error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Str("order_id", orderID).Str("status", status).Msg("order status updated in ES")
	return nil
}

type OrderESDeleteHandler struct {
	esClient *elasticsearch.Client
	logger   zerolog.Logger
}

func NewOrderESDeleteHandler(esClient *elasticsearch.Client, logger zerolog.Logger) *OrderESDeleteHandler {
	return &OrderESDeleteHandler{
		esClient: esClient,
		logger:   logger.With().Str("component", "order_es_delete_handler").Logger(),
	}
}

func (h *OrderESDeleteHandler) Handle(ctx context.Context, body []byte) error {
	var order map[string]interface{}
	if err := json.Unmarshal(body, &order); err != nil {
		return fmt.Errorf("unmarshal delete order message: %w", err)
	}

	orderID, err := extractStringID(order, "orderID")
	if err != nil {
		return err
	}

	res, err := h.esClient.Delete("orders", orderID, h.esClient.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("delete order from ES: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		h.logger.Debug().Str("order_id", orderID).Msg("order already absent in ES, idempotent OK")
		return nil
	}

	if res.IsError() {
		resBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES delete error [%s]: %s", res.Status(), string(resBody))
	}

	h.logger.Debug().Str("order_id", orderID).Msg("order deleted from ES")
	return nil
}

func extractStringID(data map[string]interface{}, key string) (string, error) {
	raw, ok := data[key]
	if !ok {
		return "", fmt.Errorf("missing %s", key)
	}

	switch v := raw.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("empty %s", key)
		}
		return v, nil
	case float64:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int:
		return strconv.Itoa(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}
