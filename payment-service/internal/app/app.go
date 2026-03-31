package app

import (
	"context"
	"os"
	"os/signal"
	"payment-service/config"
	"payment-service/internal/adapter/handlers"
	httpclient "payment-service/internal/adapter/http_client"
	"payment-service/internal/adapter/outbox"
	rmq "payment-service/internal/adapter/rabbitmq"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/service"
	"payment-service/utils/validator"
	"sync"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	middlewareGateway "payment-service/internal/middleware"
)

func RunServer() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
		return
	}

	paymentRepo := repository.NewPaymentRepository(db.DB)
	outboxRepo := repository.NewOutboxRepository(db.DB)

	httpClient := httpclient.NewHttpClient(cfg)
	midtrans := httpclient.NewMidtransClient(cfg)

	// --- RabbitMQ setup (shared connection + pooled publisher, same pattern as product/user-service) ---
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

	if cfg.PublisherName.PaymentSuccess != "" {
		paymentDLQ := cfg.PublisherName.PaymentSuccess + ".dlq"
		if err := publisher.SetupQueue(paymentDLQ, nil); err != nil {
			logger.Warn().Err(err).Str("queue", paymentDLQ).Msg("payment success DLQ setup failed")
		}
		paymentArgs := amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": paymentDLQ,
		}
		if err := publisher.SetupQueue(cfg.PublisherName.PaymentSuccess, paymentArgs); err != nil {
			logger.Warn().Err(err).Str("queue", cfg.PublisherName.PaymentSuccess).Msg("payment success queue setup failed")
		}
	}

	outboxWorkerCfg := outbox.DefaultWorkerConfig()
	outboxWorker := outbox.NewWorker(outboxRepo, publisher, outboxWorkerCfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var workerWg sync.WaitGroup
	workerWg.Add(1)
	go func() {
		defer workerWg.Done()
		if err := outboxWorker.Start(ctx); err != nil && err != context.Canceled {
			logger.Error().Err(err).Msg("outbox worker exited with error")
		}
	}()

	paymentService := service.NewPaymentService(db.DB, paymentRepo, outboxRepo, cfg, httpClient, midtrans)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middlewareGateway.GatewayValidationMiddleware())

	customValidator := validator.NewValidator()
	en.RegisterDefaultTranslations(customValidator.Validator, customValidator.Translator)
	e.Validator = customValidator

	e.GET("/api/check", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	handlers.NewPaymentHandler(paymentService, e, cfg)

	go func() {
		if cfg.App.AppPort == "" {
			cfg.App.AppPort = os.Getenv("APP_PORT")
		}

		err = e.Start(":" + cfg.App.AppPort)
		if err != nil {
			logger.Info().Err(err).Msg("http server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	<-quit

	logger.Info().Msg("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http server shutdown error")
	}

	workerWg.Wait()

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
