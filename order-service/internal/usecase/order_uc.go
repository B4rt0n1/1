package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount             = errors.New("amount must be greater than 0")
	ErrCannotCancel              = errors.New("cannot cancel an order that is already paid")
	ErrPaymentServiceUnavailable = errors.New("payment service unavailable")
)

type OrderUseCase struct {
	repo    OrderRepository
	payment PaymentGateway
}

func NewOrderUseCase(r OrderRepository, p PaymentGateway) *OrderUseCase {
	return &OrderUseCase{repo: r, payment: p}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     "Pending",
		CreatedAt:  time.Now(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	status, err := uc.payment.ProcessPayment(ctx, order.ID, order.Amount)

	if err != nil {
		uc.repo.UpdateStatus(ctx, order.ID, "Failed")
		return nil, ErrPaymentServiceUnavailable
	}

	if status == "Authorized" {
		order.Status = "Paid"
	} else {
		order.Status = "Failed"
	}

	if err := uc.repo.UpdateStatus(ctx, order.ID, order.Status); err != nil {
		return nil, err
	}

	return order, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) error {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if order.Status == "Paid" {
		return ErrCannotCancel
	}
	return uc.repo.UpdateStatus(ctx, id, "Cancelled")
}

func (uc *OrderUseCase) ListOrdersByUser(ctx context.Context, customerID string) (*domain.OrdersList, error) {
	orders, err := uc.repo.GetByUserID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return &domain.OrdersList{Orders: orders}, nil
}
