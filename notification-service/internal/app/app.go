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
	"notification-service/internal/adapter/rabbitmq"
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

	notifRepo := repository.NewNotificationRepository(db.DB)
	notifService := service.NewNotificationService(notifRepo)
	emailMessage := message.NewMessageEmail(cfg)

	notifHandler := rabbitmq.NewNotificationHandler(emailMessage, notifRepo, notifService, logger)

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
		consumer := rabbitmq.NewConsumer(cfg, queue, notifHandler, logger,
			rabbitmq.WithWorkerPoolSize(cfg.RabbitMQ.WorkerPoolSize),
			rabbitmq.WithPrefetchCount(cfg.RabbitMQ.PrefetchCount),
			rabbitmq.WithMaxRetries(cfg.RabbitMQ.MaxRetries),
			rabbitmq.WithProcessTimeout(time.Duration(cfg.RabbitMQ.ProcessTimeout)*time.Second),
		)

		consumerWg.Add(1)
		go func(q string) {
			defer consumerWg.Done()
			if err := consumer.Start(ctx); err != nil && err != context.Canceled {
				logger.Error().Err(err).Str("queue", q).Msg("consumer exited with error")
			}
		}(queue)
	}

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
	logger.Info().Msg("shutdown complete")
}

func newLogger(env string) zerolog.Logger {
	if env == "development" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().Timestamp().Caller().Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
