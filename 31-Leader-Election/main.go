package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"sort"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// BullyElectionCluster implements the Bully Algorithm for leader election.
// The node with the highest ID among alive nodes becomes the leader.
// Time complexity of Elect() is O(n log n) due to sorting.
type BullyElectionCluster struct {
	alive  map[int]bool
	roles  map[int]string
	leader int
}

// NewBullyElectionCluster creates a cluster with the given node IDs and
// immediately runs an election to pick the initial leader.
// Time complexity: O(n log n) where n is the number of nodes.
func NewBullyElectionCluster(nodeIDs []int) *BullyElectionCluster {
	defer log.Operation("NewBullyElectionCluster", "Creating cluster with %d nodes", len(nodeIDs))()
	c := &BullyElectionCluster{alive: make(map[int]bool), roles: make(map[int]string), leader: -1}
	for _, id := range nodeIDs {
		c.alive[id] = true
		c.roles[id] = "follower"
		log.Debug("Initialized node %d as follower", id)
	}
	c.Elect()
	log.Info("Initial leader elected: %d", c.leader)
	return c
}

// Fail marks a node as dead. If the failed node was the leader, a new
// election is triggered. Time complexity: O(n log n) if election needed.
func (c *BullyElectionCluster) Fail(nodeID int) {
	defer log.Operation("Fail", "Node %d failing", nodeID)()
	if _, ok := c.alive[nodeID]; !ok {
		log.Warn("Node %d was already dead", nodeID)
		return
	}
	delete(c.alive, nodeID)
	log.Debug("Node %d removed from alive set (remaining: %d)", nodeID, len(c.alive))
	if c.leader == nodeID {
		log.Warn("Leader %d has failed — triggering new election", nodeID)
		c.Elect()
	} else {
		log.Debug("Node %d was not leader, no election needed", nodeID)
	}
}

// Recover brings a previously dead node back and triggers an election
// since the recovered node may have a higher ID than the current leader.
// Time complexity: O(n log n).
func (c *BullyElectionCluster) Recover(nodeID int) {
	defer log.Operation("Recover", "Node %d recovering", nodeID)()
	c.alive[nodeID] = true
	log.Debug("Node %d marked as alive", nodeID)
	c.Elect()
}

// Elect runs the Bully election: the alive node with the highest ID
// becomes the leader. Returns the leader's ID, or -1 if no nodes are alive.
// Time complexity: O(n log n) due to sorting.
func (c *BullyElectionCluster) Elect() int {
	defer log.Operation("Elect", "Running bully election")()
	ids := make([]int, 0, len(c.alive))
	for id := range c.alive {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	c.leader = -1
	if len(ids) > 0 {
		c.leader = ids[len(ids)-1]
		log.Debug("Candidates sorted: %v, winner: %d", ids, c.leader)
	} else {
		log.Warn("No alive nodes to elect leader")
	}
	for id := range c.roles {
		if id == c.leader {
			c.roles[id] = "leader"
		} else {
			c.roles[id] = "follower"
		}
	}
	log.Info("Election complete — leader: %d", c.leader)
	return c.leader
}

func main() {
	defer log.Operation("main", "Running Bully Leader Election demo")()

	logger.Section("BULLY LEADER ELECTION — Production Scenario")

	// Simulate a 3-node cluster coming online for a distributed cache service
	log.Info("Bringing up cache cluster nodes [1, 2, 3]")
	start := time.Now()
	cluster := NewBullyElectionCluster([]int{1, 2, 3})
	log.Info("Cluster initialization completed in %v", time.Since(start))

	logger.Section("SCENARIO: Leader Node Fails")
	logger.KeyValue("current_leader", cluster.leader)
	log.Info("Simulating crash of leader node %d (e.g., OOM kill)", cluster.leader)
	cluster.Fail(cluster.leader)
	logger.KeyValue("new_leader", cluster.leader)

	logger.Section("SCENARIO: Follower Node Fails")
	log.Info("Simulating crash of a follower node 1 (non-leader)")
	cluster.Fail(1)
	logger.KeyValue("current_leader", cluster.leader)

	logger.Section("SCENARIO: Node Recovery Joins Cluster")
	log.Info("Node 1 recovers and rejoins the cluster")
	cluster.Recover(1)
	logger.KeyValue("current_leader", cluster.leader)
	log.Info("Node 1 has higher ID than current leader — triggers re-election")

	logger.Section("SCENARIO: Original Leader Recovers")
	log.Info("Original leader node 3 recovers and reclaims leadership")
	cluster.Recover(3)
	logger.KeyValue("current_leader", cluster.leader)

	// Final stats summary
	logger.Section("FINAL STATS")
	aliveCount := len(cluster.alive)
	logger.KeyValue("alive_nodes", aliveCount)
	logger.KeyValue("leader_id", cluster.leader)
	leaderRoleCount := 0
	followerRoleCount := 0
	for id, role := range cluster.roles {
		logger.KeyValue("node_"+string(rune('0'+id)), role)
		if role == "leader" {
			leaderRoleCount++
		} else {
			followerRoleCount++
		}
	}
	logger.KeyValue("leader_count", leaderRoleCount)
	logger.KeyValue("follower_count", followerRoleCount)
	log.Info("Bully election demo completed successfully")
}
