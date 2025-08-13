// Описание: Это основной файл-сервера, который инициализирует все компоненты приложения, включая базу данных, кэш и Kafka.
// Он также обрабатывает HTTP-запросы и управляет жизненным циклом приложения
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"l0_test_self/internal/cache"
	"l0_test_self/internal/config"
	"l0_test_self/internal/validation"
	"l0_test_self/models/orders"
	"l0_test_self/pkg/client/kafka"
	"l0_test_self/pkg/client/postgres"

	"github.com/jackc/pgx/v4/pgxpool"
	kafka2 "github.com/segmentio/kafka-go"
)

const configPath = "../../config.yaml"

// OrderCache - интерфейс для кэша заказов
type OrderCache interface {
	Set(order orders.Order)
	Get(id string) (orders.Order, bool)
	LoadFromSlice([]orders.Order)
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

// run - основная функция запуска сервера
func run() error {
	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем логирование
	logger := log.New(os.Stdout, "[srv] ", log.LstdFlags|log.Lmicroseconds)

	// Загружаем конфигурацию
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Инициализируем компоненты приложения
	dbCfg := cfg.Database.ToPostgresConfig()
	pool, err := postgres.NewClient(ctx, dbCfg, cfg.Database.MaxConnections) // returns v4 pool
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Println("database pool ready")

	// Инициализируем кэш
	cc, err := cache.New(cfg.Cache.ShardCount, cfg.Cache.MaxItems, cfg.Cache.TTL, cfg.Cache.CleanupInterval)
	if err != nil {
		return err
	}
	defer cc.Close()
	logger.Println("cache initialized")

	// Загружаем существующие заказы в кэш
	existingOrders, err := postgres.GetAllOrders(ctx, pool)
	if err != nil {
		return err
	}
	cc.LoadFromSlice(existingOrders)
	logger.Printf("loaded %d orders into cache", len(existingOrders))

	// Инициализируем Kafka reader
	reader := kafka.NewKafkaReader(cfg.Kafka.ToKafkaConfig())
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			logger.Printf("kafka reader close error: %v", cerr)
		}
	}()
	logger.Println("kafka reader ready")

	// Проверяем подключение к Kafka
	wg := startKafkaConsumer(ctx, reader, pool, cc, logger, cfg)

	// Запускаем HTTP сервер
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("../../web")))
	mux.HandleFunc("/order", makeOrderHandler(cc, logger))

	server := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: mux,
	}

	// Настраиваем таймауты для сервера
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Устанавливаем таймауты для сервера
	go func() {
		sig := <-sigCh
		logger.Printf("shutdown signal: %v", sig)
		cancel()

		shCtx, shCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shCancel()

		if err := server.Shutdown(shCtx); err != nil {
			logger.Printf("http shutdown error: %v", err)
		} else {
			logger.Println("http server stopped gracefully")
		}
	}()

	// Запускаем HTTP сервер
	logger.Printf("http server starting on %s", cfg.Server.Port)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Ждем завершения работы Kafka consumer
	wg.Wait()
	logger.Println("graceful shutdown complete")
	return nil
}

// startKafkaConsumer - запускает Kafka consumer в отдельной горутине
func startKafkaConsumer(
	ctx context.Context,
	reader *kafka2.Reader,
	pool *pgxpool.Pool, // now v4
	orderCache OrderCache,
	logger *log.Logger,
	cfg *config.Config,
) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Запускаем Kafka consumer в отдельной горутине
	go func() {
		defer wg.Done()
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || ctx.Err() != nil {
					logger.Println("kafka consumer stopping (context canceled)")
					return
				}
				logger.Printf("kafka read error: %v", err)
				time.Sleep(cfg.Kafka.Reader.ReadBatchTimeout)
				continue
			}

			logger.Printf("kafka message received: %s", string(msg.Value))

			var order orders.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				logger.Printf("json unmarshal error: %v", err)
				continue
			}
			if err := validation.ValidateOrder(&order); err != nil {
				logger.Printf("validation error (skip message): %v", err)
				continue
			}

			if err := postgres.InsertOrder(ctx, pool, &order); err != nil {
				logger.Printf("db insert error (order=%s): %v", order.OrderUid, err)
				continue
			}
			logger.Printf("order %s stored", order.OrderUid)

			orderCache.Set(order)
			logger.Printf("order %s cached", order.OrderUid)
		}
	}()

	return wg
}

// makeOrderHandler - HTTP обработчик для получения заказа по ID
func makeOrderHandler(orderCache OrderCache, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.URL.Query().Get("id")
		if orderID == "" {
			http.Error(w, "order id is required", http.StatusBadRequest)
			return
		}

		if !validation.ValidateOrderID(orderID) {
			http.Error(w, "invalid order id format", http.StatusBadRequest)
			return
		}

		order, ok := orderCache.Get(orderID)
		if !ok {
			logger.Printf("order %s not found", orderID)
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(order); err != nil {
			logger.Printf("encode error: %v", err)
		}
	}
}
