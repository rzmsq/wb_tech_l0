package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/pgxpool"
	"l0_test_self/models/orders"
	"l0_test_self/pkg/utils"
	"time"
)

// DBConfig holds the configuration for connecting to a PostgreSQL database.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Client is an interface that defines methods for interacting with a PostgreSQL database.
type Client interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// NewClient creates a new PostgreSQL client with the provided configuration and connection retry logic.
func NewClient(ctx context.Context, config DBConfig, maxAttempts int) (pool *pgxpool.Pool, err error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.DBName, config.SSLMode)

	err = repeatable.DoWithTries(func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		pool, err = pgxpool.Connect(ctx, dsn)
		if err != nil {
			return err
		}

		return nil
	}, maxAttempts, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxAttempts, err)
	}

	return pool, nil
}

func InsertOrder(ctx context.Context, pool *pgxpool.Pool, order *orders.Order) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert into orders table
	orderSQL := `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = tx.Exec(ctx, orderSQL, order.OrderUid, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerId, order.DeliveryService, order.Shardkey, order.SmId, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("failed to insert into orders: %w", err)
	}

	// Insert into delivery table
	deliverySQL := `INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = tx.Exec(ctx, deliverySQL, order.OrderUid, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("failed to insert into delivery: %w", err)
	}

	// Insert into payment table
	// NOTE: The payment table schema should have an order_uid foreign key.
	// Assuming transaction_id is the order_uid for this relationship.
	paymentSQL := `INSERT INTO payment (transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(ctx, paymentSQL, order.Payment.Transaction, order.Payment.RequestId, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("failed to insert into payment: %w", err)
	}

	// Insert into items table
	itemSQL := `INSERT INTO items (chrt_id, order_uid, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	for _, item := range order.Items {
		_, err = tx.Exec(ctx, itemSQL, item.ChrtId, order.OrderUid, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmId, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("failed to insert item with chrt_id %d: %w", item.ChrtId, err)
		}
	}

	return tx.Commit(ctx)
}

// GetAllOrders retrieves all orders from the database to populate the cache.
func GetAllOrders(ctx context.Context, pool *pgxpool.Pool) ([]orders.Order, error) {
	// 1. Get all base orders
	orderSQL := `SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders`
	rows, err := pool.Query(ctx, orderSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	orderMap := make(map[string]*orders.Order)

	for rows.Next() {
		var o orders.Order
		err := rows.Scan(&o.OrderUid, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerId, &o.DeliveryService, &o.Shardkey, &o.SmId, &o.DateCreated, &o.OofShard)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orderMap[o.OrderUid] = &o
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", rows.Err())
	}

	// 2. Get all deliveries and map them
	deliverySQL := `SELECT order_uid, name, phone, zip, city, address, region, email FROM delivery`
	deliveryRows, err := pool.Query(ctx, deliverySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query deliveries: %w", err)
	}
	defer deliveryRows.Close()

	for deliveryRows.Next() {
		var orderUid string
		var d orders.Delivery
		err := deliveryRows.Scan(&orderUid, &d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}
		if order, ok := orderMap[orderUid]; ok {
			order.Delivery = d
		}
	}
	if deliveryRows.Err() != nil {
		return nil, fmt.Errorf("error iterating delivery rows: %w", deliveryRows.Err())
	}

	// 3. Get all payments and map them
	// Assuming 'transaction_id' in the payment table corresponds to 'order_uid'
	paymentSQL := `SELECT transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment`
	paymentRows, err := pool.Query(ctx, paymentSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer paymentRows.Close()

	for paymentRows.Next() {
		var p orders.Payment
		err := paymentRows.Scan(&p.Transaction, &p.RequestId, &p.Currency, &p.Provider, &p.Amount, &p.PaymentDt, &p.Bank, &p.DeliveryCost, &p.GoodsTotal, &p.CustomFee)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		// The key for the map is order_uid, which is in p.Transaction
		if order, ok := orderMap[p.Transaction]; ok {
			order.Payment = p
		}
	}
	if paymentRows.Err() != nil {
		return nil, fmt.Errorf("error iterating payment rows: %w", paymentRows.Err())
	}

	// 4. Get all items and map them
	itemSQL := `SELECT chrt_id, order_uid, track_number, price, rid, name, sale, "size", total_price, nm_id, brand, status FROM items`
	itemRows, err := pool.Query(ctx, itemSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var orderUid string
		var i orders.Item
		err := itemRows.Scan(&i.ChrtId, &orderUid, &i.TrackNumber, &i.Price, &i.Rid, &i.Name, &i.Sale, &i.Size, &i.TotalPrice, &i.NmId, &i.Brand, &i.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		if order, ok := orderMap[orderUid]; ok {
			order.Items = append(order.Items, i)
		}
	}
	if itemRows.Err() != nil {
		return nil, fmt.Errorf("error iterating item rows: %w", itemRows.Err())
	}

	// 5. Convert map to slice
	var orderList []orders.Order
	for _, order := range orderMap {
		orderList = append(orderList, *order)
	}

	return orderList, nil
}
