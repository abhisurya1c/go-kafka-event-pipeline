package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/abhisurya1c/go-kafka-event-pipeline/internal/infra"
	"github.com/abhisurya1c/go-kafka-event-pipeline/internal/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := getEnv("KAFKA_TOPIC", "events")

	writer := infra.NewKafkaWriter(brokers, topic)
	defer writer.Close()

	fmt.Printf("Producer started. Producing sample events to topic '%s'...\n", topic)

	// Simulating constant stream of different events
	userID := "user-123"
	orderID := "order-456"
	sku := "SKU-999"

	for i := 0; i < 5; i++ {
		// 1. User Created
		sendEvent(writer, models.TypeUserCreated, userID, models.UserCreated{
			EventHeader: models.EventHeader{EventID: uuid.NewString(), Timestamp: time.Now(), Type: models.TypeUserCreated},
			UserID:      userID,
			Name:        "John Doe",
			Email:       "john@test.com",
		})

		// 2. Order Placed
		sendEvent(writer, models.TypeOrderPlaced, orderID, models.OrderPlaced{
			EventHeader: models.EventHeader{EventID: uuid.NewString(), Timestamp: time.Now(), Type: models.TypeOrderPlaced},
			OrderID:     orderID,
			UserID:      userID,
			Amount:      100.50,
			Status:      "PENDING",
		})

		// 3. Payment Settled
		sendEvent(writer, models.TypePaymentSettled, orderID, models.PaymentSettled{
			EventHeader: models.EventHeader{EventID: uuid.NewString(), Timestamp: time.Now(), Type: models.TypePaymentSettled},
			OrderID:     orderID,
			Amount:      100.50,
			Status:      "SUCCESS",
		})

		// 4. Inventory Adjusted
		sendEvent(writer, models.TypeInventoryAdjusted, sku, models.InventoryAdjusted{
			EventHeader: models.EventHeader{EventID: uuid.NewString(), Timestamp: time.Now(), Type: models.TypeInventoryAdjusted},
			SKU:         sku,
			Quantity:    rand.Intn(100),
		})

		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	}

	// Send an invalid JSON to test DLQ
	writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("bad-event"),
		Value: []byte("{invalid-json-payload}"),
	})
	fmt.Println("Sent invalid JSON to test DLQ.")
}

func sendEvent(writer *kafka.Writer, eventType models.EventType, key string, event interface{}) {
	payload, _ := json.Marshal(event)
	err := writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(key),
		Value: payload,
	})
	if err != nil {
		log.Printf("Failed to write message: %v", err)
	} else {
		fmt.Printf("Produced %s (Key: %s)\n", eventType, key)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
