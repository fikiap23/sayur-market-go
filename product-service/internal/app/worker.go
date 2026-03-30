package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"product-service/config"
	"product-service/internal/adapter/message"
	rmq "product-service/internal/adapter/rabbitmq"

	"github.com/rs/zerolog"
)

// RunWorkerESIndex runs a standalone consumer for the product publish queue (Elasticsearch index).
func RunWorkerESIndex() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	queue := cfg.PublisherName.ProductPublish
	if queue == "" {
		logger.Fatal().Msg("PRODUCT_PUBLISH_NAME must be set for this worker")
	}

	esClient, err := cfg.InitElasticsearch()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to elasticsearch")
	}

	connMgr, metrics, err := initRabbitConn(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("rabbitmq not reachable at startup")
	}

	handler := message.NewESIndexHandler(esClient, logger)
	consumer := rmq.NewConsumer(connMgr, queue, handler, logger, consumerOptions(cfg, metrics)...)

	logger.Info().Str("queue", queue).Msg("starting ES index worker")
	runConsumerUntilSignal(consumer, connMgr, metrics, logger)
}

// RunWorkerESDelete runs a standalone consumer for the product delete queue (Elasticsearch delete).
func RunWorkerESDelete() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	queue := cfg.PublisherName.ProductDelete
	if queue == "" {
		logger.Fatal().Msg("PRODUCT_DELETE must be set for this worker")
	}

	esClient, err := cfg.InitElasticsearch()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to elasticsearch")
	}

	connMgr, metrics, err := initRabbitConn(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("rabbitmq not reachable at startup")
	}

	handler := message.NewESDeleteHandler(esClient, logger)
	consumer := rmq.NewConsumer(connMgr, queue, handler, logger, consumerOptions(cfg, metrics)...)

	logger.Info().Str("queue", queue).Msg("starting ES delete worker")
	runConsumerUntilSignal(consumer, connMgr, metrics, logger)
}

// RunWorkerUpdateStock runs a standalone consumer for the stock update queue (DB).
func RunWorkerUpdateStock() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	queue := cfg.PublisherName.ProductUpdateStock
	if queue == "" {
		logger.Fatal().Msg("PRODUCT_UPDATE_STOCK_NAME must be set for this worker")
	}

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}

	connMgr, metrics, err := initRabbitConn(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("rabbitmq not reachable at startup")
	}

	handler := message.NewUpdateStockHandler(db.DB, logger)
	consumer := rmq.NewConsumer(connMgr, queue, handler, logger, consumerOptions(cfg, metrics)...)

	logger.Info().Str("queue", queue).Msg("starting update-stock worker")
	runConsumerUntilSignal(consumer, connMgr, metrics, logger)
}

func initRabbitConn(cfg *config.Config, logger zerolog.Logger) (*rmq.ConnectionManager, *rmq.Metrics, error) {
	connMgr := rmq.NewConnectionManager(cfg.RabbitMQURL(), logger)
	startupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := connMgr.WaitForReady(startupCtx, 10, 2*time.Second); err != nil {
		return nil, nil, err
	}
	return connMgr, rmq.NewMetrics(), nil
}

func consumerOptions(cfg *config.Config, metrics *rmq.Metrics) []rmq.ConsumerOption {
	var opts []rmq.ConsumerOption
	if cfg.RabbitMQ.WorkerPoolSize > 0 {
		opts = append(opts, rmq.WithWorkerPoolSize(cfg.RabbitMQ.WorkerPoolSize))
	}
	if cfg.RabbitMQ.PrefetchCount > 0 {
		opts = append(opts, rmq.WithPrefetchCount(cfg.RabbitMQ.PrefetchCount))
	}
	if cfg.RabbitMQ.MaxRetries > 0 {
		opts = append(opts, rmq.WithMaxRetries(cfg.RabbitMQ.MaxRetries))
	}
	if cfg.RabbitMQ.ProcessTimeoutSec > 0 {
		opts = append(opts, rmq.WithProcessTimeout(time.Duration(cfg.RabbitMQ.ProcessTimeoutSec)*time.Second))
	}
	opts = append(opts, rmq.WithConsumerMetrics(metrics))
	opts = append(opts, rmq.WithDLQ(true))
	return opts
}

func runConsumerUntilSignal(
	consumer *rmq.Consumer,
	connMgr *rmq.ConnectionManager,
	metrics *rmq.Metrics,
	logger zerolog.Logger,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info().Msg("shutting down worker...")
		cancel()
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("consumer exited with error")
		}
		shutdownRabbit(connMgr, metrics, logger)
		return
	}

	err := <-errCh
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error().Err(err).Msg("consumer exited with error")
	}
	shutdownRabbit(connMgr, metrics, logger)
}

func shutdownRabbit(connMgr *rmq.ConnectionManager, metrics *rmq.Metrics, logger zerolog.Logger) {
	if err := connMgr.Close(); err != nil {
		logger.Error().Err(err).Msg("rabbitmq connection close error")
	}
	if metrics == nil {
		return
	}
	snap := metrics.Snapshot()
	logger.Info().
		Int64("consumed", snap.ConsumeCount).
		Int64("acks", snap.AckCount).
		Int64("nacks", snap.NackCount).
		Int64("retries", snap.RetryCount).
		Msg("final consumer metrics")
}
