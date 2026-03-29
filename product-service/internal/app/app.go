package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"product-service/config"
	"product-service/internal/adapter/handlers"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/outbox"
	rmq "product-service/internal/adapter/rabbitmq"
	"product-service/internal/adapter/repository"
	"product-service/internal/adapter/storage"
	"product-service/internal/core/service"
	"product-service/utils/validator"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	middlewareGateway "product-service/internal/middleware"
)

func RunServer() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
		return
	}

	elasticInit, err := cfg.InitElasticsearch()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to elasticsearch")
		return
	}

	// --- RabbitMQ setup ---
	connMgr := rmq.NewConnectionManager(cfg.RabbitMQURL(), logger)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := connMgr.WaitForReady(startupCtx, 10, 2*time.Second); err != nil {
		startupCancel()
		logger.Fatal().Err(err).Msg("rabbitmq not reachable at startup")
		return
	}
	startupCancel()

	metrics := rmq.NewMetrics()

	var pubOpts []rmq.PublisherOption
	if cfg.RabbitMQ.PublisherPoolSize > 0 {
		pubOpts = append(pubOpts, rmq.WithPoolSize(cfg.RabbitMQ.PublisherPoolSize))
	}
	if cfg.RabbitMQ.PublisherMaxRetries > 0 {
		pubOpts = append(pubOpts, rmq.WithPublishMaxRetries(cfg.RabbitMQ.PublisherMaxRetries))
	}
	if cfg.RabbitMQ.PublishTimeoutSec > 0 {
		pubOpts = append(pubOpts, rmq.WithPublishTimeout(time.Duration(cfg.RabbitMQ.PublishTimeoutSec)*time.Second))
	}
	pubOpts = append(pubOpts, rmq.WithPublisherMetrics(metrics))

	publisher, err := rmq.NewPublisher(connMgr, logger, pubOpts...)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize rabbitmq publisher")
		return
	}

	// --- Outbox worker setup ---
	// The outbox worker replaces direct publish from service layer.
	// It uses a topic exchange so consumers bind by routing key.
	outboxWorkerCfg := outbox.DefaultWorkerConfig()
	outboxRepo := repository.NewOutboxRepository(db.DB)
	outboxWorker := outbox.NewWorker(outboxRepo, publisher, outboxWorkerCfg, logger)

	// Declare topic exchange and bind queues for each event type.
	// Routing keys match model.EventProduct* constants.
	exchangeBindings := map[string]string{}
	if cfg.PublisherName.ProductPublish != "" {
		exchangeBindings["product.created"] = cfg.PublisherName.ProductPublish
		exchangeBindings["product.updated"] = cfg.PublisherName.ProductPublish
	}
	if cfg.PublisherName.ProductDelete != "" {
		exchangeBindings["product.deleted"] = cfg.PublisherName.ProductDelete
	}
	if err := outboxWorker.SetupExchange(connMgr, exchangeBindings); err != nil {
		logger.Fatal().Err(err).Msg("failed to setup exchange and bindings")
		return
	}

	// Also setup the stock update queue (published by external services).
	if cfg.PublisherName.ProductUpdateStock != "" {
		stockDLQ := cfg.PublisherName.ProductUpdateStock + ".dlq"
		if err := publisher.SetupQueue(stockDLQ, nil); err != nil {
			logger.Warn().Err(err).Str("queue", stockDLQ).Msg("stock DLQ setup failed")
		}
		stockArgs := amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": stockDLQ,
		}
		if err := publisher.SetupQueue(cfg.PublisherName.ProductUpdateStock, stockArgs); err != nil {
			logger.Warn().Err(err).Str("queue", cfg.PublisherName.ProductUpdateStock).Msg("stock queue setup failed")
		}
	}

	// --- Consumer setup ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var consumerWg sync.WaitGroup

	type consumerDef struct {
		queue   string
		handler rmq.MessageHandler
	}

	var consumers []consumerDef

	if cfg.PublisherName.ProductPublish != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.ProductPublish,
			handler: message.NewESIndexHandler(elasticInit, logger),
		})
	}
	if cfg.PublisherName.ProductDelete != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.ProductDelete,
			handler: message.NewESDeleteHandler(elasticInit, logger),
		})
	}
	if cfg.PublisherName.ProductUpdateStock != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.ProductUpdateStock,
			handler: message.NewUpdateStockHandler(db.DB, logger),
		})
	}

	for _, cd := range consumers {
		var consOpts []rmq.ConsumerOption
		if cfg.RabbitMQ.WorkerPoolSize > 0 {
			consOpts = append(consOpts, rmq.WithWorkerPoolSize(cfg.RabbitMQ.WorkerPoolSize))
		}
		if cfg.RabbitMQ.PrefetchCount > 0 {
			consOpts = append(consOpts, rmq.WithPrefetchCount(cfg.RabbitMQ.PrefetchCount))
		}
		if cfg.RabbitMQ.MaxRetries > 0 {
			consOpts = append(consOpts, rmq.WithMaxRetries(cfg.RabbitMQ.MaxRetries))
		}
		if cfg.RabbitMQ.ProcessTimeoutSec > 0 {
			consOpts = append(consOpts, rmq.WithProcessTimeout(time.Duration(cfg.RabbitMQ.ProcessTimeoutSec) * time.Second))
		}
		consOpts = append(consOpts, rmq.WithConsumerMetrics(metrics))
		consOpts = append(consOpts, rmq.WithDLQ(true))

		c := rmq.NewConsumer(connMgr, cd.queue, cd.handler, logger, consOpts...)
		consumerWg.Add(1)
		go func(q string) {
			defer consumerWg.Done()
			if err := c.Start(ctx); err != nil && err != context.Canceled {
				logger.Error().Err(err).Str("queue", q).Msg("consumer exited with error")
			}
		}(cd.queue)
	}

	// --- Start outbox worker goroutine ---
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		if err := outboxWorker.Start(ctx); err != nil && err != context.Canceled {
			logger.Error().Err(err).Msg("outbox worker exited with error")
		}
	}()

	// --- Repositories & Services ---
	storageHandler := storage.NewSupabase(cfg)
	categoryRepo := repository.NewCategoryRepository(db.DB)
	productRepo := repository.NewProductRepository(db.DB, elasticInit)
	cartRepo := repository.NewCartRedisRepository(cfg.NewRedisClient())

	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(db.DB, productRepo, outboxRepo, categoryRepo, logger)
	cartService := service.NewCartService(cartRepo)

	// --- HTTP Server ---
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middlewareGateway.GatewayValidationMiddleware())

	customValidator := validator.NewValidator()
	en.RegisterDefaultTranslations(customValidator.Validator, customValidator.Translator)
	e.Validator = customValidator

	e.GET("/api/check", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	handlers.NewCategoryHandler(e, categoryService, cfg)
	handlers.NewProductHandler(e, cfg, productService, elasticInit)
	handlers.NewUploadImage(e, cfg, storageHandler)
	handlers.NewCartHandler(e, cfg, cartService, productService)

	go func() {
		port := cfg.App.AppPort
		if port == "" {
			port = os.Getenv("APP_PORT")
		}
		if err := e.Start(":" + port); err != nil {
			logger.Info().Err(err).Msg("http server stopped")
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http server shutdown error")
	}

	consumerWg.Wait()

	if err := publisher.Close(); err != nil {
		logger.Error().Err(err).Msg("publisher close error")
	}
	if err := connMgr.Close(); err != nil {
		logger.Error().Err(err).Msg("rabbitmq connection close error")
	}

	snap := metrics.Snapshot()
	logger.Info().
		Int64("published", snap.PublishCount).
		Int64("publish_errors", snap.PublishErrorCount).
		Int64("consumed", snap.ConsumeCount).
		Int64("acks", snap.AckCount).
		Int64("nacks", snap.NackCount).
		Int64("retries", snap.RetryCount).
		Msg("final metrics")

	logger.Info().Msg("shutdown complete")
}

func newLogger(env string) zerolog.Logger {
	if env == "development" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().Timestamp().Caller().Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
