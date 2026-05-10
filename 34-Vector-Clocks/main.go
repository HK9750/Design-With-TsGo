package main

import "fmt"

type VectorClock struct{ clock map[string]int }

func NewVectorClock() *VectorClock { return &VectorClock{clock: make(map[string]int)} }

func (vc *VectorClock) Tick(node string) { vc.clock[node]++ }

func (vc *VectorClock) Merge(other *VectorClock) {
	for node, value := range other.clock {
		if value > vc.clock[node] {
			vc.clock[node] = value
		}
	}
}

func (vc *VectorClock) Compare(other *VectorClock) string {
	less, greater := false, false
	seen := make(map[string]bool)
	for node := range vc.clock {
		seen[node] = true
	}
	for node := range other.clock {
		seen[node] = true
	}
	for node := range seen {
		a, b := vc.clock[node], other.clock[node]
		if a < b {
			less = true
		}
		if a > b {
			greater = true
		}
	}
	if less && greater {
		return "concurrent"
	}
	if less {
		return "before"
	}
	if greater {
		return "after"
	}
	return "equal"
}

func main() {
	a := NewVectorClock()
	b := NewVectorClock()
	a.Tick("a")
	b.Tick("b")
	fmt.Println(a.Compare(b))
}
