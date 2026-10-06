package subsonic

import (
	"sync"
	"time"
)

type ttlItem[V any] struct {
	value   V
	expires time.Time
}

type ttlCache[V any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	max   int
	items map[string]ttlItem[V]
}

func newTTLCache[V any](ttl time.Duration, max int) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, max: max, items: make(map[string]ttlItem[V])}
}

func (c *ttlCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.expires) {
		delete(c.items, key)
		var zero V
		return zero, false
	}
	return item.value, true
}

func (c *ttlCache[V]) set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok && len(c.items) >= c.max {
		now := time.Now()
		for k, item := range c.items {
			if now.After(item.expires) {
				delete(c.items, k)
			}
		}
		if len(c.items) >= c.max {
			clear(c.items)
		}
	}
	c.items[key] = ttlItem[V]{value: value, expires: time.Now().Add(c.ttl)}
}
