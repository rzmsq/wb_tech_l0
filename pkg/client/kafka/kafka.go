// Package kafka для работы с Kafka, включая конфигурацию и создание читателей и писателей.
package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Config содержит настройки для подключения к Kafka, включая брокеры, топик, группу и конфигурации читателя и писателя.
type Config struct {
	Brokers []string     `yaml:"brokers"`
	Topic   string       `yaml:"topic"`
	GroupID string       `yaml:"group_id"`
	Reader  ReaderConfig `yaml:"reader"`
	Writer  WriterConfig `yaml:"writer"`
}

// ReaderConfig содержит настройки для Kafka Reader, такие, как минимальный и максимальный размер сообщений, таймауты и интервал коммита.
type ReaderConfig struct {
	MinBytes         int           `yaml:"min_bytes"`
	MaxBytes         int           `yaml:"max_bytes"`
	ReadBatchTimeout time.Duration `yaml:"read_batch_timeout"`
	CommitInterval   time.Duration `yaml:"commit_interval"`
}

// WriterConfig содержит настройки для Kafka Writer, такие, как таймауты и балансировщик нагрузки.
type WriterConfig struct {
	WriteTimeout time.Duration `yaml:"write_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	Balancer     string        `yaml:"balancer"`
}

// NewKafkaReader создает новый Kafka Reader с использованием конфигурации из Config.
func NewKafkaReader(cfg Config) *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          cfg.Brokers,
		Topic:            cfg.Topic,
		GroupID:          cfg.GroupID,
		MinBytes:         cfg.Reader.MinBytes,
		MaxBytes:         cfg.Reader.MaxBytes,
		ReadBatchTimeout: cfg.Reader.ReadBatchTimeout,
		CommitInterval:   cfg.Reader.CommitInterval,
	})
	return reader
}

// NewWriter создает новый Kafka Writer с использованием конфигурации из Config.
func NewWriter(cfg Config) *kafka.Writer {
	var balancer kafka.Balancer
	switch cfg.Writer.Balancer {
	case "least_bytes":
		balancer = &kafka.LeastBytes{}
	case "round_robin":
		balancer = &kafka.RoundRobin{}
	default:
		balancer = &kafka.LeastBytes{}
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     balancer,
		WriteTimeout: cfg.Writer.WriteTimeout,
		ReadTimeout:  cfg.Writer.ReadTimeout,
	}
}
