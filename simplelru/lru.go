package simplelru

import (
	"container/list"
	"errors"
	"fmt"
)

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback func(key interface{}, value interface{}, cost int64)

// LRU implements a non-thread safe fixed size LRU cache
type LRU struct {
	maxCost       int64
	evictList     *list.List
	evictListCost int64
	items         map[interface{}]*list.Element
	onEvict       EvictCallback
}

// entry is used to hold a value in the evictList
type entry struct {
	key   interface{}
	value interface{}
	cost  int64
}

// NewLRU constructs an LRU of the given size
func NewLRU(maxCost int64, onEvict EvictCallback) (*LRU, error) {
	if maxCost <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU{
		maxCost:       maxCost,
		evictListCost: 0,
		evictList:     list.New(),
		items:         make(map[interface{}]*list.Element),
		onEvict:       onEvict,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *LRU) Purge() {
	for k, v := range c.items {
		en := v.Value.(*entry)
		c.callOnEvict(en)
		delete(c.items, k)
	}
	c.evictList.Init()
	c.evictListCost = 0
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU) Add(key, value interface{}, cost int64) (evicted int) {
	// Check for existing item - cost can't be updated
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*entry).value = value
		return 0
	}

	if cost > c.maxCost {
		panic(fmt.Errorf("cost %d is bigger than max cost %d", cost, c.maxCost))
	}

	// Add new item
	ent := &entry{key, value, cost}
	entry := c.evictList.PushFront(ent)
	c.evictListCost += cost
	c.items[key] = entry

	// Verify size not exceeded
	for c.evictListCost > c.maxCost {
		evicted++
		c.removeOldest()
	}
	return evicted
}

// Get looks up a key's value from the cache.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		if ent.Value.(*entry) == nil {
			return nil, false
		}
		return ent.Value.(*entry).value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU) Contains(key interface{}) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	var ent *list.Element
	if ent, ok = c.items[key]; ok {
		return ent.Value.(*entry).value, true
	}
	return nil, ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU) Remove(key interface{}) (present bool) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU) RemoveOldest() (key, value interface{}, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
		kv := ent.Value.(*entry)
		return kv.key, kv.value, true
	}
	return nil, nil, false
}

// GetOldest returns the oldest entry
func (c *LRU) GetOldest() (key, value interface{}, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		kv := ent.Value.(*entry)
		return kv.key, kv.value, true
	}
	return nil, nil, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU) Keys() []interface{} {
	keys := make([]interface{}, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*entry).key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
func (c *LRU) Len() int {
	return c.evictList.Len()
}

// Cost returns the total cost of items in the cache
func (c *LRU) Cost() int64 {
	return c.evictListCost
}

// Resize changes the cache size.
func (c *LRU) Resize(maxCost int64) (evicted int) {
	if maxCost <= 0 {
		panic(errors.New("must provide a positive size"))
	}
	c.maxCost = maxCost
	for c.evictListCost > c.maxCost {
		evicted++
		c.removeOldest()
	}
	return evicted
}

// removeOldest removes the oldest item from the cache.
func (c *LRU) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.items, kv.key)
	c.evictListCost -= kv.cost
	c.callOnEvict(kv)
}

// callOnEvict calls onEvict and blocks if needed
func (c *LRU) callOnEvict(e *entry) {
	if c.onEvict == nil {
		return
	}

	c.onEvict(e.key, e.value, e.cost)
}
