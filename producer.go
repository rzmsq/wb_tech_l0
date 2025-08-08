package main

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

// Order represents the full order data structure.
// Заказ представляет полную структуру данных заказа.
type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	Shardkey          string   `json:"shardkey"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

// Delivery holds delivery information.
// Delivery содержит информацию о доставке.
type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

// Payment holds payment information.
// Payment содержит информацию об оплате.
type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

// Item holds information about a single item.
// Item содержит информацию об отдельном товаре.
type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

func main() {
	// Generate a unique ID for the order
	orderID := uuid.New().String()

	// Создаем экземпляр заказа на основе предоставленной JSON-модели
	order := Order{
		OrderUID:    orderID,
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: Payment{
			Transaction:  orderID, // Use the same unique ID
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         uuid.New().String(), // Also generate a unique rid
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       "2021-11-26T06:22:19Z",
		OofShard:          "1",
	}

	// Сериализуем структуру в JSON
	jsonData, err := json.Marshal(order)
	if err != nil {
		log.Fatalf("Ошибка при сериализации JSON: %v", err)
	}

	// Настройка писателя (writer) для Kafka
	writer := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9092"), // Адрес вашего брокера Kafka
		Topic:        "orders",                    // Имя топика, в который отправляем сообщение
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	defer writer.Close()

	// Создаем сообщение Kafka
	msg := kafka.Message{
		Key:   []byte(order.OrderUID),
		Value: jsonData,
	}

	// Отправляем сообщение в Kafka
	err = writer.WriteMessages(context.Background(), msg)
	if err != nil {
		log.Fatalf("Ошибка при отправке сообщения в Kafka: %v", err)
	}

	log.Printf("Сообщение успешно отправлено в топик 'orders': %s", string(jsonData))
}
