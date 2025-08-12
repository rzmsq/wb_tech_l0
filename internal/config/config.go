package config

import (
	"os"
	"time"

	"l0_test_self/pkg/client/kafka"
	"l0_test_self/pkg/client/postgres"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Server   ServerConfig   `yaml:"server"`
}

type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	DBName         string `yaml:"db_name"`
	SSLMode        string `yaml:"ssl_mode"`
	MaxConnections int    `yaml:"max_connections"`
}

type KafkaConfig struct {
	Brokers []string     `yaml:"brokers"`
	Topic   string       `yaml:"topic"`
	GroupID string       `yaml:"group_id"`
	Reader  ReaderConfig `yaml:"reader"`
	Writer  WriterConfig `yaml:"writer"`
}

type ReaderConfig struct {
	MinBytes         int           `yaml:"min_bytes"`
	MaxBytes         int           `yaml:"max_bytes"`
	ReadBatchTimeout time.Duration `yaml:"read_batch_timeout"`
	CommitInterval   time.Duration `yaml:"commit_interval"`
}

type WriterConfig struct {
	WriteTimeout time.Duration `yaml:"write_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	Balancer     string        `yaml:"balancer"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

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

func (c *KafkaConfig) ToKafkaConfig() kafka.Config {
	return kafka.Config{
		Brokers: c.Brokers,
		Topic:   c.Topic,
		GroupID: c.GroupID,
		Reader:  kafka.ReaderConfig(c.Reader),
		Writer:  kafka.WriterConfig(c.Writer),
	}
}
