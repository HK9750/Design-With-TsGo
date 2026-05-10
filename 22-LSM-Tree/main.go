package main

import "fmt"

type LSMValue struct {
	Value    string
	Deleted  bool
	Sequence int64
}

type LSMTree struct {
	memtable       map[string]LSMValue
	sstables       []map[string]LSMValue
	sequence       int64
	flushThreshold int
}

func NewLSMTree(flushThreshold int) *LSMTree {
	if flushThreshold <= 0 {
		flushThreshold = 4
	}
	return &LSMTree{memtable: make(map[string]LSMValue), flushThreshold: flushThreshold}
}

func (t *LSMTree) Put(key, value string) {
	t.sequence++
	t.memtable[key] = LSMValue{Value: value, Sequence: t.sequence}
	if len(t.memtable) >= t.flushThreshold {
		t.Flush()
	}
}

func (t *LSMTree) Delete(key string) {
	t.sequence++
	t.memtable[key] = LSMValue{Deleted: true, Sequence: t.sequence}
	if len(t.memtable) >= t.flushThreshold {
		t.Flush()
	}
}

func (t *LSMTree) Get(key string) (string, bool) {
	if entry, ok := t.memtable[key]; ok {
		return entry.Value, !entry.Deleted
	}
	for i := len(t.sstables) - 1; i >= 0; i-- {
		if entry, ok := t.sstables[i][key]; ok {
			return entry.Value, !entry.Deleted
		}
	}
	return "", false
}

func (t *LSMTree) Flush() {
	if len(t.memtable) == 0 {
		return
	}
	table := make(map[string]LSMValue, len(t.memtable))
	for key, value := range t.memtable {
		table[key] = value
	}
	t.sstables = append(t.sstables, table)
	t.memtable = make(map[string]LSMValue)
}

func (t *LSMTree) Compact() {
	merged := make(map[string]LSMValue)
	for _, table := range t.sstables {
		for key, entry := range table {
			current, ok := merged[key]
			if !ok || entry.Sequence > current.Sequence {
				merged[key] = entry
			}
		}
	}
	for key, entry := range merged {
		if entry.Deleted {
			delete(merged, key)
		}
	}
	t.sstables = []map[string]LSMValue{merged}
}

func main() {
	tree := NewLSMTree(2)
	tree.Put("a", "1")
	tree.Put("b", "2")
	tree.Delete("a")
	a, aOK := tree.Get("a")
	b, ok := tree.Get("b")
	fmt.Println(a, aOK, b, ok)
}
