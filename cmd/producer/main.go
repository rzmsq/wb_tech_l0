package main

import (
	"context"
	"log"
	"time"

	kafkaClient "l0_test_self/pkg/client/kafka"

	"github.com/segmentio/kafka-go"
)

func main() {
	ctx := context.Background()

	// Конфигурация Kafka
	kafkaCfg := kafkaClient.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "orders",
		GroupID: "test_producer",
	}

	writer := kafkaClient.NewWriter(kafkaCfg)
	defer func(writer *kafka.Writer) {
		err := writer.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(writer)

	// Генерируем и отправляем тестовые заказы
	for i := 0; i < 10; i++ {
		orderJSON, err := GenerateTestOrderJSON()
		if err != nil {
			log.Printf("Error generating test order: %v", err)
			continue
		}

		msg := kafka.Message{
			Value: orderJSON,
		}

		if err := writer.WriteMessages(ctx, msg); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Test order %d sent successfully", i+1)
		}

		time.Sleep(2 * time.Second)
	}

	log.Println("All test orders sent")
}
