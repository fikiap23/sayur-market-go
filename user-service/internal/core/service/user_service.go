package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"user-service/config"
	"user-service/internal/adapter/repository"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"
	"user-service/utils"
	"user-service/utils/conv"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type UserServiceInterface interface {
	SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error)
	CreateUserAccount(ctx context.Context, req entity.UserEntity) error
	ForgotPassword(ctx context.Context, req entity.UserEntity) error
	VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error)
	UpdatePassword(ctx context.Context, req entity.UserEntity) error
	GetProfileUser(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateDataUser(ctx context.Context, req entity.UserEntity) error

	GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error)
	GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error)
	CreateCustomer(ctx context.Context, req entity.UserEntity) error
	UpdateCustomer(ctx context.Context, req entity.UserEntity) error
	DeleteCustomer(ctx context.Context, customerID int64) error
}

type userService struct {
	db         *gorm.DB
	repo       repository.UserRepositoryInterface
	cfg        *config.Config
	jwtService JwtServiceInterface
	repoToken  repository.VerificationTokenRepositoryInterface
	outboxRepo repository.OutboxRepositoryInterface
	logger     zerolog.Logger
}

func (u *userService) DeleteCustomer(ctx context.Context, customerID int64) error {
	return u.repo.DeleteCustomer(ctx, customerID)
}

// UpdateCustomer updates a customer. If the password was changed, a notification
// event is written to the outbox atomically with the update.
func (u *userService) UpdateCustomer(ctx context.Context, req entity.UserEntity) error {
	passwordNoencrypt := ""
	if req.Password != "" {
		passwordNoencrypt = req.Password
		password, err := conv.HashPassword(req.Password)
		if err != nil {
			u.logger.Error().Err(err).Msg("hash password failed")
			return err
		}
		req.Password = password
	}

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := u.repo.WithTx(tx)

		if err := txRepo.UpdateCustomer(ctx, req); err != nil {
			return err
		}

		if passwordNoencrypt != "" {
			msg := fmt.Sprintf("You're account has been updated. Please login use: \n Email: %s\nPassword: %s", req.Email, passwordNoencrypt)
			if err := u.insertNotifOutbox(tx, req.ID, req.Email, msg, utils.NOTIF_EMAIL_UPDATE_CUSTOMER, "Updated Data"); err != nil {
				u.logger.Error().Err(err).Msg("insert outbox for update customer failed")
				return err
			}
		}

		return nil
	})
}

// CreateCustomer creates a customer and writes a notification outbox event
// in the same transaction.
func (u *userService) CreateCustomer(ctx context.Context, req entity.UserEntity) error {
	passwordNoEncrypt := req.Password
	password, err := conv.HashPassword(passwordNoEncrypt)
	if err != nil {
		u.logger.Error().Err(err).Msg("hash password failed")
		return err
	}
	req.Password = password

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := u.repo.WithTx(tx)

		userID, err := txRepo.CreateCustomer(ctx, req)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("You have been registered in Sayur Project. Please login use: \n Email: %s\nPassword: %s", req.Email, passwordNoEncrypt)
		if err := u.insertNotifOutbox(tx, userID, req.Email, msg, utils.NOTIF_EMAIL_CREATE_CUSTOMER, "Account Exists"); err != nil {
			return err
		}

		return nil
	})
}

func (u *userService) GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error) {
	return u.repo.GetCustomerByID(ctx, customerID)
}

func (u *userService) GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error) {
	return u.repo.GetCustomerAll(ctx, query)
}

func (u *userService) UpdateDataUser(ctx context.Context, req entity.UserEntity) error {
	return u.repo.UpdateDataUser(ctx, req)
}

func (u *userService) GetProfileUser(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	return u.repo.GetUserByID(ctx, userID)
}

func (u *userService) UpdatePassword(ctx context.Context, req entity.UserEntity) error {
	token, err := u.repoToken.GetDataByToken(ctx, req.Token)
	if err != nil {
		return err
	}

	if token.TokenType != "reset_password" {
		return errors.New("401")
	}

	password, err := conv.HashPassword(req.Password)
	if err != nil {
		return err
	}
	req.Password = password
	req.ID = token.UserID

	return u.repo.UpdatePasswordByID(ctx, req)
}

func (u *userService) VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error) {
	verifyToken, err := u.repoToken.GetDataByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	user, err := u.repo.UpdateUserVerified(ctx, verifyToken.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := u.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, err
	}

	sessionData := map[string]interface{}{
		"user_id":    user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"logged_in":  true,
		"created_at": time.Now().String(),
		"token":      token,
		"role_name":  user.RoleName,
	}

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return nil, err
	}

	redisConn := config.NewConfig().NewRedisClient()
	if err := redisConn.Set(ctx, token, jsonData, time.Hour*23).Err(); err != nil {
		return nil, err
	}

	user.Token = accessToken
	return user, nil
}

// ForgotPassword creates a verification token and writes a notification
// outbox event atomically.
func (u *userService) ForgotPassword(ctx context.Context, req entity.UserEntity) error {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return err
	}

	token := uuid.New().String()

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txTokenRepo := u.repoToken.WithTx(tx)

		reqEntity := entity.VerificationTokenEntity{
			UserID:    user.ID,
			Token:     token,
			TokenType: utils.NOTIF_EMAIL_FORGOT_PASSWORD,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		if err := txTokenRepo.CreateVerificationToken(ctx, reqEntity); err != nil {
			return err
		}

		urlForgot := fmt.Sprintf("%s/auth/update-password?token=%s", u.cfg.App.UrlFrontFE, token)
		msg := fmt.Sprintf("Please click link below for reset password: %v", urlForgot)
		if err := u.insertNotifOutbox(tx, user.ID, req.Email, msg, utils.NOTIF_EMAIL_FORGOT_PASSWORD, "Reset Password"); err != nil {
			return err
		}

		return nil
	})
}

// CreateUserAccount creates a user account and writes a verification email
// outbox event in the same transaction.
func (u *userService) CreateUserAccount(ctx context.Context, req entity.UserEntity) error {
	password, err := conv.HashPassword(req.Password)
	if err != nil {
		return err
	}
	req.Password = password
	req.Token = uuid.New().String()

	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := u.repo.WithTx(tx)

		userID, err := txRepo.CreateUserAccount(ctx, req)
		if err != nil {
			return err
		}

		verifyURL := fmt.Sprintf("%s/auth/verify-account?token=%s", u.cfg.App.UrlFrontFE, req.Token)
		msg := fmt.Sprintf("Please verify your account by clicking the link: %s", verifyURL)
		if err := u.insertNotifOutbox(tx, userID, req.Email, msg, utils.NOTIF_EMAIL_VERIFICATION, "Verify Your Account"); err != nil {
			return err
		}

		return nil
	})
}

func (u *userService) SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error) {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", err
	}

	if checkPass := conv.CheckPasswordHash(req.Password, user.Password); !checkPass {
		return nil, "", errors.New("password is incorrect")
	}

	token, err := u.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	sessionData := map[string]interface{}{
		"user_id":    user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"logged_in":  true,
		"created_at": time.Now().String(),
		"token":      token,
		"role_name":  user.RoleName,
	}

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return nil, "", err
	}

	redisConn := config.NewConfig().NewRedisClient()
	if err := redisConn.Set(ctx, token, jsonData, time.Hour*23).Err(); err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// insertNotifOutbox builds a notification payload and writes it to the outbox
// table using the provided DB handle (transaction-safe).
func (u *userService) insertNotifOutbox(tx *gorm.DB, userID int64, email, msg, queueName, subject string) error {
	notifType := "EMAIL"
	if queueName == utils.PUSH_NOTIF {
		notifType = "PUSH"
	}

	payload := map[string]interface{}{
		"receiver_email":    email,
		"message":           msg,
		"receiver_id":       userID,
		"subject":           subject,
		"notification_type": notifType,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	event := &model.OutboxEvent{
		ID:        uuid.New().String(),
		EventType: queueName,
		Payload:   string(data),
		Status:    model.OutboxStatusPending,
	}
	return u.outboxRepo.Insert(tx, event)
}

func NewUserService(
	db *gorm.DB,
	repo repository.UserRepositoryInterface,
	cfg *config.Config,
	jwtService JwtServiceInterface,
	repoToken repository.VerificationTokenRepositoryInterface,
	outboxRepo repository.OutboxRepositoryInterface,
	logger zerolog.Logger,
) UserServiceInterface {
	return &userService{
		db:         db,
		repo:       repo,
		cfg:        cfg,
		jwtService: jwtService,
		repoToken:  repoToken,
		outboxRepo: outboxRepo,
		logger:     logger.With().Str("component", "user_service").Logger(),
	}
}
