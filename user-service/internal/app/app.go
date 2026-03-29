package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user-service/config"
	"user-service/internal/adapter/handler"
	"user-service/internal/adapter/message"
	rmq "user-service/internal/adapter/rabbitmq"
	"user-service/internal/adapter/repository"
	"user-service/internal/adapter/storage"
	"user-service/internal/core/service"
	"user-service/utils"
	"user-service/utils/validator"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	middlewareGateway "user-service/internal/middleware"
)

func RunServer() {
	cfg := config.NewConfig()
	logger := newLogger(cfg.App.AppEnv)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
		return
	}

	// --- RabbitMQ setup ---
	connMgr := rmq.NewConnectionManager(cfg.RabbitMQURL(), logger)

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

	queues := []string{
		utils.NOTIF_EMAIL_VERIFICATION,
		utils.NOTIF_EMAIL_FORGOT_PASSWORD,
		utils.NOTIF_EMAIL_CREATE_CUSTOMER,
		utils.NOTIF_EMAIL_UPDATE_CUSTOMER,
		utils.PUSH_NOTIF,
	}
	for _, q := range queues {
		if err := publisher.SetupQueue(q, nil); err != nil {
			logger.Warn().Err(err).Str("queue", q).Msg("queue setup failed (consumer may create it)")
		}
	}

	eventPub := message.NewEventPublisher(publisher, logger)

	// --- Repositories & Services ---
	storageHandler := storage.NewSupabase(cfg)
	userRepo := repository.NewUserRepository(db.DB)
	tokenRepo := repository.NewVerificationTokenRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)

	jwtService := service.NewJwtService(cfg)
	userService := service.NewUserService(userRepo, cfg, jwtService, tokenRepo, eventPub)
	roleService := service.NewRoleService(roleRepo)

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

	handler.NewUserHandler(e, userService, cfg, jwtService)
	handler.NewUploadImage(e, cfg, storageHandler, jwtService)
	handler.NewRoleHandler(e, roleService, cfg, jwtService)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http server shutdown error")
	}

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
