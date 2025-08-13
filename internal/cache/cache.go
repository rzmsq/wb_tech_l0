// Package cache реализует кэш для заказов с поддержкой LRU и TTL.
package cache

import (
	"container/list"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"l0_test_self/models/orders"
)

// orderEntry представляет собой элемент кэша, который хранит заказ и метаданные.
type orderEntry struct {
	key       string
	value     orders.Order
	createdAt time.Time
	elem      *list.Element
}

// Shard представляет собой отдельный сегмент кэша, который использует блокировку для обеспечения потокобезопасности.
type shard struct {
	mu    sync.RWMutex
	items map[string]*orderEntry
	lru   *list.List
}

// OrderCache представляет собой кэш заказов, который использует шардирование для повышения производительности и масштабируемости.
type OrderCache struct {
	shards         []*shard
	mask           uint32
	perShardCap    int
	ttl            time.Duration
	cleanupEvery   time.Duration
	stopCh         chan struct{}
	cleanupStarted sync.Once
}

// New создает новый экземпляр OrderCache с заданным количеством шардов, максимальным количеством элементов, временем жизни элементов и интервалом очистки.
func New(shardCount int, maxItems int, ttl time.Duration, cleanupInterval time.Duration) (*OrderCache, error) {
	if shardCount <= 0 {
		return nil, errors.New("shardCount must be > 0")
	}
	if maxItems < 0 {
		return nil, errors.New("maxItems must be >= 0")
	}
	if ttl < 0 {
		return nil, errors.New("ttl must be >= 0")
	}
	if cleanupInterval < 0 {
		return nil, errors.New("cleanupInterval must be >= 0")
	}
	if maxItems > 0 && maxItems < shardCount {
		return nil, errors.New("maxItems must be >= shardCount (or 0 for unlimited)")
	}

	// round shards to power of two
	sc := 1
	for sc < shardCount {
		sc <<= 1
	}

	c := &OrderCache{
		shards:       make([]*shard, sc),
		mask:         uint32(sc - 1),
		ttl:          ttl,
		cleanupEvery: cleanupInterval,
		stopCh:       make(chan struct{}),
	}
	for i := 0; i < sc; i++ {
		c.shards[i] = &shard{
			items: make(map[string]*orderEntry),
			lru:   list.New(),
		}
	}
	if maxItems > 0 {
		per := maxItems / sc
		if per == 0 {
			per = 1
		}
		c.perShardCap = per
	}
	if c.ttl > 0 && c.cleanupEvery <= 0 {
		c.cleanupEvery = time.Minute
	}
	if c.ttl > 0 || c.perShardCap > 0 {
		c.startCleaner()
	}
	return c, nil
}

// startCleaner запускает фоновый процесс для периодической очистки кэша от устаревших и наименее используемых элементов.
func (c *OrderCache) startCleaner() {
	c.cleanupStarted.Do(func() {
		if c.cleanupEvery <= 0 {
			return
		}
		ticker := time.NewTicker(c.cleanupEvery)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					c.evictExpired()
				case <-c.stopCh:
					return
				}
			}
		}()
	})
}

// Close останавливает фоновый процесс очистки и закрывает кэш.
func (c *OrderCache) Close() { close(c.stopCh) }

// shardFor вычисляет шард для данного ключа, используя хеш-функцию FNV-1a.
func (c *OrderCache) shardFor(key string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	idx := h.Sum32() & c.mask
	return c.shards[idx]
}

// Set добавляет или обновляет заказ в кэше. Если заказ уже существует, он обновляется, иначе добавляется новый.
func (c *OrderCache) Set(o orders.Order) {
	s := c.shardFor(o.OrderUid)
	now := time.Now()
	s.mu.Lock()
	if ent, ok := s.items[o.OrderUid]; ok {
		ent.value = o
		if c.ttl > 0 {
			ent.createdAt = now
		}
		s.lru.MoveToBack(ent.elem)
		s.mu.Unlock()
		return
	}
	ent := &orderEntry{
		key:       o.OrderUid,
		value:     o,
		createdAt: now,
	}
	ent.elem = s.lru.PushBack(ent)
	s.items[o.OrderUid] = ent
	if c.perShardCap > 0 && s.lru.Len() > c.perShardCap {
		c.evictLRULocked(s, 1)
	}
	s.mu.Unlock()
}

// Get извлекает заказ из кэша по его идентификатору. Если заказ существует и не устарел, он возвращается вместе с флагом успеха.
func (c *OrderCache) Get(id string) (orders.Order, bool) {
	s := c.shardFor(id)
	now := time.Now()
	s.mu.RLock()
	ent, ok := s.items[id]
	if !ok {
		s.mu.RUnlock()
		return orders.Order{}, false
	}
	if c.ttl > 0 && now.Sub(ent.createdAt) > c.ttl {
		s.mu.RUnlock()
		s.mu.Lock()
		if ent2, ok2 := s.items[id]; ok2 && now.Sub(ent2.createdAt) > c.ttl {
			c.removeEntryLocked(s, ent2)
			s.mu.Unlock()
			return orders.Order{}, false
		}
		s.lru.MoveToBack(ent.elem)
		val := ent.value
		s.mu.Unlock()
		return val, true
	}
	val := ent.value
	s.mu.RUnlock()
	s.mu.Lock()
	if ent2, ok2 := s.items[id]; ok2 {
		s.lru.MoveToBack(ent2.elem)
	}
	s.mu.Unlock()
	return val, true
}

// LoadFromSlice загружает список заказов в кэш. Каждый заказ добавляется или обновляется в кэше.
func (c *OrderCache) LoadFromSlice(list []orders.Order) {
	for _, o := range list {
		c.Set(o)
	}
}

// EvictExpired очищает кэш от устаревших элементов, если задано время жизни (TTL).
func (c *OrderCache) evictExpired() {
	if c.ttl <= 0 {
		return
	}
	now := time.Now()
	for _, s := range c.shards {
		s.mu.Lock()
		for e := s.lru.Front(); e != nil; {
			next := e.Next()
			ent := e.Value.(*orderEntry)
			if now.Sub(ent.createdAt) > c.ttl {
				c.removeEntryLocked(s, ent)
			} else {
				break
			}
			e = next
		}
		s.mu.Unlock()
	}
}

// evictLRULocked удаляет n наименее недавно использованных элементов из шардированного кэша.
func (c *OrderCache) evictLRULocked(s *shard, n int) {
	for i := 0; i < n; i++ {
		front := s.lru.Front()
		if front == nil {
			return
		}
		ent := front.Value.(*orderEntry)
		c.removeEntryLocked(s, ent)
	}
}

// removeEntryLocked удаляет элемент из шардированного кэша, освобождая память и удаляя его из LRU списка.
func (c *OrderCache) removeEntryLocked(s *shard, ent *orderEntry) {
	delete(s.items, ent.key)
	s.lru.Remove(ent.elem)
}
