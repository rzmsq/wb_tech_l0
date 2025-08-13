// Описание: Интеграционные тесты для Kafka Producer
package main

import (
	"context"
	"testing"
	"time"

	"l0_test_self/internal/config"
	kafkaClient "l0_test_self/pkg/client/kafka"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GenerateTestOrderJSON - функция для генерации тестового заказа в формате JSON
func TestKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	time.Sleep(5 * time.Second)

	// Загружаем конфиг из файла
	cfg, err := config.Load("../../config.yaml")
	require.NoError(t, err)

	kafkaCfg := kafkaClient.Config{
		Brokers: cfg.Test.Kafka.Brokers,
		Topic:   cfg.Test.Kafka.Topic,
		GroupID: cfg.Test.Kafka.IntegrationGroupID,
	}

	writer := kafkaClient.NewWriter(kafkaCfg)
	require.NotNil(t, writer)

	defer func() {
		err := writer.Close()
		assert.NoError(t, err)
	}()

	orderJSON, err := GenerateTestOrderJSON()
	require.NoError(t, err)
	require.NotEmpty(t, orderJSON)

	msg := kafka.Message{
		Value: orderJSON,
	}

	err = writer.WriteMessages(ctx, msg)
	assert.NoError(t, err)
}

// TestGenerateTestOrderJSON - тест для проверки генерации тестового заказа в формате JSON
func TestMultipleMessagesSending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	time.Sleep(2 * time.Second)

	cfg, err := config.Load("../../config.yaml")
	require.NoError(t, err)

	kafkaCfg := kafkaClient.Config{
		Brokers: cfg.Test.Kafka.Brokers,
		Topic:   cfg.Test.Kafka.Topic,
		GroupID: cfg.Test.Kafka.MultipleGroupID,
	}

	writer := kafkaClient.NewWriter(kafkaCfg)
	require.NotNil(t, writer)

	defer func() {
		err := writer.Close()
		assert.NoError(t, err)
	}()

	messagesCount := 3
	for i := 0; i < messagesCount; i++ {
		orderJSON, err := GenerateTestOrderJSON()
		require.NoError(t, err)

		msg := kafka.Message{
			Value: orderJSON,
		}

		err = writer.WriteMessages(ctx, msg)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	}
}
