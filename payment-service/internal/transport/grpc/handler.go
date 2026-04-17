package grpc

import (
	"context"
	"payment-service/internal/usecase"

	pb "github.com/B4rt0n1/protoB/payment"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentGrpcServer struct {
	pb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUseCase
}

func NewPaymentGrpcServer(uc *usecase.PaymentUseCase) *PaymentGrpcServer {
	return &PaymentGrpcServer{uc: uc}
}

func (s *PaymentGrpcServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	payment, err := s.uc.ProcessPayment(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	return &pb.PaymentResponse{
		Status:        payment.Status,
		TransactionId: payment.TransactionID,
	}, nil
}

func (s *PaymentGrpcServer) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	min := req.GetMinAmount()
	max := req.GetMaxAmount()

	domainPayments, err := s.uc.GetPaymentsByRange(ctx, min, max)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var pbPayments []*pb.PaymentResponse
	for _, p := range domainPayments {
		pbPayments = append(pbPayments, &pb.PaymentResponse{
			Status:        p.Status,
			TransactionId: p.TransactionID,
		})
	}

	return &pb.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
