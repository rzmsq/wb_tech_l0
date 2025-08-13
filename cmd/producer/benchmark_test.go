package main

import (
	"testing"

	"l0_test_self/internal/config"
	kafkaClient "l0_test_self/pkg/client/kafka"
)

func BenchmarkWriterCreation(b *testing.B) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		b.Fatal(err)
	}

	kafkaCfg := kafkaClient.Config{
		Brokers: cfg.Test.Kafka.Brokers,
		Topic:   cfg.Test.Kafka.BenchmarkTopic,
		GroupID: cfg.Test.Kafka.BenchmarkGroupID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer := kafkaClient.NewWriter(kafkaCfg)
		_ = writer.Close()
	}
}
