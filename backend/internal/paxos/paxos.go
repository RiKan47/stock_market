package paxos

import (
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
	Nodes  []int // IDs of all nodes in cluster
	Quorum int

	// Persistent state for different consensus instances (indexed by InstanceID)
	// In a real system, this would be backed by a disk-based log
	Instances map[string]*PaxosState
	Log       []*models.Trade // Committed trades

	mu sync.RWMutex
}

func NewNode(id int, nodeIDs []int) *Node {
	return &Node{
		ID:        id,
		Nodes:     nodeIDs,
		Quorum:    (len(nodeIDs) / 2) + 1,
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

	// In a real Paxos, we would ensure gaps in the log are filled
	// Here we just append the committed trade to the local log
	n.Log = append(n.Log, value)
	// Clean up instance state once committed
	delete(n.Instances, instanceID)
}
