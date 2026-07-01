package main

import (
	"container/list"
	"sync"

	"github.com/WangYihang/tranco-go-package"
)

// trancoListCache is a fixed-capacity, concurrency-safe LRU cache of
// *tranco.TrancoList keyed by date. Each entry holds a full list in memory
// (potentially millions of rows), so without a bound a long-running server
// queried about many distinct dates would grow its memory use forever;
// this evicts the least-recently-used date once capacity is exceeded.
type trancoListCache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[string]*list.Element
}

type cacheEntry struct {
	key   string
	value *tranco.TrancoList
}

func newTrancoListCache(capacity int) *trancoListCache {
	return &trancoListCache{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *trancoListCache) get(key string) (*tranco.TrancoList, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(elem)
	return elem.Value.(*cacheEntry).value, true
}

func (c *trancoListCache) set(key string, value *tranco.TrancoList) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value.(*cacheEntry).value = value
		return
	}

	elem := c.ll.PushFront(&cacheEntry{key: key, value: value})
	c.items[key] = elem

	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
}
