package main

import "fmt"

type RaftRole string

const (
	Follower  RaftRole = "follower"
	Candidate RaftRole = "candidate"
	Leader    RaftRole = "leader"
)

type LogEntry struct {
	Term    int
	Command string
}

type RaftNode struct {
	ID          int
	Role        RaftRole
	Term        int
	VotedFor    int
	Log         []LogEntry
	CommitIndex int
}

type RaftCluster struct {
	nodes    map[int]*RaftNode
	leaderID int
}

func NewRaftCluster(nodeIDs []int) *RaftCluster {
	c := &RaftCluster{nodes: make(map[int]*RaftNode), leaderID: -1}
	for _, id := range nodeIDs {
		c.nodes[id] = &RaftNode{ID: id, Role: Follower, VotedFor: -1, CommitIndex: -1}
	}
	return c
}

func (c *RaftCluster) Elect(candidateID int) bool {
	candidate := c.nodes[candidateID]
	candidate.Role = Candidate
	candidate.Term++
	candidate.VotedFor = candidate.ID
	votes := 1
	for _, peer := range c.nodes {
		if peer.ID == candidate.ID {
			continue
		}
		if candidate.Term >= peer.Term && (peer.VotedFor == -1 || peer.VotedFor == candidate.ID) {
			peer.Term = candidate.Term
			peer.VotedFor = candidate.ID
			votes++
		}
	}
	if votes > len(c.nodes)/2 {
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
	return false
}

func (c *RaftCluster) Append(command string) bool {
	if c.leaderID == -1 {
		return false
	}
	leader := c.nodes[c.leaderID]
	entry := LogEntry{Term: leader.Term, Command: command}
	leader.Log = append(leader.Log, entry)
	replicated := 1
	for _, peer := range c.nodes {
		if peer.ID == leader.ID {
			continue
		}
		peer.Log = append(peer.Log, entry)
		replicated++
	}
	if replicated > len(c.nodes)/2 {
		index := len(leader.Log) - 1
		for _, node := range c.nodes {
			node.CommitIndex = index
		}
		return true
	}
	return false
}

func (c *RaftCluster) CommittedCommands() []string {
	if c.leaderID == -1 {
		return nil
	}
	leader := c.nodes[c.leaderID]
	commands := make([]string, 0, leader.CommitIndex+1)
	for i := 0; i <= leader.CommitIndex && i < len(leader.Log); i++ {
		commands = append(commands, leader.Log[i].Command)
	}
	return commands
}

func main() {
	cluster := NewRaftCluster([]int{1, 2, 3})
	cluster.Elect(1)
	cluster.Append("set x=1")
	fmt.Println(cluster.CommittedCommands())
}
