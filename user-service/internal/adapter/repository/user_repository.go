package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type UserRepositoryInterface interface {
	GetUserByEmail(ctx context.Context, email string) (*entity.UserEntity, error)
	CreateUserAccount(ctx context.Context, req entity.UserEntity) (int64, error)
	UpdateUserVerified(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdatePasswordByID(ctx context.Context, req entity.UserEntity) error
	GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateDataUser(ctx context.Context, req entity.UserEntity) error

	// Modul Customers Admin
	GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error)
	GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error)
	CreateCustomer(ctx context.Context, req entity.UserEntity) (int64, error)
	UpdateCustomer(ctx context.Context, req entity.UserEntity) error
	DeleteCustomer(ctx context.Context, customerID int64) error
}

type userRepository struct {
	db *gorm.DB
}

// DeleteCustomer implements UserRepositoryInterface.
func (u *userRepository) DeleteCustomer(ctx context.Context, customerID int64) error {
	modelUser := model.User{}
	if err := u.db.Where("id =?", customerID).First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[UserRepository-1] DeleteCustomer: User not found")
			return err
		}
		log.Errorf("[UserRepository-2] DeleteCustomer: %v", err)
		return err
	}

	if err := u.db.Delete(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-3] DeleteCustomer: %v", err)
		return err
	}
	return nil
}

// UpdateCustomer implements UserRepositoryInterface.
func (u *userRepository) UpdateCustomer(ctx context.Context, req entity.UserEntity) error {
	modelRole := model.Role{}

	if err := u.db.Where("id =?", req.RoleID).First(&modelRole).Error; err != nil {
		log.Fatalf("[UserRepository-1] UpdateCustomer: %v", err)
		return err
	}

	modelUser := model.User{}
	if err := u.db.Where("id =?", req.ID).First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[UserRepository-2] UpdateCustomer: User not found")
			return err
		}
		log.Errorf("[UserRepository-3] UpdateCustomer: %v", err)
		return err
	}

	modelUser.Name = req.Name
	modelUser.Email = req.Email
	modelUser.Phone = req.Phone
	modelUser.Roles = []model.Role{modelRole}
	if req.Address != "" {
		modelUser.Address = req.Address
	}

	if req.Lat != "" {
		modelUser.Lat = req.Lat
	}

	if req.Lng != "" {
		modelUser.Lng = req.Lng
	}
	if req.Photo != "" {
		modelUser.Lat = req.Lat
	}

	if req.Password != "" {
		modelUser.Password = req.Password
	}

	if err := u.db.Save(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-4] UpdateCustomer: %v", err)
		return err
	}

	return nil
}

// CreateCustomer implements UserRepositoryInterface.
func (u *userRepository) CreateCustomer(ctx context.Context, req entity.UserEntity) (int64, error) {
	modelRole := model.Role{}

	if err := u.db.Where("id =?", req.RoleID).First(&modelRole).Error; err != nil {
		log.Fatalf("[UserRepository-1] CreateCustomer: %v", err)
		return 0, err
	}

	modelUser := model.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   req.Password,
		Address:    req.Address,
		Lat:        req.Lat,
		Lng:        req.Lng,
		Phone:      req.Phone,
		Photo:      req.Photo,
		Roles:      []model.Role{modelRole},
		IsVerified: true,
	}

	if err := u.db.Create(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-2] CreateCustomer: %v", err)
		return 0, err
	}

	return modelUser.ID, nil
}

// GetCustomerByID implements UserRepositoryInterface.
func (u *userRepository) GetCustomerByID(ctx context.Context, customerID int64) (*entity.UserEntity, error) {
	modelUser := model.User{}

	if err := u.db.Where("id = ?", customerID).Preload("Roles").First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[UserRepository-1] GetCustomerByID: User not found")
			return nil, err
		}
		log.Errorf("[UserRepository-2] GetCustomerByID: %v", err)
		return nil, err
	}

	roleID := 0
	for _, role := range modelUser.Roles {
		roleID = int(role.ID)
	}

	return &entity.UserEntity{
		ID:      customerID,
		Name:    modelUser.Name,
		Email:   modelUser.Email,
		RoleID:  int64(roleID),
		Address: modelUser.Address,
		Lat:     modelUser.Lat,
		Lng:     modelUser.Lng,
		Phone:   modelUser.Phone,
		Photo:   modelUser.Photo,
	}, nil
}

// GetCustomerAll implements UserRepositoryInterface.
func (u *userRepository) GetCustomerAll(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error) {
	modelUsers := []model.User{}
	var countData int64

	order := fmt.Sprintf("%s %s", query.OrderBy, query.OrderType)
	offset := (query.Page - 1) * query.Limit

	sqlMain := u.db.Preload("Roles", "name = ?", "Customer").
		Where("name ILIKE ? OR email ILIKE ? OR phone ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%")

	if err := sqlMain.Model(&modelUsers).Count(&countData).Error; err != nil {
		log.Errorf("[UserRepository-1] GetCustomerAll: %v", err)
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(query.Limit)))

	if err := sqlMain.Order(order).Limit(int(query.Limit)).Offset(int(offset)).Find(&modelUsers).Error; err != nil {
		log.Errorf("[UserRepository-3] GetCustomerAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelUsers) < 1 {
		err := errors.New("404")
		log.Infof("[UserRepository-4] GetCustomerAll: No Customer found")
		return nil, 0, 0, err
	}

	respEntities := []entity.UserEntity{}
	for _, val := range modelUsers {
		roleName := ""
		for _, role := range val.Roles {
			roleName = role.Name
		}
		respEntities = append(respEntities, entity.UserEntity{
			ID:       val.ID,
			Name:     val.Name,
			Email:    val.Email,
			RoleName: roleName,
			Phone:    val.Phone,
			Photo:    val.Photo,
		})
	}
	return respEntities, countData, int64(totalPage), nil
}

// UpdateDataUser implements UserRepositoryInterface.
func (u *userRepository) UpdateDataUser(ctx context.Context, req entity.UserEntity) error {
	modelUser := model.User{
		Name:    req.Name,
		Email:   req.Email,
		Address: req.Address,
		Phone:   req.Phone,
		Photo:   req.Photo,
	}

	if err := u.db.Where("id = ? AND is_verified = true", req.ID).First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Errorf("[UserRepository-1] UpdateDataUser: %v", err)
			return err
		}
		log.Errorf("[UserRepository-2] UpdateDataUser: %v", err)
		return err
	}

	modelUser.Lat = req.Lat
	modelUser.Lng = req.Lng
	modelUser.Address = req.Address
	modelUser.Phone = req.Phone

	if err := u.db.UpdateColumns(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-3] UpdateDataUser: %v", err)
		return err
	}

	return nil
}

// GetUserByID implements UserRepositoryInterface.
func (u *userRepository) GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	modelUser := model.User{}

	if err := u.db.Where("id =? AND is_verified = true", userID).Preload("Roles").First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Errorf("[UserRepository-1] GetUserByID: %v", err)
			return nil, err
		}
		log.Errorf("[UserRepository-2] GetUserByID: %v", err)
		return nil, err
	}

	return &entity.UserEntity{
		ID:       modelUser.ID,
		Email:    modelUser.Email,
		Name:     modelUser.Name,
		RoleName: modelUser.Roles[0].Name,
		Lat:      modelUser.Lat,
		Lng:      modelUser.Lng,
		Address:  modelUser.Address,
		Phone:    modelUser.Phone,
		Photo:    modelUser.Photo,
	}, nil
}

// UpdatePasswordByID implements UserRepositoryInterface.
func (u *userRepository) UpdatePasswordByID(ctx context.Context, req entity.UserEntity) error {
	modelUser := model.User{}

	if err := u.db.Where("id =?", req.ID).First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Errorf("[UserRepository-1] UpdatePasswordByID: %v", err)
			return err
		}
		log.Errorf("[UserRepository-2] UpdatePasswordByID: %v", err)
		return err
	}

	modelUser.Password = req.Password
	if err := u.db.Save(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-3] UpdatePasswordByID: %v", err)
		return err
	}

	return nil
}

// UpdateUserVerified implements UserRepositoryInterface.
func (u *userRepository) UpdateUserVerified(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	modelUser := model.User{}

	if err := u.db.Where("id = ?", userID).Preload("Roles").First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Errorf("[UserRepository-1] UpdateUserVerified: %v", err)
			return nil, err
		}
		log.Errorf("[UserRepository-2] UpdateUserVerified: %v", err)
		return nil, err
	}

	modelUser.IsVerified = true
	if err := u.db.Save(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-3] UpdateUserVerified: %v", err)
		return nil, err
	}

	return &entity.UserEntity{
		ID:         userID,
		Name:       modelUser.Name,
		Email:      modelUser.Email,
		RoleName:   modelUser.Roles[0].Name,
		Address:    modelUser.Address,
		Lat:        modelUser.Lat,
		Lng:        modelUser.Lng,
		Phone:      modelUser.Phone,
		Photo:      modelUser.Photo,
		IsVerified: modelUser.IsVerified,
	}, nil
}

// CreateUserAccount implements UserRepositoryInterface.
func (u *userRepository) CreateUserAccount(ctx context.Context, req entity.UserEntity) (int64, error) {
	var roleID int64

	if err := u.db.Select("id").
		Where("name = ?", "Customer").
		Model(&model.Role{}).
		Scan(&roleID).
		Error; err != nil {
		log.Errorf("[UserRepository-1] CreateUserAccount: %v", err)
		return 0, err
	}

	modelUser := model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Roles:    []model.Role{{ID: roleID}},
	}

	if err := u.db.Create(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-2] CreateUserAccount: %v", err)
		return 0, err
	}

	modelVerify := model.VerificationToken{
		UserID:    modelUser.ID,
		Token:     req.Token,
		TokenType: "email_verification",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := u.db.Create(&modelVerify).Error; err != nil {
		log.Errorf("[UserRepository-3] CreateUserAccount: %v", err)
		return 0, err
	}

	return modelUser.ID, nil
}

// GetUserByEmail implements UserRepositoryInterface.
func (u *userRepository) GetUserByEmail(ctx context.Context, email string) (*entity.UserEntity, error) {
	modelUser := model.User{}

	if err := u.db.Where("email = ? AND is_verified = ?", email, true).
		Preload("Roles").First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
			log.Infof("[UserRepository-1] GetUserByEmail: User not found")
			return nil, err
		}
		log.Errorf("[UserRepository-1] GetUserByEmail: %v", err)
		return nil, err
	}

	return &entity.UserEntity{
		ID:         modelUser.ID,
		Name:       modelUser.Name,
		Email:      email,
		Password:   modelUser.Password,
		RoleName:   modelUser.Roles[0].Name,
		Address:    modelUser.Address,
		Lat:        modelUser.Lat,
		Lng:        modelUser.Lng,
		Phone:      modelUser.Phone,
		Photo:      modelUser.Photo,
		IsVerified: modelUser.IsVerified,
	}, nil
}

func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}
