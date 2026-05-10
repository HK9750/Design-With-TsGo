package main

import (
	"fmt"
	"sort"
)

type RingPoint struct {
	hash uint32
	node string
}

type ConsistentHashRing struct {
	replicas int
	ring     []RingPoint
}

func NewConsistentHashRing(replicas int) *ConsistentHashRing {
	if replicas <= 0 {
		replicas = 100
	}
	return &ConsistentHashRing{replicas: replicas}
}

func (r *ConsistentHashRing) AddNode(node string) {
	r.RemoveNode(node)
	for i := 0; i < r.replicas; i++ {
		r.ring = append(r.ring, RingPoint{hash: hash32(fmt.Sprintf("%s#%d", node, i)), node: node})
	}
	sort.Slice(r.ring, func(i, j int) bool { return r.ring[i].hash < r.ring[j].hash })
}

func (r *ConsistentHashRing) RemoveNode(node string) {
	kept := r.ring[:0]
	for _, point := range r.ring {
		if point.node != node {
			kept = append(kept, point)
		}
	}
	r.ring = kept
}

func (r *ConsistentHashRing) GetNode(key string) (string, bool) {
	if len(r.ring) == 0 {
		return "", false
	}
	hash := hash32(key)
	idx := sort.Search(len(r.ring), func(i int) bool { return r.ring[i].hash >= hash })
	return r.ring[idx%len(r.ring)].node, true
}

func hash32(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	ring := NewConsistentHashRing(50)
	ring.AddNode("node-a")
	ring.AddNode("node-b")
	node, _ := ring.GetNode("customer:42")
	fmt.Println(node)
}
