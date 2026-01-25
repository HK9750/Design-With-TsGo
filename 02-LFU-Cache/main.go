package main

type ListNode struct {
	Key   int
	Value int
	Freq  int
	Prev  *ListNode
	Next  *ListNode
}

type DoublyLinkedList struct {
	Size int
	Head *ListNode
	Tail *ListNode
}

func NewDoublyLinkedList() *DoublyLinkedList {
	head := &ListNode{}
	tail := &ListNode{}
	head.Next = tail
	tail.Prev = head
	return &DoublyLinkedList{
		Size: 0,
		Head: head,
		Tail: tail,
	}
}

func (dl *DoublyLinkedList) addNode(node *ListNode) {
	node.Prev = dl.Head
	node.Next = dl.Head.Next
	dl.Head.Next.Prev = node
	dl.Head.Next = node
	dl.Size++
}

func (dl *DoublyLinkedList) remove(node *ListNode) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
	dl.Size--
}

func (dl *DoublyLinkedList) removeLast() *ListNode {
	if dl.Size == 0 {
		return nil
	}
	last := dl.Tail.Prev
	dl.remove(last)
	return last
}

type LFUCache struct {
	Capacity   int
	Size       int
	MinFreq    int
	KeyToNodes map[int]*ListNode
	FreqToList map[int]*DoublyLinkedList
}

func NewLFUCache(capacity int) *LFUCache {
	return &LFUCache{
		Capacity:   capacity,
		Size:       0,
		MinFreq:    0,
		KeyToNodes: make(map[int]*ListNode),
		FreqToList: make(map[int]*DoublyLinkedList),
	}
}

func (lfu *LFUCache) get(key int) int {
	node, ok := lfu.KeyToNodes[key]
	if !ok {
		return -1
	}
	lfu.update(node)
	return node.Value
}

func (lfu *LFUCache) put(key int, value int) {
	if lfu.Capacity == 0 {
		return
	}

	node, ok := lfu.KeyToNodes[key]
	if ok {
		node.Value = value
		lfu.update(node)
		return
	}

	if lfu.Size == lfu.Capacity {
		list := lfu.FreqToList[lfu.MinFreq]
		removed := list.removeLast()
		delete(lfu.KeyToNodes, removed.Key)
		lfu.Size--
	}

	newNode := &ListNode{Key: key, Value: value, Freq: 1}
	list, ok := lfu.FreqToList[1]
	if !ok {
		list = NewDoublyLinkedList()
		lfu.FreqToList[1] = list
	}
	list.addNode(newNode)
	lfu.KeyToNodes[key] = newNode
	lfu.MinFreq = 1
	lfu.Size++
}

func (lfu *LFUCache) update(node *ListNode) {
	freq := node.Freq
	list := lfu.FreqToList[freq]
	list.remove(node)

	if freq == lfu.MinFreq && list.Size == 0 {
		lfu.MinFreq++
	}

	node.Freq++
	newList, ok := lfu.FreqToList[node.Freq]
	if !ok {
		newList = NewDoublyLinkedList()
		lfu.FreqToList[node.Freq] = newList
	}
	newList.addNode(node)
}

func main() {
	cache := NewLFUCache(3)

	cache.put(1, 10)
	cache.get(1)
}

// the whole idea is that we maintain a map with key and value is the pointer to the node..
// for frequency we maintain a map with the frequency and the doubly linked list.. we add it from the tail(most recently used side)
// and we delete it from the head side because its the side that is least recently used with a specific frequency......
