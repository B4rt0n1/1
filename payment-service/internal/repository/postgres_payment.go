package repository

import (
	"context"
	"database/sql"
	"payment-service/internal/domain"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Save(ctx context.Context, payment *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, transaction_id, amount, status) 
	          VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status)
	return err
}

func (r *PostgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p := &domain.Payment{}
	query := `SELECT id, order_id, transaction_id, amount, status FROM payments WHERE order_id = $1`
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status)
	return p, err
}

func (r *PostgresPaymentRepository) FindByAmountRange(ctx context.Context, min, max int64) ([]*domain.Payment, error) {
	var payments []*domain.Payment

	query := `
		SELECT id, order_id, transaction_id, amount, status 
		FROM payments 
		WHERE (amount >= $1 OR $1 = 0) 
		  AND (amount <= $2 OR $2 = 0)
	`

	rows, err := r.db.QueryContext(ctx, query, min, max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p := &domain.Payment{}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}
