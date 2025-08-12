package main

import (
	"context"
	"encoding/json"
	"fmt"
	"l0_test_self/internal/cache"
	"l0_test_self/internal/config"
	"log"
	"net/http"
	"time"

	"l0_test_self/models/orders"
	"l0_test_self/pkg/client/kafka"
	"l0_test_self/pkg/client/postgres"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database connection using config
	dbCfg := cfg.Database.ToPostgresConfig()
	pool, err := postgres.NewClient(ctx, dbCfg, cfg.Database.MaxConnections)
	if err != nil {
		fmt.Printf("Error creating database pool: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("Database connection pool created successfully")

	// Initialize cache
	orderCache := cache.New()
	log.Println("Cache initialized")

	existingOrders, err := postgres.GetAllOrders(ctx, pool)
	if err != nil {
		log.Fatalf("Error fetching existing orders: %v\n", err)
	}
	orderCache.LoadFromSlice(existingOrders)
	log.Printf("Order cache initialized")

	// Initialize Kafka reader using config
	reader := kafka.NewKafkaReader(cfg.Kafka.ToKafkaConfig())
	defer reader.Close()

	log.Println("Reader opened successfully")

	// Message processing loop
	go func() {
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Printf("Received message: %s\n", string(msg.Value))

			var order orders.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Error parsing order data: %v\n", err)
				continue
			}

			if err := postgres.InsertOrder(ctx, pool, &order); err != nil {
				log.Printf("Error inserting order into database: %v\n", err)
				continue
			}
			log.Printf("Order %s inserted successfully\n", order.OrderUid)

			orderCache.Set(order)
			log.Printf("Order %s cached successfully\n", order.OrderUid)
		}
	}()

	// Setup HTTP server
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	http.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		orderID := r.URL.Query().Get("id")
		if orderID == "" {
			http.Error(w, "Order ID is required", http.StatusBadRequest)
			return
		}

		order, found := orderCache.Get(orderID)
		if !found {
			log.Printf("Order %s not found in cache, checking DB", orderID)
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	})

	log.Printf("Starting server on %s", cfg.Server.Port)
	if err := http.ListenAndServe(cfg.Server.Port, nil); err != nil {
		log.Fatalf("could not start server: %v\n", err)
	}
}
