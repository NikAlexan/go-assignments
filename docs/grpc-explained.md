# Как gRPC реализован в коде проекта

---

## 1. Payment Service запускает gRPC сервер

```go
// payment-service/cmd/payment-service/main.go

// Composition Root — собираем зависимости руками
paymentRepository := repository.NewPostgresPaymentRepo(database)
paymentUseCase := usecase.NewPaymentUseCase(paymentRepository)

go func() {
    lis, _ := net.Listen("tcp", ":50051")  // открыть TCP порт

    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(transportgrpc.LoggingInterceptor), // middleware
    )

    // pb.RegisterPaymentServiceServer — сгенерирована protoc-ом
    // говорит grpcServer: "вызов ProcessPayment → наш PaymentServer"
    pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentServer(paymentUseCase))

    grpcServer.Serve(lis) // начать принимать входящие gRPC вызовы
}()

// HTTP сервер запускается в основной горутине параллельно
router.Run(":8081")
```

`grpc.NewServer` и `RegisterPaymentServiceServer` — из библиотеки `google.golang.org/grpc`.  
`pb.RegisterPaymentServiceServer` — сгенерирована `protoc` из `.proto` файла.

---

## 2. PaymentServer реализует сгенерированный интерфейс

`protoc` сгенерировал интерфейс:

```go
// go-proto-gen/payment/payment_grpc.pb.go (сгенерировано)
type PaymentServiceServer interface {
    ProcessPayment(context.Context, *PaymentRequest) (*PaymentResponse, error)
}
```

Наш код реализует его:

```go
// payment-service/internal/transport/grpc/server.go

import pb "github.com/nikalexan/go-proto-gen/payment"

type PaymentServer struct {
    pb.UnimplementedPaymentServiceServer // встроенная заглушка от protoc
    useCase *usecase.PaymentUseCase
}

func (s *PaymentServer) ProcessPayment(
    ctx context.Context,
    req *pb.PaymentRequest, // pb.PaymentRequest — сгенерированная структура
) (*pb.PaymentResponse, error) {

    // req уже готовая Go структура с полями OrderId и Amount
    // gRPC фреймворк десериализовал байты из сети в неё до вызова этого метода

    payment, err := s.useCase.Authorize(ctx, req.OrderId, req.Amount)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "authorize: %v", err)
    }

    return &pb.PaymentResponse{
        Status:        payment.Status,
        TransactionId: payment.TransactionID,
    }, nil
}
```

`UnimplementedPaymentServiceServer` — заглушка от protoc. Если добавить новый RPC
в `.proto` и не реализовать метод — вернёт `codes.Unimplemented` вместо паники компилятора.

---

## 3. Interceptor — middleware для каждого RPC вызова

```go
// payment-service/internal/transport/grpc/interceptor.go

func LoggingInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo, // info.FullMethod = "/payment.PaymentService/ProcessPayment"
    handler grpc.UnaryHandler,
) (any, error) {
    start := time.Now()

    resp, err := handler(ctx, req) // ← здесь вызывается наш ProcessPayment

    log.Printf("[gRPC] method=%s duration=%s err=%v",
        info.FullMethod, time.Since(start), err)

    return resp, err
}
```

Регистрируется один раз в `grpc.NewServer(grpc.UnaryInterceptor(...))` —
после этого оборачивает каждый входящий RPC вызов автоматически.

---

## 4. Order Service при старте открывает соединение

```go
// order-service/cmd/order-service/main.go

paymentGRPCAddr := os.Getenv("PAYMENT_GRPC_ADDR") // "payment-service:50051" из docker-compose

// NewPaymentGRPCClient открывает TCP соединение и создаёт клиент
paymentClient, _ := repository.NewPaymentGRPCClient(paymentGRPCAddr)

// Передаём клиент в UseCase через интерфейс PaymentClient
orderUseCase := usecase.NewOrderUseCase(orderRepository, paymentClient)
```

---

## 5. PaymentGRPCClient — реализация PaymentClient интерфейса

UseCase знает только об интерфейсе `PaymentClient` из `ports.go`:

```go
// order-service/internal/usecase/ports.go
type PaymentClient interface {
    Authorize(ctx context.Context, orderID string, amount int64) (string, error)
}
```

Реализация этого интерфейса через gRPC:

```go
// order-service/internal/repository/payment_grpc_client.go

import pb "github.com/nikalexan/go-proto-gen/payment"

type PaymentGRPCClient struct {
    client pb.PaymentServiceClient // сгенерированный тип
}

func NewPaymentGRPCClient(addr string) (*PaymentGRPCClient, error) {
    // grpc.NewClient открывает HTTP/2 соединение
    // insecure — без TLS (нормально внутри Docker сети)
    conn, err := grpc.NewClient(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )

    // pb.NewPaymentServiceClient — сгенерирована protoc-ом
    // создаёт клиент который умеет вызывать методы PaymentService
    return &PaymentGRPCClient{
        client: pb.NewPaymentServiceClient(conn),
    }, nil
}

// Реализуем интерфейс PaymentClient
func (c *PaymentGRPCClient) Authorize(ctx context.Context, orderID string, amount int64) (string, error) {
    // Выглядит как обычный вызов функции
    // Под капотом: gRPC фреймворк сериализует req и отправляет по сети
    resp, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
        OrderId: orderID,
        Amount:  amount,
    })
    if err != nil {
        return "", fmt.Errorf("payment grpc call: %w", err)
    }
    return resp.Status, nil
}
```

UseCase не знает что внутри gRPC — он работает только с интерфейсом `PaymentClient`.
Это Clean Architecture: поменяй реализацию на HTTP или очередь — UseCase не изменится.

---

## 6. Сгенерированный клиент внутри

`pb.NewPaymentServiceClient(conn)` возвращает объект из `payment_grpc.pb.go`:

```go
// go-proto-gen/payment/payment_grpc.pb.go (сгенерировано)

type paymentServiceClient struct {
    cc grpc.ClientConnInterface
}

func (c *paymentServiceClient) ProcessPayment(
    ctx context.Context,
    req *PaymentRequest,
) (*PaymentResponse, error) {
    out := new(PaymentResponse)
    // conn.Invoke — сериализует req, отправляет по HTTP/2, ждёт ответ, десериализует в out
    err := c.cc.Invoke(ctx, "/payment.PaymentService/ProcessPayment", req, out)
    return out, err
}
```

`Invoke` — метод gRPC библиотеки. Именно здесь происходит сетевой вызов.
До и после `Invoke` твой код работает с обычными Go структурами.

---

## 7. Как UseCase вызывает клиент

```go
// order-service/internal/usecase/order_usecase.go

func (useCase *OrderUseCase) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
    // ... создаём заказ, сохраняем в БД ...

    // Таймаут 2 секунды — если Payment не ответил, заказ → Failed
    paymentCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    // paymentClient.Authorize — наш PaymentGRPCClient.Authorize из шага 5
    // UseCase не знает что внутри gRPC
    paymentStatus, err := useCase.paymentClient.Authorize(paymentCtx, order.ID, order.Amount)
    if err != nil {
        useCase.repository.UpdateStatus(ctx, order.ID, "Failed")
        return order, ErrPaymentUnavailable
    }

    newStatus := "Failed"
    if paymentStatus == "Authorized" {
        newStatus = "Paid"
    }
    useCase.repository.UpdateStatus(ctx, order.ID, newStatus)
    return order, nil
}
```

---

## Цепочка вызовов при POST /orders

```
HTTP handler (gin)
  → orderUseCase.CreateOrder()
    → paymentClient.Authorize()               ← интерфейс PaymentClient
      → PaymentGRPCClient.Authorize()         ← наша реализация
        → pb.client.ProcessPayment()          ← сгенерированный клиент
          → conn.Invoke()                     ← gRPC библиотека, сеть
            → LoggingInterceptor              ← middleware на стороне Payment
              → PaymentServer.ProcessPayment() ← наш обработчик
                → paymentUseCase.Authorize()  ← бизнес-логика
                  → postgresRepo.Save()       ← БД
```

---

## Где какой код живёт

```
go-proto-gen/payment/
├── payment.pb.go        — структуры PaymentRequest, PaymentResponse (сгенерировано)
└── payment_grpc.pb.go   — PaymentServiceClient, PaymentServiceServer,
                           NewPaymentServiceClient, RegisterPaymentServiceServer (сгенерировано)

payment-service/
├── cmd/.../main.go                  — запуск grpc.NewServer + RegisterPaymentServiceServer
└── internal/transport/grpc/
    ├── server.go                    — реализация ProcessPayment (наш код)
    └── interceptor.go               — LoggingInterceptor (наш код)

order-service/
├── cmd/.../main.go                  — NewPaymentGRPCClient(addr)
├── internal/usecase/ports.go        — интерфейс PaymentClient
└── internal/repository/
    └── payment_grpc_client.go       — реализация через pb.NewPaymentServiceClient (наш код)
```