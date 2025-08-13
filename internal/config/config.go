// Package config содержит структуры и функции для загрузки конфигурации приложения из файла YAML.
package config

import (
	"os"
	"time"

	"l0_test_self/pkg/client/kafka"
	"l0_test_self/pkg/client/postgres"

	"gopkg.in/yaml.v3"
)

// CacheConfig содержит настройки кэша
type CacheConfig struct {
	ShardCount      int           `yaml:"shard_count"`
	MaxItems        int           `yaml:"max_items"`
	TTL             time.Duration `yaml:"ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// Config содержит настройки приложения, включая параметры подключения к базе данных PostgreSQL, конфигурацию Kafka и настройки сервера.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Server   ServerConfig   `yaml:"server"`
	Cache    CacheConfig    `yaml:"cache"`
	Test     TestConfig     `yaml:"test"`
}

// TestConfig содержит настройки для тестов
type TestConfig struct {
	Kafka TestKafkaConfig `yaml:"kafka"`
}

// TestKafkaConfig содержит настройки Kafka для тестов
type TestKafkaConfig struct {
	Brokers            []string `yaml:"brokers"`
	Topic              string   `yaml:"topic"`
	IntegrationGroupID string   `yaml:"integration_group_id"`
	MultipleGroupID    string   `yaml:"multiple_group_id"`
	BenchmarkGroupID   string   `yaml:"benchmark_group_id"`
	BenchmarkTopic     string   `yaml:"benchmark_topic"`
}

// DatabaseConfig Config содержит настройки приложения, включая параметры подключения к базе данных PostgreSQL, конфигурацию Kafka и настройки сервера.
type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	DBName         string `yaml:"db_name"`
	SSLMode        string `yaml:"ssl_mode"`
	MaxConnections int    `yaml:"max_connections"`
}

// KafkaConfig DatabaseConfig содержит настройки для подключения к базе данных PostgreSQL, такие как хост, порт, пользователь, пароль, имя базы данных и режим SSL.
type KafkaConfig struct {
	Brokers []string     `yaml:"brokers"`
	Topic   string       `yaml:"topic"`
	GroupID string       `yaml:"group_id"`
	Reader  ReaderConfig `yaml:"reader"`
	Writer  WriterConfig `yaml:"writer"`
}

// ReaderConfig содержит настройки для Kafka Reader, такие как минимальный и максимальный размер сообщений, таймауты и интервал коммита.
type ReaderConfig struct {
	MinBytes         int           `yaml:"min_bytes"`
	MaxBytes         int           `yaml:"max_bytes"`
	ReadBatchTimeout time.Duration `yaml:"read_batch_timeout"`
	CommitInterval   time.Duration `yaml:"commit_interval"`
}

// WriterConfig содержит настройки для Kafka Writer, такие как таймауты и балансировщик нагрузки.
type WriterConfig struct {
	WriteTimeout time.Duration `yaml:"write_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	Balancer     string        `yaml:"balancer"`
}

// ServerConfig содержит настройки сервера, такие как порт.
type ServerConfig struct {
	Port            string        `yaml:"port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// Load загружает конфигурацию из файла YAML по указанному пути.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ToPostgresConfig преобразует DatabaseConfig в postgres.DBConfig.
func (c *DatabaseConfig) ToPostgresConfig() postgres.DBConfig {
	return postgres.DBConfig{
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Password: c.Password,
		DBName:   c.DBName,
		SSLMode:  c.SSLMode,
	}
}

// ToKafkaConfig NewKafkaReader создает новый Kafka Reader с использованием конфигурации из KafkaConfig.
func (c *KafkaConfig) ToKafkaConfig() kafka.Config {
	return kafka.Config{
		Brokers: c.Brokers,
		Topic:   c.Topic,
		GroupID: c.GroupID,
		Reader:  kafka.ReaderConfig(c.Reader),
		Writer:  kafka.WriterConfig(c.Writer),
	}
}
