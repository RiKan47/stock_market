package paxos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"stock-market/pkg/models"
	"sync"
)

// ProposalNumber uniquely identifies a proposal
type ProposalNumber struct {
	Round  int
	NodeID int
}

func (p ProposalNumber) GreaterThan(other ProposalNumber) bool {
	if p.Round != other.Round {
		return p.Round > other.Round
	}
	return p.NodeID > other.NodeID
}

// PaxosState tracks the state for a single consensus instance
type PaxosState struct {
	PromisedNumber ProposalNumber
	AcceptedNumber ProposalNumber
	AcceptedValue  *models.Trade
}

// Node represents a single participant in the Paxos cluster
type Node struct {
	ID     int
	Nodes  map[int]string // Map of NodeID to "host:port"
	Quorum int

	// Persistent state for different consensus instances (indexed by InstanceID)
	// In a real system, this would be backed by a disk-based log
	Instances map[string]*PaxosState
	Log       []*models.Trade // Committed trades

	mu sync.RWMutex
}

func NewNode(id int, nodeAddresses map[int]string) *Node {
	return &Node{
		ID:        id,
		Nodes:     nodeAddresses,
		Quorum:    (len(nodeAddresses) / 2) + 1,
		Instances: make(map[string]*PaxosState),
		Log:       make([]*models.Trade, 0),
	}
}

// Prepare handles Phase 1a: Proposer sends Prepare(n) to Acceptors
type PrepareRequest struct {
	InstanceID string
	Number     ProposalNumber
}

// Promise handles Phase 1b: Acceptor sends Promise(n, [n_accepted, v_accepted]) to Proposer
type PromiseResponse struct {
	InstanceID    string
	Promised      bool
	LastAcceptedN ProposalNumber
	LastAcceptedV *models.Trade
}

// HandlePrepare is called by an Acceptor when it receives a Prepare request
func (n *Node) HandlePrepare(req PrepareRequest) PromiseResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	state, ok := n.Instances[req.InstanceID]
	if !ok {
		state = &PaxosState{}
		n.Instances[req.InstanceID] = state
	}

	if req.Number.GreaterThan(state.PromisedNumber) {
		state.PromisedNumber = req.Number
		return PromiseResponse{
			InstanceID:    req.InstanceID,
			Promised:      true,
			LastAcceptedN: state.AcceptedNumber,
			LastAcceptedV: state.AcceptedValue,
		}
	}

	return PromiseResponse{
		InstanceID: req.InstanceID,
		Promised:   false,
	}
}

// AcceptRequest handles Phase 2a: Proposer sends Accept(n, v) to Acceptors
type AcceptRequest struct {
	InstanceID string
	Number     ProposalNumber
	Value      *models.Trade
}

// AcceptResponse handles Phase 2b: Acceptor sends Accepted(n, v) to Proposer/Learners
type AcceptResponse struct {
	InstanceID string
	Accepted   bool
}

// HandleAccept is called by an Acceptor when it receives an Accept request
func (n *Node) HandleAccept(req AcceptRequest) AcceptResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	state, ok := n.Instances[req.InstanceID]
	if !ok {
		state = &PaxosState{}
		n.Instances[req.InstanceID] = state
	}

	// Accept only if we haven't promised to a higher number
	if !state.PromisedNumber.GreaterThan(req.Number) {
		state.PromisedNumber = req.Number
		state.AcceptedNumber = req.Number
		state.AcceptedValue = req.Value
		return AcceptResponse{
			InstanceID: req.InstanceID,
			Accepted:   true,
		}
	}

	return AcceptResponse{
		InstanceID: req.InstanceID,
		Accepted:   false,
	}
}

// Commit handles Phase 3: The value is committed once a quorum of acceptors has accepted it
func (n *Node) Commit(instanceID string, value *models.Trade) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.Log = append(n.Log, value)
	delete(n.Instances, instanceID)
}

// Propose orchestrates the Paxos phases across the network
func (n *Node) Propose(instanceID string, value *models.Trade) bool {
	n.mu.Lock()
	state, ok := n.Instances[instanceID]
	if !ok {
		state = &PaxosState{}
		n.Instances[instanceID] = state
	}
	state.PromisedNumber.Round++
	state.PromisedNumber.NodeID = n.ID
	propNum := state.PromisedNumber
	n.mu.Unlock()

	// Phase 1: Prepare
	promises := 0
	var highestAcceptedVal *models.Trade
	var highestAcceptedNum ProposalNumber

	for _, addr := range n.Nodes {
		resp, err := n.sendPrepare(addr, PrepareRequest{instanceID, propNum})
		if err == nil && resp.Promised {
			promises++
			if resp.LastAcceptedV != nil && resp.LastAcceptedN.GreaterThan(highestAcceptedNum) {
				highestAcceptedNum = resp.LastAcceptedN
				highestAcceptedVal = resp.LastAcceptedV
			}
		}
	}

	if promises < n.Quorum {
		return false
	}

	valToPropose := value
	if highestAcceptedVal != nil {
		valToPropose = highestAcceptedVal
	}

	// Phase 2: Accept
	accepts := 0
	for _, addr := range n.Nodes {
		resp, err := n.sendAccept(addr, AcceptRequest{instanceID, propNum, valToPropose})
		if err == nil && resp.Accepted {
			accepts++
		}
	}

	if accepts < n.Quorum {
		return false
	}

	// Phase 3: Commit
	for _, addr := range n.Nodes {
		n.sendCommit(addr, instanceID, valToPropose)
	}

	return true
}

func (n *Node) sendPrepare(addr string, req PrepareRequest) (PromiseResponse, error) {
	url := fmt.Sprintf("http://%s/paxos/prepare", addr)
	body, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return PromiseResponse{}, err
	}
	defer resp.Body.Close()
	var promise PromiseResponse
	json.NewDecoder(resp.Body).Decode(&promise)
	return promise, nil
}

func (n *Node) sendAccept(addr string, req AcceptRequest) (AcceptResponse, error) {
	url := fmt.Sprintf("http://%s/paxos/accept", addr)
	body, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return AcceptResponse{}, err
	}
	defer resp.Body.Close()
	var accept AcceptResponse
	json.NewDecoder(resp.Body).Decode(&accept)
	return accept, nil
}

func (n *Node) sendCommit(addr string, instanceID string, val *models.Trade) {
	url := fmt.Sprintf("http://%s/paxos/commit", addr)
	req := struct {
		InstanceID string        `json:"instance_id"`
		Value      *models.Trade `json:"value"`
	}{instanceID, val}
	body, _ := json.Marshal(req)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}
