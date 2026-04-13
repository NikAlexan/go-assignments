// streaming-client demonstrates the SubscribeToOrderUpdates server-side streaming RPC.
// Usage: go run main.go <order-id>
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "github.com/nikalexan/go-proto-gen/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go <order-id>")
		os.Exit(1)
	}
	orderID := os.Args[1]

	addr := os.Getenv("ORDER_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50052"
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderServiceClient(conn)
	stream, err := client.SubscribeToOrderUpdates(context.Background(), &pb.OrderRequest{OrderId: orderID})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	fmt.Printf("Subscribed to order %s — waiting for updates...\n", orderID)
	for {
		update, err := stream.Recv()
		if err != nil {
			fmt.Printf("stream closed: %v\n", err)
			return
		}
		fmt.Printf("[UPDATE] order=%s status=%s\n", update.OrderId, update.Status)
	}
}
