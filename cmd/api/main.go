package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/abhisurya1c/go-kafka-event-pipeline/internal/infra"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type UserWithOrders struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	CreatedAt time.Time   `json:"created_at"`
	Orders    []OrderInfo `json:"last_5_orders"`
}

type OrderInfo struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderWithPayment struct {
	ID            string       `json:"id"`
	Amount        float64      `json:"amount"`
	Status        string       `json:"status"`
	PaymentStatus string       `json:"payment_status"`
	SettledAt     *time.Time   `json:"settled_at,omitempty"`
}

var db *sql.DB
var redisClient *redis.Client
var kafkaWriter *kafka.Writer

func main() {
	dbConn := getEnv("MSSQL_CONN", "sqlserver://sa:Password123!@localhost:1433?database=EventsDB&encrypt=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC", "events")
	port := getEnv("PORT", "8080")

	var err error
	db, err = infra.NewDB(dbConn)
	if err != nil {
		log.Fatalf("Failed to connect to MS SQL: %v", err)
	}
	defer db.Close()

	redisClient = infra.NewRedis(redisAddr)
	defer redisClient.Close()

	kafkaWriter = infra.NewKafkaWriter(kafkaBrokers, kafkaTopic)
	defer kafkaWriter.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/users/{id}", getUserOrders)
	r.Get("/orders/{id}", getOrderPayment)
	r.Get("/dlq", getDLQ)
	r.Post("/events", produceEvent)

	log.Printf("Read API listening on port %s...", port)
	http.ListenAndServe(":"+port, r)
}

func getUserOrders(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "id")
	ctx := r.Context()

	// 1. Fetch User
	var user UserWithOrders
	err := db.QueryRowContext(ctx, "SELECT Id, Name, Email, CreatedAt FROM Users WHERE Id = @Id", sql.Named("Id", userId)).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Fetch Last 5 Orders
	rows, err := db.QueryContext(ctx, "SELECT TOP 5 Id, Amount, Status, CreatedAt FROM Orders WHERE UserId = @UserId ORDER BY CreatedAt DESC", sql.Named("UserId", userId))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var o OrderInfo
			if err := rows.Scan(&o.ID, &o.Amount, &o.Status, &o.CreatedAt); err == nil {
				user.Orders = append(user.Orders, o)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getOrderPayment(w http.ResponseWriter, r *http.Request) {
	orderId := chi.URLParam(r, "id")
	ctx := r.Context()

	var order OrderWithPayment
	query := `
    SELECT o.Id, o.Amount, o.Status, ISNULL(p.Status, 'UNPAID') as PaymentStatus, p.SettledAt
    FROM Orders o
    LEFT JOIN Payments p ON o.Id = p.OrderId
    WHERE o.Id = @OrderID`

	err := db.QueryRowContext(ctx, query, sql.Named("OrderID", orderId)).
		Scan(&order.ID, &order.Amount, &order.Status, &order.PaymentStatus, &order.SettledAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func getDLQ(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := redisClient.LRange(ctx, "event_pipeline_dlq", 0, 19).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var result []interface{}
	for _, item := range items {
		var m interface{}
		json.Unmarshal([]byte(item), &m)
		result = append(result, m)
	}
	json.NewEncoder(w).Encode(result)
}

func produceEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// In a real system, you'd extract the key from the payload.
	// For this test endpoint, we'll just use a random key or let it be empty.
	err := kafkaWriter.WriteMessages(ctx, kafka.Message{
		Value: payload,
	})
	if err != nil {
		http.Error(w, "Failed to send to Kafka: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"event sent to kafka"}`))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
