package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	item, ok := c.cache[key]
	if !ok {
		return
	}

	c.ll.MoveToFront(item)
	entry := item.Value.(*entry)
	value = entry.value
	return
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	item := c.ll.Back()
	if item != nil {
		c.ll.Remove(item)
		entry := item.Value.(*entry)
		delete(c.cache, entry.key)

		c.nbytes -= int64(len(entry.key)) + int64(entry.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(entry.key, entry.value)
		}
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	item, ok := c.cache[key]
	if ok {
		c.ll.MoveToFront(item)
		entry := item.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(entry.value.Len())
		entry.value = value

	} else {
		entry := &entry{
			key:   key,
			value: value,
		}
		item = c.ll.PushFront(entry)
		c.cache[key] = item
		c.nbytes += int64(len(entry.key)) + int64(entry.value.Len())
	}

	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
