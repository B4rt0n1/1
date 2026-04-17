package usecase

import (
	"context"
	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	UpdateStatus(ctx context.Context, id string, status string) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	GetByUserID(ctx context.Context, customerID string) ([]*domain.Order, error)
}

type PaymentGateway interface {
	ProcessPayment(ctx context.Context, orderID string, amount int64) (string, error)
}
