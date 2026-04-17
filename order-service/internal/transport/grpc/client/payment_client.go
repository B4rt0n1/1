package client

import (
	"context"
	"time"

	pb "github.com/B4rt0n1/protoB/payment"

	"google.golang.org/grpc"
)

type PaymentGrpcClient struct {
	client pb.PaymentServiceClient
}

func NewPaymentGrpcClient(conn *grpc.ClientConn) *PaymentGrpcClient {
	return &PaymentGrpcClient{
		client: pb.NewPaymentServiceClient(conn),
	}
}

func (c *PaymentGrpcClient) ProcessPayment(ctx context.Context, orderID string, amount int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	res, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return "", err
	}
	return res.Status, nil
}
