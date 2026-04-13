package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor logs method name and duration for every incoming unary RPC.
func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("[gRPC] method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
	return resp, err
}
