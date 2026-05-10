package main

import "fmt"

type GCounter struct{ counts map[string]int }

func NewGCounter() *GCounter { return &GCounter{counts: make(map[string]int)} }

func (c *GCounter) Increment(node string, amount int) {
	if amount < 0 {
		panic("GCounter cannot decrement")
	}
	c.counts[node] += amount
}

func (c *GCounter) Merge(other *GCounter) {
	for node, count := range other.counts {
		if count > c.counts[node] {
			c.counts[node] = count
		}
	}
}

func (c *GCounter) Value() int {
	total := 0
	for _, count := range c.counts {
		total += count
	}
	return total
}

type PNCounter struct {
	positive *GCounter
	negative *GCounter
}

func NewPNCounter() *PNCounter                         { return &PNCounter{positive: NewGCounter(), negative: NewGCounter()} }
func (c *PNCounter) Increment(node string, amount int) { c.positive.Increment(node, amount) }
func (c *PNCounter) Decrement(node string, amount int) { c.negative.Increment(node, amount) }
func (c *PNCounter) Merge(other *PNCounter) {
	c.positive.Merge(other.positive)
	c.negative.Merge(other.negative)
}
func (c *PNCounter) Value() int { return c.positive.Value() - c.negative.Value() }

func main() {
	a := NewPNCounter()
	b := NewPNCounter()
	a.Increment("a", 3)
	b.Decrement("b", 1)
	a.Merge(b)
	fmt.Println(a.Value())
}
