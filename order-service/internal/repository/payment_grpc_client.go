package repository

import (
	"context"
	"fmt"

	pb "github.com/nikalexan/go-proto-gen/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/usecase"
)

type PaymentGRPCClient struct {
	client pb.PaymentServiceClient
}

func NewPaymentGRPCClient(addr string) (*PaymentGRPCClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial payment service: %w", err)
	}
	return &PaymentGRPCClient{client: pb.NewPaymentServiceClient(conn)}, nil
}

func (c *PaymentGRPCClient) Authorize(ctx context.Context, orderID string, amount int64) (string, error) {
	resp, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return "", fmt.Errorf("payment grpc call: %w", err)
	}
	return resp.Status, nil
}

func (c *PaymentGRPCClient) GetPaymentStats(ctx context.Context) (*usecase.PaymentStats, error) {
	resp, err := c.client.GetPaymentStats(ctx, &pb.GetPaymentStatsRequest{})
	if err != nil {
		return nil, fmt.Errorf("get payment stats grpc call: %w", err)
	}
	return &usecase.PaymentStats{
		TotalCount:      resp.TotalCount,
		AuthorizedCount: resp.AuthorizedCount,
		DeclinedCount:   resp.DeclinedCount,
		TotalAmount:     resp.TotalAmount,
	}, nil
}
