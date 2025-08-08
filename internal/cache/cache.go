package cache

import (
	"l0_test_self/models/orders"
	"sync"
)

// OrderCache holds the in-memory cache for orders.
type OrderCache struct {
	mu    sync.RWMutex
	items map[string]orders.Order
}

// New creates a new OrderCache
func New() *OrderCache {
	return &OrderCache{
		items: make(map[string]orders.Order),
	}
}

func (c *OrderCache) Set(order orders.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[order.OrderUid] = order
}

// Get retrieves an order from the cache
func (c *OrderCache) Get(orderUid string) (orders.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, found := c.items[orderUid]
	return order, found
}

// LoafFromSlice loads multiple orders from a slice into the cache
func (c *OrderCache) LoadFromSlice(ordersSlice []orders.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, order := range ordersSlice {
		c.items[order.OrderUid] = order
	}
}
