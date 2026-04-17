package server

import (
	"time"

	"order-service/internal/usecase"

	pb "github.com/B4rt0n1/protoB/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderTrackingServer struct {
	pb.UnimplementedOrderTrackingServiceServer
	uc *usecase.OrderUseCase
}

func NewOrderTrackingServer(uc *usecase.OrderUseCase) *OrderTrackingServer {
	return &OrderTrackingServer{uc: uc}
}

func (s *OrderTrackingServer) SubscribeToOrderUpdates(req *pb.OrderRequest, stream pb.OrderTrackingService_SubscribeToOrderUpdatesServer) error {
	orderID := req.GetOrderId()
	var lastStatus string

	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			order, err := s.uc.GetOrder(stream.Context(), orderID)
			if err != nil {
				return status.Errorf(codes.NotFound, "order not found")
			}

			if order.Status != lastStatus {
				lastStatus = order.Status
				err := stream.Send(&pb.OrderStatusUpdate{
					OrderId:   order.ID,
					Status:    order.Status,
					UpdatedAt: timestamppb.Now(),
				})
				if err != nil {
					return err
				}
			}

			if lastStatus == "Paid" || lastStatus == "Failed" || lastStatus == "Cancelled" {
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}
