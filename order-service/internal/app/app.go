package app

import (
	"context"
	"order-service/internal/adapter/message"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"order-service/config"
	"order-service/internal/adapter/handlers"
	httpclient "order-service/internal/adapter/http_client"
	"order-service/internal/adapter/outbox"
	rmq "order-service/internal/adapter/rabbitmq"
	"order-service/internal/adapter/repository"
	"order-service/internal/core/domain/model"
	"order-service/internal/core/service"
	"order-service/utils"
	"order-service/utils/validator"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	middlewareGateway "order-service/internal/middleware"
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

	outboxWorkerCfg := outbox.DefaultWorkerConfig()
	outboxRepo := repository.NewOutboxRepository(db.DB)
	outboxWorker := outbox.NewWorker(outboxRepo, publisher, outboxWorkerCfg, logger)

	exchangeBindings := map[string]string{}
	if cfg.PublisherName.OrderPublish != "" {
		exchangeBindings[model.EventOrderCreated] = cfg.PublisherName.OrderPublish
	}
	if cfg.PublisherName.PublisherDeleteOrder != "" {
		exchangeBindings[model.EventOrderDeleted] = cfg.PublisherName.PublisherDeleteOrder
	}
	if cfg.PublisherName.ProductUpdateStock != "" {
		exchangeBindings[model.EventOrderStockUpdate] = cfg.PublisherName.ProductUpdateStock
	}
	if cfg.PublisherName.EmailUpdateStatus != "" {
		exchangeBindings[model.EventNotificationOrderEmail] = cfg.PublisherName.EmailUpdateStatus
	}
	exchangeBindings[model.EventNotificationOrderPush] = utils.PUSH_NOTIF
	if cfg.PublisherName.PublisherUpdateStatus != "" {
		exchangeBindings[model.EventOrderStatusES] = cfg.PublisherName.PublisherUpdateStatus
	}

	if err := outboxWorker.SetupExchange(connMgr, exchangeBindings); err != nil {
		logger.Fatal().Err(err).Msg("failed to setup exchange and bindings")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	type consumerDef struct {
		queue   string
		handler rmq.MessageHandler
	}

	var consumers []consumerDef
	if cfg.PublisherName.OrderPublish != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.OrderPublish,
			handler: message.NewOrderESIndexHandler(elasticInit, logger),
		})
	}
	if cfg.PublisherName.PublisherPaymentSuccess != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.PublisherPaymentSuccess,
			handler: message.NewOrderPaymentSuccessHandler(elasticInit, logger),
		})
	}
	if cfg.PublisherName.PublisherUpdateStatus != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.PublisherUpdateStatus,
			handler: message.NewOrderUpdateStatusHandler(elasticInit, logger),
		})
	}
	if cfg.PublisherName.PublisherDeleteOrder != "" {
		consumers = append(consumers, consumerDef{
			queue:   cfg.PublisherName.PublisherDeleteOrder,
			handler: message.NewOrderESDeleteHandler(elasticInit, logger),
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
			consOpts = append(consOpts, rmq.WithProcessTimeout(time.Duration(cfg.RabbitMQ.ProcessTimeoutSec)*time.Second))
		}
		consOpts = append(consOpts, rmq.WithConsumerMetrics(metrics))
		consOpts = append(consOpts, rmq.WithDLQ(true))

		c := rmq.NewConsumer(connMgr, cd.queue, cd.handler, logger, consOpts...)
		wg.Add(1)
		go func(q string) {
			defer wg.Done()
			if err := c.Start(ctx); err != nil && err != context.Canceled {
				logger.Error().Err(err).Str("queue", q).Msg("consumer exited with error")
			}
		}(cd.queue)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := outboxWorker.Start(ctx); err != nil && err != context.Canceled {
			logger.Error().Err(err).Msg("outbox worker exited with error")
		}
	}()

	orderRepo := repository.NewOrderRepository(db.DB)
	elasticRepo := repository.NewElasticRepository(elasticInit)
	httpClient := httpclient.NewHttpClient(cfg)

	orderService := service.NewOrderService(db.DB, orderRepo, outboxRepo, cfg, httpClient, elasticRepo, logger)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middlewareGateway.GatewayValidationMiddleware())

	customValidator := validator.NewValidator()
	en.RegisterDefaultTranslations(customValidator.Validator, customValidator.Translator)
	e.Validator = customValidator

	e.GET("/api/check", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	handlers.NewOrderHandler(orderService, e, cfg)

	go func() {
		port := cfg.App.AppPort
		if port == "" {
			port = os.Getenv("APP_PORT")
		}
		if err := e.Start(":" + port); err != nil {
			logger.Info().Err(err).Msg("http server stopped")
		}
	}()

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

	wg.Wait()

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
