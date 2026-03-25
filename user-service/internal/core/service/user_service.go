package service

import (
	"context"
	"errors"
	"user-service/internal/adapter/repository"
	"user-service/internal/core/domain/entity"
	"user-service/utils/conv"

	"github.com/labstack/gommon/log"
)

type UserServiceInterface interface {
	SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error)
}

type UserService struct {
	repo repository.UserRepositoryInterface
}

// SignIn implements [UserServiceInterface].
func (u *UserService) SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error) {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Errorf("Failed to get user by email: %v", err)
		return nil, "", err
	}

	if !conv.CheckPasswordHash(req.Password, user.Password) {
		log.Error("Failed to check password: invalid password")
		return nil, "", errors.New("invalid password")
	}

	return user, "", nil
}

func NewUserService(repo repository.UserRepositoryInterface) UserServiceInterface {
	return &UserService{
		repo: repo,
	}
}
