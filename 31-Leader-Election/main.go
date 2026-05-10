package main

import (
	"fmt"
	"sort"
)

type BullyElectionCluster struct {
	alive  map[int]bool
	roles  map[int]string
	leader int
}

func NewBullyElectionCluster(nodeIDs []int) *BullyElectionCluster {
	c := &BullyElectionCluster{alive: make(map[int]bool), roles: make(map[int]string), leader: -1}
	for _, id := range nodeIDs {
		c.alive[id] = true
		c.roles[id] = "follower"
	}
	c.Elect()
	return c
}

func (c *BullyElectionCluster) Fail(nodeID int) {
	delete(c.alive, nodeID)
	if c.leader == nodeID {
		c.Elect()
	}
}

func (c *BullyElectionCluster) Recover(nodeID int) {
	c.alive[nodeID] = true
	c.Elect()
}

func (c *BullyElectionCluster) Elect() int {
	ids := make([]int, 0, len(c.alive))
	for id := range c.alive {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	c.leader = -1
	if len(ids) > 0 {
		c.leader = ids[len(ids)-1]
	}
	for id := range c.roles {
		if id == c.leader {
			c.roles[id] = "leader"
		} else {
			c.roles[id] = "follower"
		}
	}
	return c.leader
}

func main() {
	cluster := NewBullyElectionCluster([]int{1, 2, 3})
	cluster.Fail(3)
	fmt.Println(cluster.leader)
}
