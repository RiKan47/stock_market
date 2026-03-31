package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"stock-market/internal/cache"
	"stock-market/internal/market"

	"github.com/gorilla/websocket"
)

var (
	hub      *market.Hub
	lru      *cache.LRUCache
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	mu sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	hub = market.NewHub()
	go hub.Run()

	// LRU cache for trade history (100 trades)
	lru = cache.NewLRUCache(100)

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/api/market/history", handleHistory)
	http.HandleFunc("/api/market/broadcast", handleBroadcast) // Internal endpoint to receive updates
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	log.Printf("Market Data Service listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		return
	}

	client := &market.Client{
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	hub.Register <- client

	// Start reading from client in a goroutine
	go func() {
		defer func() {
			hub.Unregister <- client
			client.Conn.Close()
		}()
		for {
			_, _, err := client.Conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	// Write messages to client
	go func() {
		for message := range client.Send {
			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				break
			}
		}
	}()
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	// Return last 100 trades from LRU
	keys := lru.Keys()
	history := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		if val, ok := lru.Get(key); ok {
			history = append(history, val)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	// Receive trade update from matching engine
	var payload struct {
		Type   string      `json:"type"`
		Symbol string      `json:"symbol"`
		Data   interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update LRU if it's a trade
	if payload.Type == "trade" {
		mu.Lock()
		tradeID := fmt.Sprintf("t_%d", lru.Size()) // Simplified key
		lru.Put(tradeID, payload.Data)
		mu.Unlock()
	}

	// Broadcast to all websocket clients
	message, _ := json.Marshal(payload)
	hub.Broadcast <- message

	w.WriteHeader(http.StatusOK)
}
