package cache

import (
	"container/list"
	"sync"
)

// LRUCache is a thread-safe Least Recently Used cache
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.RWMutex
}

// entry holds the key-value pair for the cache
type entry struct {
	key   string
	value interface{}
}

// NewLRUCache creates a new LRUCache with the given capacity
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves a value from the cache
func (l *LRUCache) Get(key string) (interface{}, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if elem, ok := l.cache[key]; ok {
		l.list.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

// Put stores a key-value pair in the cache
func (l *LRUCache) Put(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// If key exists, update value and move to front
	if elem, ok := l.cache[key]; ok {
		l.list.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}
	
	// If at capacity, remove least recently used
	if l.list.Len() >= l.capacity {
		back := l.list.Back()
		if back != nil {
			l.list.Remove(back)
			delete(l.cache, back.Value.(*entry).key)
		}
	}
	
	// Add new entry
	elem := l.list.PushFront(&entry{key, value})
	l.cache[key] = elem
}

// Delete removes a key from the cache
func (l *LRUCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if elem, ok := l.cache[key]; ok {
		l.list.Remove(elem)
		delete(l.cache, key)
	}
}

// Clear empties the entire cache
func (l *LRUCache) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.cache = make(map[string]*list.Element)
	l.list.Init()
}

// Size returns the current number of items in the cache
func (l *LRUCache) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.list.Len()
}

// Keys returns all keys in the cache (from most to least recent)
func (l *LRUCache) Keys() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	keys := make([]string, 0, l.list.Len())
	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*entry).key)
	}
	return keys
}