package orderbook

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"
	"time"

	"stock-market/pkg/models"
)

// OrderBook maintains buy and sell orders for a symbol
type OrderBook struct {
	symbol    string
	bids      *MaxHeap // Highest price first (max-heap)
	asks      *MinHeap // Lowest price first (min-heap)
	trades    []models.Trade
	orderMap  map[string]*models.Order // For quick lookup
	lastPrice float64
	mu        sync.RWMutex
}

// NewOrderBook creates a new order book for a symbol
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		symbol:   symbol,
		bids:     NewMaxHeap(),
		asks:     NewMinHeap(),
		trades:   make([]models.Trade, 0),
		orderMap: make(map[string]*models.Order),
	}
}

// AddOrder places a new order in the order book
func (ob *OrderBook) AddOrder(order *models.Order) []models.Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Add to map for quick lookup
	ob.orderMap[order.ID] = order

	// Try to match immediately
	trades := ob.matchOrder(order)

	// If not fully filled, add to appropriate heap
	if order.Remaining > 0 {
		if order.Side == models.OrderSideBuy {
			heap.Push(&ob.bids.OrderHeap, order)
		} else {
			heap.Push(&ob.asks.OrderHeap, order)
		}
	}

	return trades
}

// matchOrder attempts to match an incoming order against the opposite side
func (ob *OrderBook) matchOrder(order *models.Order) []models.Trade {
	trades := make([]models.Trade, 0)

	remaining := order.Remaining

	if order.Side == models.OrderSideBuy {
		// Match against asks (sell orders)
		for remaining > 0 && ob.asks.Len() > 0 {
			opposite := ob.asks.Peek()
			if opposite == nil {
				break
			}
			oppositeOrder := opposite.(*models.Order)

			// For limit orders, check price condition
			if order.Type == models.OrderTypeLimit && order.Price < oppositeOrder.Price {
				break // Can't match at this price
			}

			// Determine execution price and quantity
			execPrice := oppositeOrder.Price
			if order.Type == models.OrderTypeMarket {
				// Market buy takes best ask price
				execPrice = oppositeOrder.Price
			} else if order.Price >= oppositeOrder.Price {
				// Limit buy can match if price >= ask
				execPrice = oppositeOrder.Price
			}

			execQuantity := min(remaining, oppositeOrder.Remaining)

			// Create trade
			trade := models.Trade{
				ID:          generateID(),
				Symbol:      ob.symbol,
				Price:       execPrice,
				Quantity:    execQuantity,
				BuyOrderID:  order.ID,
				SellOrderID: oppositeOrder.ID,
				Timestamp:   time.Now(),
			}
			trades = append(trades, trade)
			ob.trades = append(ob.trades, trade)

			// Update order quantities
			remaining -= execQuantity
			order.Remaining -= execQuantity
			oppositeOrder.Remaining -= execQuantity

			ob.lastPrice = execPrice

			// If opposite order is filled, remove from heap
			if oppositeOrder.Remaining == 0 {
				heap.Pop(&ob.asks.OrderHeap)
				delete(ob.orderMap, oppositeOrder.ID)
				oppositeOrder.Status = models.OrderStatusFilled
			} else {
				oppositeOrder.Status = models.OrderStatusPartially
			}

			// Update order status
			if order.Remaining == 0 {
				order.Status = models.OrderStatusFilled
				break
			} else {
				order.Status = models.OrderStatusPartially
			}
		}
	} else {
		// Match against bids (buy orders)
		for remaining > 0 && ob.bids.Len() > 0 {
			opposite := ob.bids.Peek()
			if opposite == nil {
				break
			}
			oppositeOrder := opposite.(*models.Order)

			// For limit orders, check price condition
			if order.Type == models.OrderTypeLimit && order.Price > oppositeOrder.Price {
				break
			}

			execPrice := oppositeOrder.Price
			if order.Type == models.OrderTypeMarket {
				execPrice = oppositeOrder.Price
			} else if order.Price <= oppositeOrder.Price {
				execPrice = oppositeOrder.Price
			}

			execQuantity := min(remaining, oppositeOrder.Remaining)

			trade := models.Trade{
				ID:          generateID(),
				Symbol:      ob.symbol,
				Price:       execPrice,
				Quantity:    execQuantity,
				BuyOrderID:  oppositeOrder.ID,
				SellOrderID: order.ID,
				Timestamp:   time.Now(),
			}
			trades = append(trades, trade)
			ob.trades = append(ob.trades, trade)

			remaining -= execQuantity
			order.Remaining -= execQuantity
			oppositeOrder.Remaining -= execQuantity

			ob.lastPrice = execPrice

			if oppositeOrder.Remaining == 0 {
				heap.Pop(&ob.bids.OrderHeap)
				delete(ob.orderMap, oppositeOrder.ID)
				oppositeOrder.Status = models.OrderStatusFilled
			} else {
				oppositeOrder.Status = models.OrderStatusPartially
			}

			if order.Remaining == 0 {
				order.Status = models.OrderStatusFilled
				break
			} else {
				order.Status = models.OrderStatusPartially
			}
		}
	}

	order.Remaining = remaining
	return trades
}

// CancelOrder removes an order from the order book
func (ob *OrderBook) CancelOrder(orderID string) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.orderMap[orderID]
	if !exists {
		return false
	}

	// Remove from heap
	if order.Side == models.OrderSideBuy {
		for i, o := range ob.bids.OrderHeap {
			if o.ID == orderID {
				heap.Remove(&ob.bids.OrderHeap, i)
				break
			}
		}
	} else {
		for i, o := range ob.asks.OrderHeap {
			if o.ID == orderID {
				heap.Remove(&ob.asks.OrderHeap, i)
				break
			}
		}
	}

	// Update order status
	order.Status = models.OrderStatusCancelled
	delete(ob.orderMap, orderID)
	return true
}

// GetDepth returns the current order book depth
func (ob *OrderBook) GetDepth(levels int) ([]models.Order, []models.Order) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	// Copy all bids and sort them (highest price first)
	bidsCount := ob.bids.Len()
	allBids := make([]models.Order, 0, bidsCount)
	for i := 0; i < bidsCount; i++ {
		allBids = append(allBids, *ob.bids.OrderHeap[i])
	}
	sort.Slice(allBids, func(i, j int) bool {
		if allBids[i].Price == allBids[j].Price {
			return allBids[i].CreatedAt.Before(allBids[j].CreatedAt)
		}
		return allBids[i].Price > allBids[j].Price
	})

	// Copy all asks and sort them (lowest price first)
	asksCount := ob.asks.Len()
	allAsks := make([]models.Order, 0, asksCount)
	for i := 0; i < asksCount; i++ {
		allAsks = append(allAsks, *ob.asks.OrderHeap[i])
	}
	sort.Slice(allAsks, func(i, j int) bool {
		if allAsks[i].Price == allAsks[j].Price {
			return allAsks[i].CreatedAt.Before(allAsks[j].CreatedAt)
		}
		return allAsks[i].Price < allAsks[j].Price
	})

	// Take only the requested number of levels
	limitBids := min(levels, len(allBids))
	limitAsks := min(levels, len(allAsks))

	return allBids[:limitBids], allAsks[:limitAsks]
}

// GetLastPrice returns the last traded price
func (ob *OrderBook) GetLastPrice() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return ob.lastPrice
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateID() string {
	return fmt.Sprintf("tr_%d", time.Now().UnixNano())
}
