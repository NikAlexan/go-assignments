package grpc

import (
	"context"

	pb "github.com/nikalexan/go-proto-gen/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/usecase"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	useCase *usecase.PaymentUseCase
}

func NewPaymentServer(uc *usecase.PaymentUseCase) *PaymentServer {
	return &PaymentServer{useCase: uc}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	payment, err := s.useCase.Authorize(ctx, req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "authorize payment: %v", err)
	}

	return &pb.PaymentResponse{
		Status:        payment.Status,
		TransactionId: payment.TransactionID,
	}, nil
}
