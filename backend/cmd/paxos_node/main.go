package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"stock-market/internal/paxos"
	"stock-market/pkg/models"
)

var node *paxos.Node

func main() {
	nodeIDStr := os.Getenv("NODE_ID")
	nodeID, _ := strconv.Atoi(nodeIDStr)
	port := os.Getenv("PORT")
	peerStr := os.Getenv("PEERS") // Format: "1=host1:port1,2=host2:port2"

	peerAddresses := make(map[int]string)
	if peerStr != "" {
		peers := strings.Split(peerStr, ",")
		for _, p := range peers {
			parts := strings.Split(p, "=")
			id, _ := strconv.Atoi(parts[0])
			peerAddresses[id] = parts[1]
		}
	}

	// Ensure our own address is in the map for proposals
	peerAddresses[nodeID] = "localhost:" + port

	node = paxos.NewNode(nodeID, peerAddresses)

	http.HandleFunc("/paxos/prepare", handlePrepare)
	http.HandleFunc("/paxos/accept", handleAccept)
	http.HandleFunc("/paxos/commit", handleCommit)
	http.HandleFunc("/paxos/propose", handlePropose)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	log.Printf("Paxos Node %d listening on port %s", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handlePrepare(w http.ResponseWriter, r *http.Request) {
	var req paxos.PrepareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := node.HandlePrepare(req)
	json.NewEncoder(w).Encode(resp)
}

func handleAccept(w http.ResponseWriter, r *http.Request) {
	var req paxos.AcceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := node.HandleAccept(req)
	json.NewEncoder(w).Encode(resp)
}

func handleCommit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceID string        `json:"instance_id"`
		Value      *models.Trade `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node.Commit(req.InstanceID, req.Value)
	w.WriteHeader(http.StatusOK)
}

func handlePropose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceID string        `json:"instance_id"`
		Value      *models.Trade `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	success := node.Propose(req.InstanceID, req.Value)
	if success {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Consensus reached")
	} else {
		http.Error(w, "Failed to reach consensus", http.StatusConflict)
	}
}
