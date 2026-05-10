package main

import (
	"fmt"
	"math/rand"
)

type SkipNode[V any] struct {
	key     int
	value   V
	forward []*SkipNode[V]
}

type SkipList[V any] struct {
	head        *SkipNode[V]
	level       int
	maxLevel    int
	probability float64
}

func NewSkipList[V any](maxLevel int, probability float64) *SkipList[V] {
	if maxLevel <= 0 {
		maxLevel = 16
	}
	if probability <= 0 || probability >= 1 {
		probability = 0.5
	}
	return &SkipList[V]{head: &SkipNode[V]{key: -1 << 62, forward: make([]*SkipNode[V], maxLevel+1)}, maxLevel: maxLevel, probability: probability}
}

func (sl *SkipList[V]) Search(key int) (V, bool) {
	var zero V
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
	}
	node = node.forward[0]
	if node != nil && node.key == key {
		return node.value, true
	}
	return zero, false
}

func (sl *SkipList[V]) Insert(key int, value V) {
	update := make([]*SkipNode[V], sl.maxLevel+1)
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
		update[i] = node
	}
	if existing := node.forward[0]; existing != nil && existing.key == key {
		existing.value = value
		return
	}

	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level + 1; i <= newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}
	created := &SkipNode[V]{key: key, value: value, forward: make([]*SkipNode[V], newLevel+1)}
	for i := 0; i <= newLevel; i++ {
		created.forward[i] = update[i].forward[i]
		update[i].forward[i] = created
	}
}

func (sl *SkipList[V]) Delete(key int) bool {
	update := make([]*SkipNode[V], sl.maxLevel+1)
	node := sl.head
	for i := sl.level; i >= 0; i-- {
		for node.forward[i] != nil && node.forward[i].key < key {
			node = node.forward[i]
		}
		update[i] = node
	}
	target := node.forward[0]
	if target == nil || target.key != key {
		return false
	}
	for i := 0; i <= sl.level && update[i].forward[i] == target; i++ {
		if i < len(target.forward) {
			update[i].forward[i] = target.forward[i]
		} else {
			update[i].forward[i] = nil
		}
	}
	for sl.level > 0 && sl.head.forward[sl.level] == nil {
		sl.level--
	}
	return true
}

func (sl *SkipList[V]) randomLevel() int {
	level := 0
	for level < sl.maxLevel && rand.Float64() < sl.probability {
		level++
	}
	return level
}

func main() {
	list := NewSkipList[string](16, 0.5)
	list.Insert(3, "three")
	list.Insert(1, "one")
	list.Insert(2, "two")
	value, ok := list.Search(2)
	fmt.Println(value, ok)
}
