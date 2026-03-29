package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"notification-service/config"
	"notification-service/internal/adapter/handlers"
	"notification-service/internal/adapter/message"
	rmq "notification-service/internal/adapter/rabbitmq"
	"notification-service/internal/adapter/repository"
	"notification-service/internal/core/service"
	"notification-service/utils"

	middlewareGateway "notification-service/internal/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func RunServer() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
		return
	}

	// --- Dependencies ---
	notifRepo := repository.NewNotificationRepository(db.DB)
	notifService := service.NewNotificationService(notifRepo)
	emailMessage := message.NewMessageEmail(cfg)

	notifHandler := rmq.NewNotificationHandler(emailMessage, notifRepo, notifService, logger)

	// --- RabbitMQ setup ---
	connMgr := rmq.NewConnectionManager(cfg.RabbitMQURL(), logger)
	metrics := rmq.NewMetrics()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queues := []string{
		utils.NOTIF_EMAIL_VERIFICATION,
		utils.NOTIF_EMAIL_FORGOT_PASSWORD,
		utils.NOTIF_EMAIL_CREATE_CUSTOMER,
		utils.NOTIF_EMAIL_UPDATE_STATUS_ORDER,
		utils.PUSH_NOTIF,
	}

	var consumerWg sync.WaitGroup
	for _, queue := range queues {
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

		consumer := rmq.NewConsumer(connMgr, queue, notifHandler, logger, consOpts...)

		consumerWg.Add(1)
		go func(q string) {
			defer consumerWg.Done()
			if err := consumer.Start(ctx); err != nil && err != context.Canceled {
				logger.Error().Err(err).Str("queue", q).Msg("consumer exited with error")
			}
		}(queue)
	}

	// --- HTTP Server ---
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middlewareGateway.GatewayValidationMiddleware())

	handlers.NewNotificationHandler(notifService, e, cfg)

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

	if err := connMgr.Close(); err != nil {
		logger.Error().Err(err).Msg("rabbitmq connection close error")
	}

	snap := metrics.Snapshot()
	logger.Info().
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
