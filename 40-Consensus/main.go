package main

import (
	"design-with-tsgo/pkg/logger"
	"fmt"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// RaftRole represents the role of a node in a Raft consensus cluster.
type RaftRole string

const (
	// Follower is the default state. Followers respond to RPCs from leaders and candidates.
	Follower RaftRole = "follower"
	// Candidate is the state during an election. A node becomes a candidate
	// when it times out waiting for a heartbeat from the leader.
	Candidate RaftRole = "candidate"
	// Leader is the state of the elected coordinator. The leader handles all
	// client requests and replicates log entries to followers.
	Leader RaftRole = "leader"
)

// LogEntry represents a single command in the Raft replicated log.
// Each entry is associated with the term in which it was created.
type LogEntry struct {
	Term    int
	Command string
}

// RaftNode represents a single node in a Raft consensus cluster.
// It tracks its role, current term, vote, log entries, and commit index.
type RaftNode struct {
	ID          int
	Role        RaftRole
	Term        int
	VotedFor    int
	Log         []LogEntry
	CommitIndex int
}

// RaftCluster manages a collection of Raft nodes and provides
// leader election and log replication operations.
type RaftCluster struct {
	nodes    map[int]*RaftNode
	leaderID int
}

// NewRaftCluster creates a cluster with the given node IDs, all starting
// as Followers. Time complexity: O(n) where n is the number of node IDs.
func NewRaftCluster(nodeIDs []int) *RaftCluster {
	defer log.Operation("NewRaftCluster", "Creating Raft cluster with %d nodes", len(nodeIDs))()
	c := &RaftCluster{nodes: make(map[int]*RaftNode), leaderID: -1}
	for _, id := range nodeIDs {
		c.nodes[id] = &RaftNode{ID: id, Role: Follower, VotedFor: -1, CommitIndex: -1}
		log.Debug("Node %d initialized as follower", id)
	}
	return c
}

// Elect triggers an election with the given candidate. The candidate
// increments its term, votes for itself, and requests votes from peers.
// A node grants its vote if the candidate's term >= the node's term
// and the node hasn't voted for someone else. Returns true if the
// candidate received a majority of votes and becomes leader.
// Time complexity: O(n) where n is the number of nodes.
func (c *RaftCluster) Elect(candidateID int) bool {
	defer log.Operation("Elect", "Node %d starting election", candidateID)()
	candidate, ok := c.nodes[candidateID]
	if !ok {
		log.Warn("Elect failed: node %d not found in cluster", candidateID)
		return false
	}
	candidate.Role = Candidate
	candidate.Term++
	candidate.VotedFor = candidate.ID
	votes := 1
	log.Debug("Candidate %d voted for self, term=%d", candidate.ID, candidate.Term)

	for _, peer := range c.nodes {
		if peer.ID == candidate.ID {
			continue
		}
		granted := false
		if candidate.Term >= peer.Term && (peer.VotedFor == -1 || peer.VotedFor == candidate.ID) {
			peer.Term = candidate.Term
			peer.VotedFor = candidate.ID
			granted = true
		}
		if granted {
			votes++
			log.Debug("Node %d voted for %d (term=%d)", peer.ID, candidate.ID, peer.Term)
		} else {
			log.Debug("Node %d denied vote to %d (peerTerm=%d candidateTerm=%d votedFor=%d)",
				peer.ID, candidate.ID, peer.Term, candidate.Term, peer.VotedFor)
		}
	}

	quorum := len(c.nodes)/2 + 1
	if votes > len(c.nodes)/2 {
		log.Info("Election won: %d votes (need %d) — node %d is now leader", votes, quorum, candidate.ID)
		c.leaderID = candidate.ID
		for _, node := range c.nodes {
			if node.ID == candidate.ID {
				node.Role = Leader
			} else {
				node.Role = Follower
			}
		}
		return true
	}
	log.Warn("Election lost: %d votes (need %d)", votes, quorum)
	return false
}

// Append creates a new log entry for the given command, replicates it
// to a majority of nodes, and advances the commit index if successful.
// Returns true if the entry was committed (replicated to a majority).
// Time complexity: O(n) where n is the number of nodes.
func (c *RaftCluster) Append(command string) bool {
	defer log.Operation("Append", "Appending command: %s", command)()
	if c.leaderID == -1 {
		log.Warn("Append failed: no leader elected")
		return false
	}
	leader := c.nodes[c.leaderID]
	entry := LogEntry{Term: leader.Term, Command: command}
	leader.Log = append(leader.Log, entry)
	replicated := 1
	log.Debug("Leader %d appended entry at index %d", leader.ID, len(leader.Log)-1)

	for _, peer := range c.nodes {
		if peer.ID == leader.ID {
			continue
		}
		peer.Log = append(peer.Log, entry)
		replicated++
		log.Debug("Replicated to node %d (log length=%d)", peer.ID, len(peer.Log))
	}

	quorum := len(c.nodes)/2 + 1
	if replicated > len(c.nodes)/2 {
		index := len(leader.Log) - 1
		for _, node := range c.nodes {
			node.CommitIndex = index
		}
		log.Info("Entry committed: index=%d replicated=%d/%d (quorum=%d)",
			index, replicated, len(c.nodes), quorum)
		return true
	}
	log.Warn("Entry not committed: replicated=%d/%d (need %d)", replicated, len(c.nodes), quorum)
	return false
}

// CommittedCommands returns the list of commands at or below the leader's
// commit index, in log order. Returns nil if no leader exists.
// Time complexity: O(c) where c is the commit index + 1.
func (c *RaftCluster) CommittedCommands() []string {
	if c.leaderID == -1 {
		log.Warn("No leader elected — no committed commands available")
		return nil
	}
	leader := c.nodes[c.leaderID]
	commands := make([]string, 0, leader.CommitIndex+1)
	for i := 0; i <= leader.CommitIndex && i < len(leader.Log); i++ {
		commands = append(commands, leader.Log[i].Command)
	}
	log.Debug("Retrieved %d committed commands", len(commands))
	return commands
}

func main() {
	defer log.Operation("main", "Running Raft Consensus demo")()

	logger.Section("RAFT CONSENSUS — Production Scenario")

	// Simulate a 5-node Raft cluster for a distributed configuration store
	log.Info("Bringing up 5-node Raft cluster")
	start := time.Now()
	cluster := NewRaftCluster([]int{1, 2, 3, 4, 5})
	log.Info("Cluster initialized in %v", time.Since(start))

	logger.Section("SCENARIO: Leader Election")
	log.Info("Node 3 starts an election (triggered by heartbeat timeout)")
	electionStart := time.Now()
	elected := cluster.Elect(3)
	log.Info("Election completed in %v", time.Since(electionStart))
	logger.KeyValue("elected", elected)
	logger.KeyValue("leader_id", cluster.leaderID)

	logger.Section("SCENARIO: Committed Log Entries (State Machine)")
	log.Info("Appending configuration commands to the replicated log")
	commands := []string{
		"set db.max_connections=200",
		"set cache.eviction_policy=LRU",
		"set feature.dark_mode=enabled",
		"set rate.limit=5000",
	}
	appendStart := time.Now()
	committedCount := 0
	for _, cmd := range commands {
		if cluster.Append(cmd) {
			committedCount++
		}
	}
	log.Info("Replication completed in %v", time.Since(appendStart))

	logger.Section("SCENARIO: Reading Committed State")
	committed := cluster.CommittedCommands()
	log.Info("Committed log entries (applied to state machine):")
	for i, cmd := range committed {
		logger.KeyValue(fmt.Sprintf("entry_%d", i), cmd)
	}

	logger.Section("SCENARIO: Failed Append Without Leader")
	// Create a new cluster without electing a leader
	noLeaderCluster := NewRaftCluster([]int{10, 11, 12})
	log.Info("Attempting append on leaderless cluster")
	if !noLeaderCluster.Append("set x=1") {
		log.Info("Correctly rejected append — no leader elected")
	}

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("cluster_nodes", len(cluster.nodes))
	logger.KeyValue("leader_id", cluster.leaderID)
	logger.KeyValue("current_term", cluster.nodes[cluster.leaderID].Term)
	logger.KeyValue("log_entries", len(cluster.nodes[cluster.leaderID].Log))
	logger.KeyValue("commit_index", cluster.nodes[cluster.leaderID].CommitIndex)
	logger.KeyValue("commands_committed", committedCount)
	logger.KeyValue("total_commands_sent", len(commands))
	nodesInRole := map[RaftRole]int{Follower: 0, Candidate: 0, Leader: 0}
	for _, node := range cluster.nodes {
		nodesInRole[node.Role]++
	}
	logger.KeyValue("followers", nodesInRole[Follower])
	logger.KeyValue("candidates", nodesInRole[Candidate])
	logger.KeyValue("leaders", nodesInRole[Leader])
	log.Info("Raft consensus demo completed successfully")
}
