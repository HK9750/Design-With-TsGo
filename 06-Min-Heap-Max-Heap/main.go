package main

import "fmt"

type BinaryHeap[T any] struct {
	data           []T
	higherPriority func(a, b T) bool
}

func NewBinaryHeap[T any](higherPriority func(a, b T) bool, values ...T) *BinaryHeap[T] {
	heap := &BinaryHeap[T]{data: append([]T(nil), values...), higherPriority: higherPriority}
	for i := len(heap.data)/2 - 1; i >= 0; i-- {
		heap.siftDown(i)
	}
	return heap
}

func (h *BinaryHeap[T]) Size() int { return len(h.data) }

func (h *BinaryHeap[T]) Peek() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		return zero, false
	}
	return h.data[0], true
}

func (h *BinaryHeap[T]) Insert(value T) {
	h.data = append(h.data, value)
	h.siftUp(len(h.data) - 1)
}

func (h *BinaryHeap[T]) Extract() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		return zero, false
	}
	top := h.data[0]
	last := h.data[len(h.data)-1]
	h.data = h.data[:len(h.data)-1]
	if len(h.data) > 0 {
		h.data[0] = last
		h.siftDown(0)
	}
	return top, true
}

func (h *BinaryHeap[T]) siftUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if !h.higherPriority(h.data[index], h.data[parent]) {
			return
		}
		h.data[index], h.data[parent] = h.data[parent], h.data[index]
		index = parent
	}
}

func (h *BinaryHeap[T]) siftDown(index int) {
	for {
		left, right, best := index*2+1, index*2+2, index
		if left < len(h.data) && h.higherPriority(h.data[left], h.data[best]) {
			best = left
		}
		if right < len(h.data) && h.higherPriority(h.data[right], h.data[best]) {
			best = right
		}
		if best == index {
			return
		}
		h.data[index], h.data[best] = h.data[best], h.data[index]
		index = best
	}
}

func main() {
	heap := NewBinaryHeap(func(a, b int) bool { return a < b }, 5, 1, 3)
	heap.Insert(2)
	first, _ := heap.Extract()
	second, _ := heap.Extract()
	fmt.Println(first, second)
}
