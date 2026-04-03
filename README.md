# Kafka Event Pipeline (Go)

This project implements a robust event-driven pipeline using Go, Kafka, MS SQL, and Redis.

## Architecture

1.  **Producer**: Publishes 4 JSON event types (`UserCreated`, `OrderPlaced`, `PaymentSettled`, `InventoryAdjusted`) to a Kafka topic.
2.  **Consumer**: A Go consumer group that routes messages by type, performs idempotent upserts into MS SQL Server, and pushes failures to a Redis Dead Letter Queue (DLQ).
3.  **Read & Produce API**: A REST API to fetch User and Order information, and manually produce events.
4.  **Infrastructure**: Managed via `docker-compose.yml` (Kafka, MS SQL, Redis).

## Event Types

| Event Type | Key | Description |
| :--- | :--- | :--- |
| `UserCreated` | `userId` | New user registered. |
| `OrderPlaced` | `orderId` | New order created by a user. |
| `PaymentSettled` | `orderId` | Payment received for an order. |
| `InventoryAdjusted` | `sku` | Inventory level changes. |

## Implementation Plan

1.  **Project Setup**: Initialize Go module and folder structure.
2.  **Infrastructure**: Define `docker-compose.yml` with Kafka, SQL Server, and Redis.
3.  **Schema**: Create SQL scripts for `Users`, `Orders`, `Payments`, and `Inventory`.
4.  **Kafka Layer**: Implement Producer and Consumer logic using a Go Kafka library (e.g., `segmentio/kafka-go`).
5.  **Persistence Layer**: Implement MS SQL upsert logic with idempotency.
6.  **DLQ Layer**: Implement Redis client to store failed messages + error metadata.
7.  **API Layer**: Implement `GET /users/{id}` and `GET /orders/{id}` using standard library or a router (e.g., `chi`).
8.  **Metrics & Logging**: Implement basic stats (msg/sec, p95 latency) and structured logging with `eventId` correlation.
9.  **Verification**: Write sample HTTP requests and documentation.

## Running the Project

1.  **Prerequisites**: Docker and Docker Compose installed.
2.  **Start Infrastructure & Services**:
    ```bash
    docker-compose up --build
    ```
    This will:
    - Start Kafka, MS SQL, and Redis.
    - Run the `setup-db` container to initialize the schema in SQL Server.
    - Start the `api`, `consumer`, and `producer` services.

3.  **Produce Sample Data**:
    If you want to produce more data, you can restart the producer container:
    ```bash
    docker-compose restart producer
    ```

4.  **Test the API**:
    - `POST http://localhost:8080/events`: Manually trigger a new Kafka event (JSON body).
    - `GET http://localhost:8080/users/user-123`: Fetch user and recent orders.
    - `GET http://localhost:8080/orders/order-456`: Fetch order and payment info.
    - `GET http://localhost:8080/dlq`: View failed events stored in Redis.
    - Use the `api-tests.http` file if you have the REST Client extension.

5.  **Check DLQ**:
    Execute inside the redis container:
    ```bash
    docker exec -it <redis-container-id> redis-cli LRANGE event_pipeline_dlq 0 -1
    ```

## Key Design Decisions

- **Idempotency**: Implemented using SQL Server `IF EXISTS ... UPDATE ELSE INSERT`.
- **Failure Handling**: Any parsing or DB error in the consumer triggers a push to Redis `event_pipeline_dlq` list.
- **Correlation**: All logs in the consumer are prefixed with the `eventId` extracted from the message payload.
- **Metrics**: Throughput (msg/sec) and DLQ size are logged every 10 seconds. Individual message latency is logged per event.

---
*Self-Correction*: The user asked for a "Plan of execution in this repo". I should ensure I stick to the 60-minute timebox mentioned in the prompt. I'll work quickly!
