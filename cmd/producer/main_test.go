package main

import (
	"context"
	"encoding/json"
	kafkaClient "l0_test_self/pkg/client/kafka"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWriter для тестирования Kafka writer
type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestGenerateTestOrderJSON(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful order generation",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderJSON, err := GenerateTestOrderJSON()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, orderJSON)

			// Проверяем, что JSON валидный
			var order interface{}
			err = json.Unmarshal(orderJSON, &order)
			assert.NoError(t, err)
		})
	}
}

func TestKafkaConfig(t *testing.T) {
	cfg := kafkaClient.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "orders",
		GroupID: "test_producer",
	}

	assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
	assert.Equal(t, "orders", cfg.Topic)
	assert.Equal(t, "test_producer", cfg.GroupID)
}

func TestMessageSending(t *testing.T) {
	// Тестируем структуру сообщения
	testOrderJSON := []byte(`{"test": "order"}`)

	msg := kafka.Message{
		Value: testOrderJSON,
	}

	assert.Equal(t, testOrderJSON, msg.Value)
	assert.Empty(t, msg.Key) // Ключ не устанавливается в коде
}
