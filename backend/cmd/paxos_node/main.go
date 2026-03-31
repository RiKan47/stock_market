package main

import (
	"fmt"
	"os"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	port := os.Getenv("PORT")
	fmt.Printf("Paxos Node %s Started on port %s\n", nodeID, port)
	select {}
}