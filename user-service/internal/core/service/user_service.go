package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"user-service/config"
	"user-service/internal/adapter/message"
	"user-service/internal/adapter/repository"
	"user-service/internal/core/domain/entity"
	"user-service/utils"
	"user-service/utils/conv"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
)

type UserServiceInterface interface {
	SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error)
	CreateUserAccount(ctx context.Context, req entity.UserEntity) error
	ForgotPassword(ctx context.Context, req entity.UserEntity) error
	VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error)
	UpdatePassword(ctx context.Context, req entity.UserEntity) error
	GetProfileUser(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateDataUser(ctx context.Context, req entity.UserEntity) error

	// Modul Customers Admin
	GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error)
	GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error)
	CreateCustomer(ctx context.Context, req entity.UserEntity) error
	UpdateCustomer(ctx context.Context, req entity.UserEntity) error
	DeleteCustomer(ctx context.Context, customerID int64) error
}

type userService struct {
	repo       repository.UserRepositoryInterface
	cfg        *config.Config
	jwtService JwtServiceInterface
	repoToken  repository.VerificationTokenRepositoryInterface
}

// DeleteCustomer implements UserServiceInterface.
func (u *userService) DeleteCustomer(ctx context.Context, customerID int64) error {
	return u.repo.DeleteCustomer(ctx, customerID)
}

// UpdateCustomer implements UserServiceInterface.
func (u *userService) UpdateCustomer(ctx context.Context, req entity.UserEntity) error {
	passwordNoencrypt := ""
	if req.Password != "" {
		passwordNoencrypt = req.Password
		password, err := conv.HashPassword(req.Password)
		if err != nil {
			log.Fatalf("[UserService-1] UpdateCustomer: %v", err)
			return err
		}

		req.Password = password
	}

	err := u.repo.UpdateCustomer(ctx, req)
	if err != nil {
		log.Fatalf("[UserService-2] UpdateCustomer: %v", err)
		return err
	}

	if passwordNoencrypt != "" {
		messageparam := fmt.Sprintf("You're account has been updated. Please login use: \n Email: %s\nPassword: %s", req.Email, passwordNoencrypt)
		go message.PublishMessage(req.ID,
			req.Email,
			messageparam,
			utils.NOTIF_EMAIL_UPDATE_CUSTOMER,
			"Updated Data")
	}

	return nil
}

// CreateCustomer implements UserServiceInterface.
func (u *userService) CreateCustomer(ctx context.Context, req entity.UserEntity) error {
	passwordNoEncrypt := req.Password
	password, err := conv.HashPassword(passwordNoEncrypt)
	if err != nil {
		log.Fatalf("[UserService-1] CreateCustomer: %v", err)
		return err
	}

	req.Password = password
	userID, err := u.repo.CreateCustomer(ctx, req)
	if err != nil {
		log.Fatalf("[UserService-2] CreateCustomer: %v", err)
		return err
	}

	messageparam := fmt.Sprintf("You have been registered in Sayur Project. Please login use: \n Email: %s\nPassword: %s", req.Email, passwordNoEncrypt)
	go message.PublishMessage(userID,
		req.Email,
		messageparam,
		utils.NOTIF_EMAIL_CREATE_CUSTOMER,
		"Account Exists")

	return nil
}

// GetCustomerByID implements UserServiceInterface.
func (u *userService) GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error) {
	return u.repo.GetCustomerByID(ctx, customerID)
}

// GetCustomerAll implements UserServiceInterface.
func (u *userService) GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error) {
	return u.repo.GetCustomerAll(ctx, query)
}

// UpdateDataUser implements UserServiceInterface.
func (u *userService) UpdateDataUser(ctx context.Context, req entity.UserEntity) error {
	return u.repo.UpdateDataUser(ctx, req)
}

// GetProfileUser implements UserServiceInterface.
func (u *userService) GetProfileUser(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	return u.repo.GetUserByID(ctx, userID)
}

// UpdatePassword implements UserServiceInterface.
func (u *userService) UpdatePassword(ctx context.Context, req entity.UserEntity) error {
	token, err := u.repoToken.GetDataByToken(ctx, req.Token)
	if err != nil {
		log.Errorf("[UserService-1] UpdatePassword: %v", err)
		return err
	}

	if token.TokenType != "reset_password" {
		err = errors.New("401")
		log.Errorf("[UserService-2] UpdatePassword: %v", err)
		return err
	}

	password, err := conv.HashPassword(req.Password)
	if err != nil {
		log.Errorf("[UserService-3] UpdatePassword: %v", err)
		return err
	}
	req.Password = password
	req.ID = token.UserID

	err = u.repo.UpdatePasswordByID(ctx, req)
	if err != nil {
		log.Errorf("[UserService-4] UpdatePassword: %v", err)
		return err
	}

	return nil
}

// VerifyToken implements UserServiceInterface.
func (u *userService) VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error) {
	verifyToken, err := u.repoToken.GetDataByToken(ctx, token)
	if err != nil {
		log.Errorf("[UserService-1] VerifyToken: %v", err)
		return nil, err
	}

	user, err := u.repo.UpdateUserVerified(ctx, verifyToken.UserID)
	if err != nil {
		log.Errorf("[UserService-2] VerifyToken: %v", err)
		return nil, err
	}

	accessToken, err := u.jwtService.GenerateToken(user.ID)
	if err != nil {
		log.Errorf("[UserService-3] VerifyToken: %v", err)
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
		fmt.Println("Error encoding JSON:", err)
		return nil, err
	}

	redisConn := config.NewConfig().NewRedisClient()
	err = redisConn.Set(ctx, token, jsonData, time.Hour*23).Err()
	if err != nil {
		log.Errorf("[UserService-4] VerifyToken: %v", err)
		return nil, err
	}

	user.Token = accessToken

	return user, nil
}

// ForgotPassword implements UserServiceInterface.
func (u *userService) ForgotPassword(ctx context.Context, req entity.UserEntity) error {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Errorf("[UserService-1] ForgotPassword: %v", err)
		return err
	}

	token := uuid.New().String()
	reqEntity := entity.VerificationTokenEntity{
		UserID:    user.ID,
		Token:     token,
		TokenType: utils.NOTIF_EMAIL_FORGOT_PASSWORD,
	}

	err = u.repoToken.CreateVerificationToken(ctx, reqEntity)
	if err != nil {
		log.Errorf("[UserService-2] ForgotPassword: %v", err)
		return err
	}

	urlForgot := fmt.Sprintf("%s/auth/update-password?token=%s", u.cfg.App.UrlFrontFE, token)
	messageparam := fmt.Sprintf("Please click link below for reset password: %v", urlForgot)
	go message.PublishMessage(user.ID,
		req.Email,
		messageparam,
		utils.NOTIF_EMAIL_FORGOT_PASSWORD,
		"Reset Password")

	return nil
}

// CreateUserAccount implements UserServiceInterface.
func (u *userService) CreateUserAccount(ctx context.Context, req entity.UserEntity) error {
	password, err := conv.HashPassword(req.Password)
	if err != nil {
		log.Errorf("[UserService-1] CreateUserAccount: %v", err)
		return err
	}

	req.Password = password
	req.Token = uuid.New().String()

	userID, err := u.repo.CreateUserAccount(ctx, req)
	if err != nil {
		log.Errorf("[UserService-2] CreateUserAccount: %v", err)
		return err
	}

	verifyURL := fmt.Sprintf("%s/auth/verify-account?token=%s", u.cfg.App.UrlFrontFE, req.Token)
	verifyMsg := fmt.Sprintf("Please verify your account by clicking the link: %s", verifyURL)
	go message.PublishMessage(
		userID,
		req.Email,
		verifyMsg,
		utils.NOTIF_EMAIL_VERIFICATION,
		"Verify Your Account",
	)

	return nil
}

// SignIn implements UserServiceInterface.
func (u *userService) SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error) {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Errorf("[UserService-1] SignIn: %v", err)
		return nil, "", err
	}

	if checkPass := conv.CheckPasswordHash(req.Password, user.Password); !checkPass {
		err = errors.New("password is incorrect")
		log.Errorf("[UserService-2] SignIn: %v", err)
		return nil, "", err
	}

	token, err := u.jwtService.GenerateToken(user.ID)
	if err != nil {
		log.Errorf("[UserService-3] SignIn: %v", err)
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
		fmt.Println("Error encoding JSON:", err)
		return nil, "", err
	}

	redisConn := config.NewConfig().NewRedisClient()
	err = redisConn.Set(ctx, token, jsonData, time.Hour*23).Err()
	if err != nil {
		log.Errorf("[UserService-4] SignIn: %v", err)
		return nil, "", err
	}

	return user, token, nil
}

func NewUserService(repo repository.UserRepositoryInterface, cfg *config.Config, jwtService JwtServiceInterface, repoToken repository.VerificationTokenRepositoryInterface) UserServiceInterface {
	return &userService{
		repo:       repo,
		cfg:        cfg,
		jwtService: jwtService,
		repoToken:  repoToken,
	}
}
