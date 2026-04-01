package models

import (
	"time"
)

// OrderSide is the direction of the order (buy or sell)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType determines the execution logic
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus tracks lifecycle of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusActive    OrderStatus = "active"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartially OrderStatus = "partially_filled"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order represents a stock trading order
type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Symbol    string      `json:"symbol"`
	Side      OrderSide   `json:"side"`
	Type      OrderType   `json:"type"`
	Price     float64     `json:"price"` // For limit orders
	Quantity  int         `json:"quantity"`
	Remaining int         `json:"remaining"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Trade represents an executed transaction between two orders
type Trade struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Price      float64   `json:"price"`
	Quantity   int       `json:"quantity"`
	BuyOrderID string    `json:"buy_order_id"`
	SellOrderID string   `json:"sell_order_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// User represents a trading platform user
type User struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Balance  float64 `json:"balance"` // Cash balance
}

// UserPortfolio holds a user's stock holdings
type UserPortfolio struct {
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Quantity int    `json:"quantity"`
}