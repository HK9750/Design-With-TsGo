package main

import "fmt"

type BTreeNode struct {
	keys     []int
	children []*BTreeNode
	leaf     bool
}

type BTree struct {
	root      *BTreeNode
	minDegree int
}

func NewBTree(minDegree int) *BTree {
	if minDegree < 2 {
		panic("minDegree must be at least 2")
	}
	return &BTree{root: &BTreeNode{leaf: true}, minDegree: minDegree}
}

func (t *BTree) Search(key int) bool { return searchBTree(t.root, key) }

func searchBTree(node *BTreeNode, key int) bool {
	i := 0
	for i < len(node.keys) && key > node.keys[i] {
		i++
	}
	if i < len(node.keys) && key == node.keys[i] {
		return true
	}
	if node.leaf {
		return false
	}
	return searchBTree(node.children[i], key)
}

func (t *BTree) Insert(key int) {
	root := t.root
	if len(root.keys) == 2*t.minDegree-1 {
		next := &BTreeNode{leaf: false, children: []*BTreeNode{root}}
		t.splitChild(next, 0)
		t.root = next
	}
	t.insertNonFull(t.root, key)
}

func (t *BTree) insertNonFull(node *BTreeNode, key int) {
	i := len(node.keys) - 1
	if node.leaf {
		for i >= 0 && key < node.keys[i] {
			i--
		}
		if i >= 0 && node.keys[i] == key {
			return
		}
		node.keys = append(node.keys, 0)
		copy(node.keys[i+2:], node.keys[i+1:])
		node.keys[i+1] = key
		return
	}
	for i >= 0 && key < node.keys[i] {
		i--
	}
	i++
	if len(node.children[i].keys) == 2*t.minDegree-1 {
		t.splitChild(node, i)
		if key > node.keys[i] {
			i++
		}
	}
	t.insertNonFull(node.children[i], key)
}

func (t *BTree) splitChild(parent *BTreeNode, index int) {
	full := parent.children[index]
	sibling := &BTreeNode{leaf: full.leaf}
	middle := full.keys[t.minDegree-1]
	sibling.keys = append(sibling.keys, full.keys[t.minDegree:]...)
	full.keys = full.keys[:t.minDegree-1]
	if !full.leaf {
		sibling.children = append(sibling.children, full.children[t.minDegree:]...)
		full.children = full.children[:t.minDegree]
	}
	parent.keys = append(parent.keys, 0)
	copy(parent.keys[index+1:], parent.keys[index:])
	parent.keys[index] = middle
	parent.children = append(parent.children, nil)
	copy(parent.children[index+2:], parent.children[index+1:])
	parent.children[index+1] = sibling
}

func main() {
	tree := NewBTree(2)
	for _, value := range []int{10, 20, 5, 6, 12, 30, 7, 17} {
		tree.Insert(value)
	}
	fmt.Println(tree.Search(12), tree.Search(99))
}
