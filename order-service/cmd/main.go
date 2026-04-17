package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/repository"
	"order-service/internal/transport/grpc/client"
	"order-service/internal/transport/grpc/server"
	ginhttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	pb "https://github.com/B4rt0n1/protoB.git/order"
)

func main() {
	_ = godotenv.Load()

	// 1. Database Connection
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// 2. Dial Payment gRPC Server
	paymentAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	conn, err := grpc.Dial(paymentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 3. Composition Root / DI
	repo := repository.NewPostgresOrderRepository(db)
	paymentClient := client.NewPaymentGrpcClient(conn)
	uc := usecase.NewOrderUseCase(repo, paymentClient)

	// 4. Start gRPC Server for Order Streaming Updates
	go func() {
		lis, err := net.Listen("tcp", os.Getenv("ORDER_GRPC_PORT"))
		if err != nil {
			log.Fatal(err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterOrderTrackingServiceServer(grpcServer, server.NewOrderTrackingServer(uc))
		log.Printf("Order Streaming gRPC Server running on %s", os.Getenv("ORDER_GRPC_PORT"))
		grpcServer.Serve(lis)
	}()

	// 5. Start External REST API (Gin)
	r := gin.Default()
	r.Use(ginhttp.CORSMiddleware())

	restHandler := ginhttp.NewGinOrderHandler(uc)
	r.POST("/orders", restHandler.CreateOrder)
	r.GET("/orders/:id", restHandler.GetOrder)
	r.PATCH("/orders/:id/cancel", restHandler.CancelOrder)

	log.Printf("Order REST API running on %s", os.Getenv("REST_PORT"))
	r.Run(os.Getenv("REST_PORT"))
}
