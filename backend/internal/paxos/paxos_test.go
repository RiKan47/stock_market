package paxos

import (
	"stock-market/pkg/models"
	"testing"
	"time"
)

func TestPaxosConsensus(t *testing.T) {
	// Setup 3 nodes
	nodeIDs := []int{1, 2, 3}
	nodes := make(map[int]*Node)
	for _, id := range nodeIDs {
		nodes[id] = NewNode(id, nodeIDs)
	}

	trade := &models.Trade{
		ID:        "trade1",
		Symbol:    "AAPL",
		Price:     150.0,
		Quantity:  10,
		Timestamp: time.Now(),
	}

	instanceID := "inst1"
	propNum := ProposalNumber{Round: 1, NodeID: 1}

	// Phase 1: Prepare
	promises := 0
	for _, node := range nodes {
		resp := node.HandlePrepare(PrepareRequest{
			InstanceID: instanceID,
			Number:     propNum,
		})
		if resp.Promised {
			promises++
		}
	}

	// Majority check (Quorum = 2)
	if promises < 2 {
		t.Fatalf("Failed to get quorum for Prepare: got %d promises", promises)
	}

	// Phase 2: Accept
	accepts := 0
	for _, node := range nodes {
		resp := node.HandleAccept(AcceptRequest{
			InstanceID: instanceID,
			Number:     propNum,
			Value:      trade,
		})
		if resp.Accepted {
			accepts++
		}
	}

	if accepts < 2 {
		t.Fatalf("Failed to get quorum for Accept: got %d accepts", accepts)
	}

	// Phase 3: Commit
	for _, node := range nodes {
		node.Commit(instanceID, trade)
	}

	// Verify log on each node
	for id, node := range nodes {
		if len(node.Log) != 1 {
			t.Errorf("Node %d: Expected log size 1, got %d", id, len(node.Log))
		} else if node.Log[0].ID != trade.ID {
			t.Errorf("Node %d: Log content mismatch. Expected trade ID %s, got %s", id, trade.ID, node.Log[0].ID)
		}
	}
}
