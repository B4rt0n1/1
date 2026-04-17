# Assignment 2: gRPC Migration & Contract-First Microservices


* **Repository A (Source Protos):** [https://github.com/B4rt0n1/protoA](https://github.com/B4rt0n1/protoA)
* **Repository B (Generated Code):** [https://github.com/B4rt0n1/protoB](https://github.com/B4rt0n1/protoB)

---

## Architecture Diagram

```mermaid
graph TD
    subgraph "External/Client"
        Client[Web/Mobile Client]
        gRPC_Client[gRPC Streaming Client]
    end

    subgraph "Order Service (Port: 8080 & 50051)"
        O_REST[Gin REST Handler]
        O_UC[Order UseCase]
        O_gRPC_Client[Payment gRPC Client]
        O_gRPC_Server[Order Streaming Server]
        
        O_REST --> O_UC
        O_gRPC_Server --> O_UC
        O_UC --> O_Repo[(Order DB - Port 54321)]
        O_UC --> O_gRPC_Client
    end

    subgraph "Payment Service (Port: 50052)"
        P_Interceptor(Logging Interceptor)
        P_gRPC_Server[Payment gRPC Server]
        P_UC[Payment UseCase]
        
        P_Interceptor --> P_gRPC_Server
        P_gRPC_Server --> P_UC
        P_UC --> P_Repo[(Payment DB - Port 5433)]
    end

    %% Communication Lines
    Client -- "POST /orders (REST)" --> O_REST
    O_gRPC_Client -- "gRPC Unary (ProcessPayment)" --> P_Interceptor
    gRPC_Client -. "gRPC Server-Side Stream (SubscribeToOrderUpdates)" .-> O_gRPC_Server