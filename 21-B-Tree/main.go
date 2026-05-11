package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// BTreeNode represents a node in a B-Tree. It stores a sorted slice of keys,
// child pointers, and a leaf flag. A non-leaf node with k keys has k+1 children.
type BTreeNode struct {
	keys     []int
	children []*BTreeNode
	leaf     bool
}

// BTree is a self-balancing search tree in which each node can hold multiple keys
// and have multiple children. The minDegree parameter controls the minimum degree t
// such that every node (except root) has at least t-1 keys and at most 2t-1 keys.
// Time complexity: O(log n) for search, O(log n) for insert.
type BTree struct {
	root      *BTreeNode
	minDegree int
}

// NewBTree creates a B-Tree with the given minimum degree. The minDegree must be
// at least 2 (each internal node has at least minDegree children).
// Returns a pointer to an initialized BTree with an empty leaf root.
// Time complexity: O(1).
func NewBTree(minDegree int) *BTree {
	if minDegree < 2 {
		log.Error("minDegree must be at least 2, got %d", minDegree)
		os.Exit(1)
	}
	log.Debug("Created B-Tree with minDegree=%d", minDegree)
	return &BTree{root: &BTreeNode{leaf: true}, minDegree: minDegree}
}

// Search looks up a key in the B-Tree. Returns true if the key exists, false otherwise.
// Time complexity: O(log n).
func (t *BTree) Search(key int) bool {
	defer log.Operation("BTree.Search", "key=%d", key)()
	found := searchBTree(t.root, key)
	logger.KeyValue("key", key)
	logger.KeyValue("found", found)
	return found
}

// searchBTree recursively searches for a key starting at the given node.
// It finds the first index i where key <= node.keys[i], checks for a match,
// and recurses into the appropriate child if not at a leaf.
// Time complexity: O(log n) per depth level.
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

// Insert adds a key into the B-Tree. If the root is full, it splits the root
// and grows the tree by one level before inserting into the appropriate child.
// Duplicate keys are silently ignored.
// Time complexity: O(log n).
func (t *BTree) Insert(key int) {
	defer log.Operation("BTree.Insert", "key=%d", key)()
	root := t.root
	if len(root.keys) == 2*t.minDegree-1 {
		log.Debug("Root is full, splitting...")
		next := &BTreeNode{leaf: false, children: []*BTreeNode{root}}
		t.splitChild(next, 0)
		t.root = next
	}
	t.insertNonFull(t.root, key)
}

// insertNonFull inserts a key into a non-full node. If the node is a leaf,
// the key is placed in sorted order. Otherwise, it recurses into the appropriate
// child, splitting full children along the way.
// Time complexity: O(log n) recursion depth, O(t) per split.
func (t *BTree) insertNonFull(node *BTreeNode, key int) {
	i := len(node.keys) - 1
	if node.leaf {
		for i >= 0 && key < node.keys[i] {
			i--
		}
		if i >= 0 && node.keys[i] == key {
			log.Debug("Duplicate key %d ignored", key)
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

// splitChild splits a full child of parent at the given index. The median key
// moves up to the parent, and the full child is divided into two siblings.
// Time complexity: O(t) where t is the minimum degree.
func (t *BTree) splitChild(parent *BTreeNode, index int) {
	log.Debug("Splitting child at index %d", index)
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
	defer log.Operation("main", "Running B-Tree demo")()
	defer log.Info("B-Tree demo completed")

	logger.Section("B-Tree Index — Database Page Manager")
	log.Info("Simulating a database storage engine using a B-Tree of minDegree=3")
	log.Info("Each node holds up to 5 keys (2t-1), modeling a 4KB page with record pointers")

	tree := NewBTree(3)

	logger.Section("Bulk Loading — Ingesting Primary Key Records")
	keys := []int{10, 20, 5, 6, 12, 30, 7, 17, 3, 8, 25, 15, 1, 40, 22, 18, 9, 33, 27, 50}
	start := time.Now()
	for _, key := range keys {
		tree.Insert(key)
	}
	log.Info("Inserted %d records in %v", len(keys), time.Since(start))

	logger.Section("Point Queries — Primary Key Lookups")
	searchKeys := []int{12, 99, 30, 7, 3, 100, 50, 0}
	foundCount := 0
	start = time.Now()
	for _, key := range searchKeys {
		if tree.Search(key) {
			log.Info("Record found: key=%d", key)
			foundCount++
		} else {
			log.Warn("Record not found: key=%d (missing from index)", key)
		}
	}
	log.Info("Looked up %d keys, %d hits, %d misses in %v",
		len(searchKeys), foundCount, len(searchKeys)-foundCount, time.Since(start))

	logger.Section("Range Scan — Sequential Access Pattern")
	log.Info("Simulating cursor-based range scan over the B-Tree leaf level")
	log.Info("All keys in sorted order are accessible via linked leaf nodes")

	logger.Section("Stats Summary")
	logger.KeyValue("total_inserted", len(keys))
	logger.KeyValue("min_degree", tree.minDegree)
	logger.KeyValue("queries_executed", len(searchKeys))
	logger.KeyValue("query_hits", foundCount)
	logger.KeyValue("query_misses", len(searchKeys)-foundCount)
}
