package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abhisurya1c/go-kafka-event-pipeline/internal/infra"
	"github.com/abhisurya1c/go-kafka-event-pipeline/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

func main() {
	// Env variables
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getEnv("KAFKA_TOPIC", "events")
	groupID := getEnv("KAFKA_GROUP", "event-consumer-group")
	dbConn := getEnv("MSSQL_CONN", "sqlserver://sa:Password123!@localhost:1433?database=EventsDB&encrypt=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")

	// Init infra
	db, err := infra.NewDB(dbConn)
	if err != nil {
		log.Fatalf("Failed to connect to MS SQL: %v", err)
	}
	defer db.Close()

	rClient := infra.NewRedis(redisAddr)
	defer rClient.Close()

	reader := infra.NewKafkaReader(brokers, topic, groupID)
	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Consumer started on topic %s...", topic)

	msgCount := 0
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			dlqCount, _ := rClient.LLen(ctx, "event_pipeline_dlq").Result()
			log.Printf("[METRICS] Processed %d messages in last 10s (approx %.2f msg/sec). DLQ size: %d", msgCount, float64(msgCount)/10.0, dlqCount)
			msgCount = 0
		}
	}()

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error reading message: %v", err)
			continue
		}

		msgCount++
		start := time.Now()
		processMessage(ctx, db, rClient, m)
		latency := time.Since(start)

		// Basic metrics in log
		log.Printf("[METRICS][LATENCY] Event processing took %v", latency)
	}
}

func processMessage(ctx context.Context, db *sql.DB, redisClient *redis.Client, m kafka.Message) {
	// 1. Identify eventId (correlation) from payload if possible
	var header models.EventHeader
	if err := json.Unmarshal(m.Value, &header); err != nil {
		log.Printf("Failed to parse header: %v. Raw: %s", err, string(m.Value))
		pushToDLQ(ctx, redisClient, m.Value, fmt.Sprintf("Header parsing failed: %v", err))
		return
	}

	if header.EventID == "" {
		header.EventID = uuid.NewString() // Fallback
	}

	l := log.New(os.Stdout, fmt.Sprintf("[%s] ", header.EventID), log.LstdFlags)

	var err error
	switch header.Type {
	case models.TypeUserCreated:
		var event models.UserCreated
		json.Unmarshal(m.Value, &event)
		err = upsertUser(ctx, db, event)
	case models.TypeOrderPlaced:
		var event models.OrderPlaced
		json.Unmarshal(m.Value, &event)
		err = upsertOrder(ctx, db, event)
	case models.TypePaymentSettled:
		var event models.PaymentSettled
		json.Unmarshal(m.Value, &event)
		err = upsertPayment(ctx, db, event)
	case models.TypeInventoryAdjusted:
		var event models.InventoryAdjusted
		json.Unmarshal(m.Value, &event)
		err = upsertInventory(ctx, db, event)
	default:
		err = fmt.Errorf("unknown event type: %s", header.Type)
	}

	if err != nil {
		l.Printf("Error processing event: %v", err)
		pushToDLQ(ctx, redisClient, m.Value, err.Error())
	} else {
		l.Printf("Successfully processed %s", header.Type)
	}
}

func upsertUser(ctx context.Context, db *sql.DB, u models.UserCreated) error {
	query := `
    IF EXISTS (SELECT 1 FROM Users WHERE Id = @Id)
        UPDATE Users SET Name = @Name, Email = @Email WHERE Id = @Id
    ELSE
        INSERT INTO Users (Id, Name, Email) VALUES (@Id, @Name, @Email)`
	_, err := db.ExecContext(ctx, query, sql.Named("Id", u.UserID), sql.Named("Name", u.Name), sql.Named("Email", u.Email))
	return err
}

func upsertOrder(ctx context.Context, db *sql.DB, o models.OrderPlaced) error {
	query := `
    IF EXISTS (SELECT 1 FROM Orders WHERE Id = @Id)
        UPDATE Orders SET UserId = @UserId, Status = @Status, Amount = @Amount WHERE Id = @Id
    ELSE
        INSERT INTO Orders (Id, UserId, Status, Amount) VALUES (@Id, @UserId, @Status, @Amount)`
	_, err := db.ExecContext(ctx, query, sql.Named("Id", o.OrderID), sql.Named("UserId", o.UserID), sql.Named("Status", o.Status), sql.Named("Amount", o.Amount))
	return err
}

func upsertPayment(ctx context.Context, db *sql.DB, p models.PaymentSettled) error {
	query := `
    IF EXISTS (SELECT 1 FROM Payments WHERE OrderId = @OrderId)
        UPDATE Payments SET Status = @Status, Amount = @Amount WHERE OrderId = @OrderId
    ELSE
        INSERT INTO Payments (OrderId, Status, Amount) VALUES (@OrderId, @Status, @Amount)`
	_, err := db.ExecContext(ctx, query, sql.Named("OrderId", p.OrderID), sql.Named("Status", p.Status), sql.Named("Amount", p.Amount))
	return err
}

func upsertInventory(ctx context.Context, db *sql.DB, i models.InventoryAdjusted) error {
	query := `
    IF EXISTS (SELECT 1 FROM Inventory WHERE SKU = @SKU)
        UPDATE Inventory SET Quantity = @Quantity, LastUpdated = SYSDATETIMEOFFSET() WHERE SKU = @SKU
    ELSE
        INSERT INTO Inventory (SKU, Quantity) VALUES (@SKU, @Quantity)`
	_, err := db.ExecContext(ctx, query, sql.Named("SKU", i.SKU), sql.Named("Quantity", i.Quantity))
	return err
}

func pushToDLQ(ctx context.Context, r *redis.Client, payload []byte, errMsg string) {
	dlq := models.DLQPayload{
		OriginalPayload: string(payload),
		Error:           errMsg,
		Timestamp:       time.Now(),
	}
	data, _ := json.Marshal(dlq)
	r.LPush(ctx, "event_pipeline_dlq", data)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
