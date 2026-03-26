package repository

import (
	"context"
	"errors"
	"time"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type VerificationTokenRepositoryInterface interface {
	CreateVerificationToken(ctx context.Context, req entity.VerificationTokenEntity) error
	GetDataByToken(ctx context.Context, token string) (*entity.VerificationTokenEntity, error)
}

type verificationTokenRepository struct {
	db *gorm.DB
}

// GetDataByToken implements VerificationTokenRepositoryInterface.
func (v *verificationTokenRepository) GetDataByToken(ctx context.Context, token string) (*entity.VerificationTokenEntity, error) {
	modelToken := model.VerificationToken{}

	if err := v.db.Where("token =?", token).First(&modelToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Errorf("[VerificationTokenRepository-1] GetDataByToken: %v", err)
			return nil, err
		}
		log.Errorf("[VerificationTokenRepository-2] GetDataByToken: %v", err)
		return nil, err
	}

	currentTime := time.Now()
	if currentTime.After(modelToken.ExpiresAt) {
		err := errors.New("401")
		log.Errorf("[VerificationTokenRepository-3] GetDataByToken: %v", err)
		return nil, err
	}

	return &entity.VerificationTokenEntity{
		ID:        modelToken.ID,
		UserID:    modelToken.UserID,
		Token:     token,
		TokenType: modelToken.TokenType,
		ExpiresAt: modelToken.ExpiresAt,
	}, nil
}

// CreateVerificationToken implements VerificationTokenRepositoryInterface.
func (v *verificationTokenRepository) CreateVerificationToken(ctx context.Context, req entity.VerificationTokenEntity) error {
	modelVerificationToken := model.VerificationToken{
		UserID:    req.UserID,
		Token:     req.Token,
		TokenType: req.TokenType,
	}

	if err := v.db.Create(&modelVerificationToken).Error; err != nil {
		log.Errorf("[VerificationTokenRepository-1] CreateVerificationToken: %v", err)
		return err
	}

	return nil
}

func NewVerificationTokenRepository(db *gorm.DB) VerificationTokenRepositoryInterface {
	return &verificationTokenRepository{db: db}
}
