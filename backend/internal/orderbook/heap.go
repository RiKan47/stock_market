package orderbook

import (
	"container/heap"

	"stock-market/pkg/models"
)

// OrderHeap implements heap.Interface for orders
type OrderHeap []*models.Order

// MinHeap for asks (sell orders) - lowest price first
type MinHeap struct {
	OrderHeap
}

// MaxHeap for bids (buy orders) - highest price first
type MaxHeap struct {
	OrderHeap
}

// NewMinHeap creates a new min heap
func NewMinHeap() *MinHeap {
	mh := &MinHeap{}
	heap.Init(&mh.OrderHeap)
	return mh
}

// NewMaxHeap creates a new max heap
func NewMaxHeap() *MaxHeap {
	mh := &MaxHeap{}
	heap.Init(&mh.OrderHeap)
	return mh
}

// Len returns the number of elements in the heap
func (h OrderHeap) Len() int {
	return len(h)
}

// Less defines the ordering for the heap
func (h OrderHeap) Less(i, j int) bool {
	// Compare by price for min heap (asks)
	if h[i].Price == h[j].Price {
		// If same price, older order gets priority
		return h[i].CreatedAt.Before(h[j].CreatedAt)
	}
	return h[i].Price < h[j].Price
}

// Less for MaxHeap (bids) - highest price first
func (h MaxHeap) Less(i, j int) bool {
	if h.OrderHeap[i].Price == h.OrderHeap[j].Price {
		return h.OrderHeap[i].CreatedAt.Before(h.OrderHeap[j].CreatedAt)
	}
	return h.OrderHeap[i].Price > h.OrderHeap[j].Price
}

// Swap swaps elements at indices i and j
func (h OrderHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push adds an element to the heap
func (h *OrderHeap) Push(x interface{}) {
	*h = append(*h, x.(*models.Order))
}

// Pop removes and returns the smallest element from the heap
func (h *OrderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// Peek returns the top element without removing it
func (h *OrderHeap) Peek() interface{} {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}