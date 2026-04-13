package grpc

import (
	"time"

	pb "github.com/nikalexan/go-proto-gen/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/usecase"
)

// terminalStatuses are order states that will never change again.
var terminalStatuses = map[string]bool{
	"Paid":      true,
	"Failed":    true,
	"Cancelled": true,
}

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	orderUseCase *usecase.OrderUseCase
}

func NewOrderServer(uc *usecase.OrderUseCase) *OrderServer {
	return &OrderServer{orderUseCase: uc}
}

// SubscribeToOrderUpdates streams order status changes to the client.
// It polls the database every second and sends an update whenever the status changes.
// The stream is closed when the order reaches a terminal state.
func (s *OrderServer) SubscribeToOrderUpdates(req *pb.OrderRequest, stream pb.OrderService_SubscribeToOrderUpdatesServer) error {
	if req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	ctx := stream.Context()
	var lastStatus string

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			order, err := s.orderUseCase.GetOrder(ctx, req.OrderId)
			if err != nil {
				return status.Errorf(codes.NotFound, "order not found: %v", err)
			}

			if order.Status != lastStatus {
				lastStatus = order.Status
				if err := stream.Send(&pb.OrderStatusUpdate{
					OrderId:   order.ID,
					Status:    order.Status,
					UpdatedAt: timestamppb.New(order.CreatedAt),
				}); err != nil {
					return err
				}
			}

			if terminalStatuses[order.Status] {
				return nil
			}
		}
	}
}
