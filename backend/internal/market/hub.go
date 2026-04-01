package market

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected user on a WebSocket
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte
}

// Hub manages active clients and broadcasting
type Hub struct {
	// Registered clients
	clients map[*Client]bool
	// Inbound messages to broadcast to all clients
	Broadcast chan []byte
	// Register requests from clients
	Register chan *Client
	// Unregister requests from clients
	Unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// If client channel is full, drop the client
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}
