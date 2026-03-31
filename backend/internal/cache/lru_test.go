package cache

import (
	"testing"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(2)
	
	// Test Put and Get
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	
	if val, found := cache.Get("key1"); !found || val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	
	if val, found := cache.Get("key2"); !found || val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}
	
	// Test eviction (capacity is 2)
	cache.Put("key3", "value3") // Should evict key1 (least recently used)
	
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have been evicted")
	}
	
	// Test updating existing key
	cache.Put("key2", "value2-updated")
	if val, found := cache.Get("key2"); !found || val != "value2-updated" {
		t.Errorf("Expected value2-updated, got %v", val)
	}
	
	// Test Delete
	cache.Delete("key3")
	if _, found := cache.Get("key3"); found {
		t.Error("key3 should have been deleted")
	}
	
	// Test Size
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}
	
	// Test Clear
	cache.Clear()
	if size := cache.Size(); size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}