package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/core/domain/entity"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/gommon/log"
)

type CartRedisRepositoryInterface interface {
	AddToCart(ctx context.Context, userID string, items []entity.CartItem) error
	GetCart(ctx context.Context, userID string) ([]entity.CartItem, error)
	RemoveFromCart(ctx context.Context, userID int64, productID int64) error
	RemoveAllCart(ctx context.Context, userID int64) error
}

type CartRedisRepository struct {
	Client *redis.Client
}

// RemoveAllCart implements CartRedisRepositoryInterface.
func (c *CartRedisRepository) RemoveAllCart(ctx context.Context, userID int64) error {
	return c.Client.Del(ctx, fmt.Sprintf("cart:cart:%d", userID)).Err()
}

// AddToCart implements CartRedisRepositoryInterface.
func (c *CartRedisRepository) AddToCart(ctx context.Context, userID string, items []entity.CartItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		log.Errorf("[CartRedisRepository-1] AddToCart: %v", err)
		return err
	}
	return c.Client.Set(ctx, fmt.Sprintf("cart:%s", userID), data, 0).Err()
}

// GetCart implements CartRedisRepositoryInterface.
func (c *CartRedisRepository) GetCart(ctx context.Context, userID string) ([]entity.CartItem, error) {
	val, err := c.Client.Get(ctx, fmt.Sprintf("cart:%s", userID)).Result()
	if err == redis.Nil {
		log.Infof("[CartRedisRepository-1] GetCart: Cart not found")
		return nil, nil
	}
	if err != nil {
		log.Errorf("[CartRedisRepository-2] GetCart: %v", err)
		return nil, err
	}
	var items []entity.CartItem
	err = json.Unmarshal([]byte(val), &items)
	if err != nil {
		log.Errorf("[CartRedisRepository-3] GetCart: %v", err)
		return nil, err
	}
	return items, nil
}

// RemoveFromCart implements CartRedisRepositoryInterface.
func (c *CartRedisRepository) RemoveFromCart(ctx context.Context, userID int64, productID int64) error {
	cart, err := c.GetCart(ctx, fmt.Sprintf("cart:%d", userID))
	if err != nil {
		log.Errorf("[CartRedisRepository-1] RemoveFromCart: %v", err)
		return err
	}

	newCart := []entity.CartItem{}
	for _, item := range cart {
		if item.ProductID != productID {
			newCart = append(newCart, item)
		}
	}

	err = c.Client.Del(ctx, fmt.Sprintf("cart:cart:%d", userID)).Err()
	if err != nil {
		log.Errorf("[CartRedisRepository-2] RemoveFromCart: %v", err)
		return err
	}

	return c.AddToCart(ctx, fmt.Sprintf("cart:%d", userID), newCart)
}

func NewCartRedisRepository(client *redis.Client) CartRedisRepositoryInterface {
	return &CartRedisRepository{
		Client: client,
	}
}
