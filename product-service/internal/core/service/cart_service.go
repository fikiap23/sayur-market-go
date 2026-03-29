package service

import (
	"context"
	"fmt"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

type CartServiceInterface interface {
	AddToCart(ctx context.Context, userID int64, req entity.CartItem) error
	GetCartByUserID(ctx context.Context, userID int64) ([]entity.CartItem, error)
	RemoveFromCart(ctx context.Context, userID int64, productID int64) error
	RemoveAllCart(ctx context.Context, userID int64) error
}

type cartService struct {
	cartRepository repository.CartRedisRepositoryInterface
}

// RemoveAllCart implements CartServiceInterface.
func (c *cartService) RemoveAllCart(ctx context.Context, userID int64) error {
	return c.cartRepository.RemoveAllCart(ctx, userID)
}

// RemoveFromCart implements CartServiceInterface.
func (c *cartService) RemoveFromCart(ctx context.Context, userID int64, productID int64) error {
	return c.cartRepository.RemoveFromCart(ctx, userID, productID)
}

// AddToCart implements CartServiceInterface.
func (c *cartService) AddToCart(ctx context.Context, userID int64, req entity.CartItem) error {
	cart, err := c.cartRepository.GetCart(ctx, fmt.Sprintf("cart:%d", userID))
	if err != nil {
		log.Errorf("[CartService-1] AddToCart: %v", err)
		return err
	}

	found := false
	for i, item := range cart {
		if item.ProductID == req.ProductID {
			cart[i].Quantity += req.Quantity
			found = true
			break
		}
	}

	if !found {
		cart = append(cart, req)
	}

	return c.cartRepository.AddToCart(ctx, fmt.Sprintf("cart:%d", userID), cart)
}

// GetCartByUserID implements CartServiceInterface.
func (c *cartService) GetCartByUserID(ctx context.Context, userID int64) ([]entity.CartItem, error) {
	cart, err := c.cartRepository.GetCart(ctx, fmt.Sprintf("cart:%d", userID))
	if err != nil {
		log.Errorf("[CartService-1] GetCartByUserID: %v", err)
		return nil, err
	}

	return cart, nil
}

func NewCartService(cartRepository repository.CartRedisRepositoryInterface) CartServiceInterface {
	return &cartService{
		cartRepository: cartRepository,
	}
}
