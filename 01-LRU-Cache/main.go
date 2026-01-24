package main

import "fmt"

type ListNode struct {
	Key   int
	Value int
	Next  *ListNode
	Prev  *ListNode
}

type LRUCache struct {
	Size     int
	Capacity int
	Map      map[int]*ListNode
	Head     *ListNode
	Tail     *ListNode
}

func NewLRUCache(capacity int) *LRUCache {
	head := &ListNode{}
	tail := &ListNode{}
	head.Next = tail
	tail.Prev = head

	return &LRUCache{
		Size:     0,
		Capacity: capacity,
		Map:      make(map[int]*ListNode),
		Head:     head,
		Tail:     tail,
	}
}

func (l *LRUCache) get(key int) int {
	node, ok := l.Map[key]
	if !ok {
		return -1
	}
	l.deleteNode(node)
	l.addNode(node)
	return node.Value
}

func (l *LRUCache) put(key int, value int) {
	if existing, ok := l.Map[key]; ok {
		existing.Value = value
		l.deleteNode(existing)
		l.addNode(existing)
		return
	}

	if l.Size == l.Capacity {
		lru := l.Head.Next
		l.deleteNode(lru)
		delete(l.Map, lru.Key)
		l.Size--
	}

	newNode := &ListNode{Key: key, Value: value}
	l.addNode(newNode)
	l.Map[key] = newNode
	l.Size++
}

func (l *LRUCache) addNode(node *ListNode) {
	node.Prev = l.Tail.Prev
	node.Next = l.Tail
	l.Tail.Prev.Next = node
	l.Tail.Prev = node
}

func (l *LRUCache) deleteNode(node *ListNode) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
}

func main() {
	cache := NewLRUCache(2)
	fmt.Println("LRU Cache initialized with capacity:", cache.Capacity)

	cache.put(1, 100)
	cache.put(2, 200)
	fmt.Println("Get key 1:", cache.get(1)) // 100

	cache.put(3, 300)                       // evicts key 2
	fmt.Println("Get key 2:", cache.get(2)) // -1 (evicted)
	fmt.Println("Get key 3:", cache.get(3)) // 300

	cache.put(4, 400)                       // evicts key 1
	fmt.Println("Get key 1:", cache.get(1)) // -1 (evicted)
	fmt.Println("Get key 3:", cache.get(3)) // 300
	fmt.Println("Get key 4:", cache.get(4)) // 400
}
