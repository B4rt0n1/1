package usecase

import (
	"context"
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	FindByAmountRange(ctx context.Context, min, max int64) ([]*domain.Payment, error)
}

type PaymentUseCase struct {
	repo PaymentRepository
}

func NewPaymentUseCase(r PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: r}
}

func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		Amount:        amount,
		TransactionID: uuid.NewString(),
	}

	if amount > 100000 {
		payment.Status = "Declined"
	} else {
		payment.Status = "Authorized"
	}

	if err := uc.repo.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (uc *PaymentUseCase) GetPayment(ctx context.Context, orderID string) (*domain.Payment, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}

func (uc *PaymentUseCase) GetPaymentsByRange(ctx context.Context, min, max int64) ([]*domain.Payment, error) {
	if min > 0 && max > 0 && min > max {
		return nil, errors.New("min_amount cannot be greater than max_amount")
	}

	return uc.repo.FindByAmountRange(ctx, min, max)
}
