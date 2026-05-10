package main

import "fmt"

type GossipValue struct {
	Value   string
	Version int
}

type GossipNode struct {
	ID    string
	state map[string]GossipValue
}

func NewGossipNode(id string) *GossipNode {
	return &GossipNode{ID: id, state: make(map[string]GossipValue)}
}

func (n *GossipNode) Set(key, value string) {
	current := n.state[key]
	n.state[key] = GossipValue{Value: value, Version: current.Version + 1}
}

func (n *GossipNode) GossipTo(peer *GossipNode) {
	peer.merge(n.state)
	n.merge(peer.state)
}

func (n *GossipNode) Get(key string) (string, bool) {
	value, ok := n.state[key]
	return value.Value, ok
}

func (n *GossipNode) merge(remote map[string]GossipValue) {
	for key, incoming := range remote {
		if local, ok := n.state[key]; !ok || incoming.Version > local.Version {
			n.state[key] = incoming
		}
	}
}

func main() {
	a := NewGossipNode("a")
	b := NewGossipNode("b")
	a.Set("leader", "a")
	a.GossipTo(b)
	value, _ := b.Get("leader")
	fmt.Println(value)
}
