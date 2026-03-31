package market

import (
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{
		Hub:  hub,
		Send: make(chan []byte, 10),
	}
	client2 := &Client{
		Hub:  hub,
		Send: make(chan []byte, 10),
	}

	hub.Register <- client1
	hub.Register <- client2

	// Wait for registration to complete (simulated)
	time.Sleep(10 * time.Millisecond)

	message := []byte("hello")
	hub.Broadcast <- message

	// Verify both clients received the message
	select {
	case msg := <-client1.Send:
		if string(msg) != "hello" {
			t.Errorf("Client 1: Expected 'hello', got %s", string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 1: Broadcast timed out")
	}

	select {
	case msg := <-client2.Send:
		if string(msg) != "hello" {
			t.Errorf("Client 2: Expected 'hello', got %s", string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Error("Client 2: Broadcast timed out")
	}
}
