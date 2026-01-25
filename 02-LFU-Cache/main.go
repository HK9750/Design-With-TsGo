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
	return &DoublyLinkedList{
		Size: 0,
		Head: &ListNode{},
		Tail: &ListNode{},
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
	last := dl.Tail.Prev
	dl.remove(last)
	return last
}

type LFUCache struct {
}

func (lfu *LFUCache) get() {}

func (lfu *LFUCache) put() {}

func (lfu *LFUCache) update() {}

func main() {}
