package kafka

import (
	"github.com/segmentio/kafka-go"
	"time"
)

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

func NewKafkaReader(cfg KafkaConfig) *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          cfg.Brokers,
		Topic:            cfg.Topic,
		GroupID:          cfg.GroupID,
		MinBytes:         10e3, // 10KB
		MaxBytes:         10e6, // 10MB
		ReadBatchTimeout: 1 * time.Second,
		CommitInterval:   time.Second,
	})
	return reader
}
