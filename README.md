# AP2 Assignment 1 - Clean Architecture Microservices (Order & Payment)

**Student:** Nikita Vassilenko, SE-2410

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          Client (curl / Postman)                │
└───────────────────────────────┬─────────────────────────────────┘
                                │ HTTP REST
                                |
┌───────────────────────────────────────────────────┐
│              Order Service  :8080                 │
│                                                   │
│  Transport (Gin handlers)                         │
│       │                                           │
│  Use Case (business logic, state transitions)     │
│       │                          │                │
│  OrderRepository            PaymentClient         │
│  (interface)                (interface)           │
│       │                          │                │
│  PostgresOrderRepo     PaymentHTTPClient          │
└───────┬──────────────────────────┬────────────────┘
        │                          │ HTTP POST /payments
        |                          |
  ┌──────────┐        ┌────────────────────────────────┐
  │ orders_db│        │     Payment Service  :8081     │
  │(Postgres)│        │                                │
  └──────────┘        │  Transport (Gin handlers)      │
                      │       │                        │
                      │  Use Case (auth + limit check) │
                      │       │                        │
                      │  PaymentRepository (interface) │
                      │       │                        │
                      │  PostgresPaymentRepo           │
                      └───────┬────────────────────────┘
                              │
                        ┌─────────────┐
                        │ payments_db │
                        │  (Postgres) │
                        └─────────────┘
```

---

## Bounded Contexts

| Context         | Owns                              | Database     |
|-----------------|-----------------------------------|--------------|
| Order Service   | Orders, their state, idempotency  | `orders_db`  |
| Payment Service | Payments, transaction IDs, limits | `payments_db`|

Each service has its own internal domain model. There is **no shared code** or common package between services.

---

## Clean Architecture Layers (per service)

```
internal/
├── domain/       <- Pure Go structs + domain errors. No framework deps.
├── usecase/      <- Business logic. Depends on interfaces (ports) only.
├── repository/   <- Implements ports. Talks to PostgreSQL / HTTP.
└── transport/http/<- Thin Gin handlers. Parses requests, calls use cases.
cmd/<service>/main.go <- Composition Root. Manual DI, no magic.
```

---

## Failure Handling

**Scenario:** Payment Service is unavailable (down / timeout).

**Decision:** Order is marked **`Failed`** (not left as `Pending`).

**Rationale:**  

- `Pending` is ambiguous – the client can't tell if the order is still being processed or stuck.  
- `Failed` is explicit: the client knows to retry (using a new `Idempotency-Key`).  
- The Order Service returns **HTTP 503 Service Unavailable** to the caller, along with the failed order data.  
- The `http.Client` in Order Service has a **2-second timeout**, so it never hangs.

---

## Business Rules

| Rule          | Detail                                                                |
|---------------|-----------------------------------------------------------------------|
| Amount type   | `int64` (cents). Float64 is forbidden for money.                      |
| Amount > 0    | Validated in Order use case.                                          |
| Payment limit | Amount > 100 000 cents ($1000) -> Payment Service returns `Declined`. |
| Cancel rule   | Only `Pending` orders can be cancelled. `Paid` orders return 409.     |
| Timeout       | HTTP client for inter-service calls has a 2-second timeout.           |

---

## Idempotency (Bonus)

Send `Idempotency-Key: <uuid>` header with `POST /orders`.  
If the same key is used again, the existing order is returned without creating a duplicate.  
The key is stored in `orders.idempotency_key` (UNIQUE constraint).

---

## How to Run

**Prerequisites:** Docker & Docker Compose.

```bash
docker compose up --build
```

Services start on:

- Order Service -> `http://localhost:8080`
- Payment Service -> `http://localhost:8081`

Databases are automatically migrated on startup.

---

## API Examples

### POST /orders - create order (will be Paid)

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: key-001" \
  -d '{"customer_id":"cust-1","item_name":"Laptop","amount":50000}'
```

### POST /orders - create order (will be Failed, amount > $1000)

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-1","item_name":"Server","amount":150000}'
```

### GET /orders/:id - get order

```bash
curl http://localhost:8080/orders/<order-id>
```

### PATCH /orders/:id/cancel - cancel order

```bash
curl -X PATCH http://localhost:8080/orders/<order-id>/cancel
```

### GET /payments/:order_id - get payment by order

```bash
curl http://localhost:8081/payments/<order-id>
```
