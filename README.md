# l0_test_self

## Описание
Проект реализует сервис обработки заказов с использованием микросервисной архитектуры на Go. В качестве брокера сообщений используется Kafka, для хранения данных — PostgreSQL. Проект поддерживает кэширование, валидацию данных и генерацию тестовых данных.

## Структура проекта
- `cmd/producer/` — сервис-отправитель заказов (Kafka producer)
- `cmd/server/` — сервис-обработчик заказов (Kafka consumer, API)
- `internal/cache/` — реализация кэша
- `internal/config/` — работа с конфигурацией
- `internal/validation/` — валидация входящих данных
- `models/orders/` — модели данных заказов
- `pkg/client/kafka/` — клиент Kafka
- `pkg/client/postgres/` — клиент PostgreSQL
- `pkg/utils/` — утилиты
- `web/` — статические файлы 

## Запуск проекта
1. Установите Docker и Docker Compose.
2. Скопируйте файл `config.yaml` и при необходимости измените параметры.
3. Запустите инфраструктуру:
   ```bash
   docker-compose up --build
   ```
4. Запустите сервисы:
   - Producer: `go run cmd/producer/main.go`
   - Server: `go run cmd/server/main.go`

## Тестирование
Для запуска тестов используйте:
```bash
go test ./...
```

## Зависимости
- Go 1.20+
- Kafka
- PostgreSQL
- Docker Compose

