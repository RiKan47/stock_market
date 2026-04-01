package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"stock-market/internal/orderbook"
	"stock-market/pkg/models"

	"github.com/google/uuid"
)

type Engine struct {
	OrderBooks map[string]*orderbook.OrderBook
	PaxosNodes []string // List of paxos node URLs
	MarketAddr string   // Market data service URL
	mu         sync.RWMutex
}

func NewEngine(paxosNodes []string, marketAddr string) *Engine {
	return &Engine{
		OrderBooks: make(map[string]*orderbook.OrderBook),
		PaxosNodes: paxosNodes,
		MarketAddr: marketAddr,
	}
}

func (e *Engine) GetOrderBook(symbol string) *orderbook.OrderBook {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ob, ok := e.OrderBooks[symbol]; ok {
		return ob
	}
	ob := orderbook.NewOrderBook(symbol)
	e.OrderBooks[symbol] = ob
	return ob
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	paxosNodesStr := os.Getenv("PAXOS_NODES") // Comma-separated list of host:port
	var paxosNodes []string
	if paxosNodesStr != "" {
		paxosNodes = strings.Split(paxosNodesStr, ",")
	} else {
		// Fallback for local testing
		paxosNodes = []string{"localhost:9001", "localhost:9002", "localhost:9003"}
	}

	marketAddr := os.Getenv("MARKET_ADDR")
	if marketAddr == "" {
		marketAddr = "http://localhost:8083"
	}

	engine := NewEngine(paxosNodes, marketAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/orders", engine.handleOrder)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	corsMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	log.Printf("Matching Engine listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMux))
}

func (e *Engine) handleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Basic validation
	if order.ID == "" {
		order.ID = uuid.New().String()
	}
	if order.Symbol == "" || order.Quantity <= 0 {
		http.Error(w, "Invalid order symbol or quantity", http.StatusBadRequest)
		return
	}
	order.Remaining = order.Quantity
	order.Status = models.OrderStatusActive
	order.CreatedAt = time.Now()

	ob := e.GetOrderBook(order.Symbol)
	trades := ob.AddOrder(&order)

	// If matches found, try to reach consensus on the first trade (simplification)
	// In a real system, we'd batch these or run them in parallel
	for _, trade := range trades {
		log.Printf("Matching trade found: %v", trade)
		instanceID := uuid.New().String()

		// Propose to the first available Paxos node
		go e.proposeTrade(instanceID, trade)
	}

	// Broadcast order book depth update
	go e.broadcastDepth(ob)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (e *Engine) proposeTrade(instanceID string, trade models.Trade) {
	// Try proposing to any paxos node until consensus or exhaustion
	for _, node := range e.PaxosNodes {
		url := fmt.Sprintf("http://%s/paxos/propose", node)
		payload := struct {
			InstanceID string       `json:"instance_id"`
			Value      models.Trade `json:"value"`
		}{instanceID, trade}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("Consensus reached on trade %s via paxos node %s", trade.ID, node)
			// On success, broadcast to market data service
			e.broadcastTrade(trade)
			return
		}
		if err != nil {
			log.Printf("Error proposing to node %s: %v", node, err)
		} else {
			log.Printf("Node %s rejected proposal for trade %s (status %d)", node, trade.ID, resp.StatusCode)
		}
	}
	log.Printf("Failed to reach consensus on trade %s across all nodes", trade.ID)
}

func (e *Engine) broadcastTrade(trade models.Trade) {
	payload := struct {
		Type   string       `json:"type"`
		Symbol string       `json:"symbol"`
		Data   models.Trade `json:"data"`
	}{"trade", trade.Symbol, trade}

	body, _ := json.Marshal(payload)
	http.Post(e.MarketAddr+"/api/market/broadcast", "application/json", bytes.NewBuffer(body))
}

func (e *Engine) broadcastDepth(ob *orderbook.OrderBook) {
	bids, asks := ob.GetDepth(10)
	depth := struct {
		Bids []models.Order `json:"bids"`
		Asks []models.Order `json:"asks"`
	}{bids, asks}

	payload := struct {
		Type   string      `json:"type"`
		Symbol string      `json:"symbol"`
		Data   interface{} `json:"data"`
	}{"depth", bids[0].Symbol, depth} // Assuming at least one bid/ask exists for symbol

	if len(bids) == 0 && len(asks) == 0 {
		return
	}

	symbol := ""
	if len(bids) > 0 {
		symbol = bids[0].Symbol
	} else {
		symbol = asks[0].Symbol
	}
	payload.Symbol = symbol

	body, _ := json.Marshal(payload)
	http.Post(e.MarketAddr+"/api/market/broadcast", "application/json", bytes.NewBuffer(body))
}
