package orderbook

import (
	"testing"
	"time"

	"stock-market/pkg/models"
)

func TestOrderBookBasic(t *testing.T) {
	ob := NewOrderBook("AAPL")
	
	// Create a buy order
	buyOrder := &models.Order{
		ID:        "buy1",
		UserID:    "user1",
		Symbol:    "AAPL",
		Side:      models.OrderSideBuy,
		Type:      models.OrderTypeLimit,
		Price:     150.0,
		Quantity:  100,
		Remaining: 100,
		Status:    models.OrderStatusPending,
		CreatedAt: time.Now(),
	}
	
	// Create a sell order
	sellOrder := &models.Order{
		ID:        "sell1",
		UserID:    "user2",
		Symbol:    "AAPL",
		Side:      models.OrderSideSell,
		Type:      models.OrderTypeLimit,
		Price:     149.0,
		Quantity:  50,
		Remaining: 50,
		Status:    models.OrderStatusPending,
		CreatedAt: time.Now(),
	}
	
	// Add sell order first (no match)
	trades := ob.AddOrder(sellOrder)
	if len(trades) != 0 {
		t.Errorf("Expected no trades initially, got %d", len(trades))
	}
	
	// Add buy order (should match with sell)
	trades = ob.AddOrder(buyOrder)
	if len(trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(trades))
	}
	
	if trades[0].Price != 149.0 {
		t.Errorf("Expected price 149.0, got %f", trades[0].Price)
	}
	
	if trades[0].Quantity != 50 {
		t.Errorf("Expected quantity 50, got %d", trades[0].Quantity)
	}
	
	// Check order statuses
	if buyOrder.Remaining != 50 {
		t.Errorf("Buy order should have 50 remaining, got %d", buyOrder.Remaining)
	}
	
	if sellOrder.Remaining != 0 {
		t.Errorf("Sell order should have 0 remaining, got %d", sellOrder.Remaining)
	}
	
	if sellOrder.Status != models.OrderStatusFilled {
		t.Errorf("Sell order should be filled, got %s", sellOrder.Status)
	}
}

func TestOrderBookCancel(t *testing.T) {
	ob := NewOrderBook("AAPL")
	
	order := &models.Order{
		ID:        "order1",
		UserID:    "user1",
		Symbol:    "AAPL",
		Side:      models.OrderSideBuy,
		Type:      models.OrderTypeLimit,
		Price:     150.0,
		Quantity:  100,
		Remaining: 100,
		Status:    models.OrderStatusPending,
		CreatedAt: time.Now(),
	}
	
	// Add order
	ob.AddOrder(order)
	
	// Cancel order
	success := ob.CancelOrder("order1")
	if !success {
		t.Error("Expected cancel to succeed")
	}
	
	if order.Status != models.OrderStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", order.Status)
	}
}

func TestOrderBookDepth(t *testing.T) {
	ob := NewOrderBook("AAPL")
	
	// Add multiple orders
	for i := 1; i <= 5; i++ {
		order := &models.Order{
			ID:        string(rune('a' + i - 1)),
			UserID:    "user1",
			Symbol:    "AAPL",
			Side:      models.OrderSideBuy,
			Type:      models.OrderTypeLimit,
			Price:     float64(150 + i),
			Quantity:  100,
			Remaining: 100,
			Status:    models.OrderStatusPending,
			CreatedAt: time.Now(),
		}
		ob.AddOrder(order)
	}
	
	// Get depth (should return highest prices first)
	bids, _ := ob.GetDepth(3)
	if len(bids) != 3 {
		t.Errorf("Expected 3 bids, got %d", len(bids))
	}
	
	// The first element should be the highest price (max heap)
	if bids[0].Price != 155.0 {
		t.Errorf("First bid should be highest price (155), got %f", bids[0].Price)
	}
}