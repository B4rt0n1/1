package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"payment-service/internal/broker"
	"payment-service/internal/repository"
	transport "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	pb "github.com/B4rt0n1/protoB/payment"
)

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	res, err := handler(ctx, req)
	log.Printf("Method: %s | Duration: %s | Error: %v", info.FullMethod, time.Since(start), err)
	return res, err
}

func main() {
	_ = godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPostgresPaymentRepository(db)
	publisher, err := broker.NewRabbitMQPublisher(os.Getenv("RABBITMQ_URL"), os.Getenv("RABBITMQ_QUEUE"))
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer func() {
		_ = publisher.Close()
	}()

	uc := usecase.NewPaymentUseCase(repo, publisher)
	grpcServerHandler := transport.NewPaymentGrpcServer(uc)

	port := os.Getenv("GRPC_PORT")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(loggingInterceptor))
	pb.RegisterPaymentServiceServer(grpcServer, grpcServerHandler)

	log.Printf("Payment gRPC Service running on %s", port)
	log.Fatal(grpcServer.Serve(lis))
}
