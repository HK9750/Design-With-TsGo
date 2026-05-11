package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// LSMValue holds a key's value, deletion tombstone flag, and a monotonically
// increasing sequence number used for conflict resolution during compaction.
type LSMValue struct {
	Value    string
	Deleted  bool
	Sequence int64
}

// LSMTree is a Log-Structured Merge-Tree implementation that buffers writes in
// an in-memory memtable and periodically flushes it to immutable on-disk
// SSTables (simulated as in-memory maps). Reads scan from newest to oldest
// segment. Compaction merges SSTables to reclaim space.
// Time complexity: O(1) amortized writes, O(k) reads where k is SSTable count.
type LSMTree struct {
	memtable       map[string]LSMValue
	sstables       []map[string]LSMValue
	sequence       int64
	flushThreshold int
}

// NewLSMTree creates an LSM-Tree with the given memtable flush threshold.
// When the memtable reaches this size, it is automatically flushed to an SSTable.
// If threshold <= 0, defaults to 4.
// Time complexity: O(1).
func NewLSMTree(flushThreshold int) *LSMTree {
	if flushThreshold <= 0 {
		flushThreshold = 4
	}
	log.Debug("Created LSM-Tree with flushThreshold=%d", flushThreshold)
	return &LSMTree{memtable: make(map[string]LSMValue), flushThreshold: flushThreshold}
}

// Put writes a key-value pair into the memtable with an auto-incremented sequence
// number. If the memtable reaches the flush threshold, it triggers an automatic
// flush to an SSTable.
// Time complexity: O(1) amortized.
func (t *LSMTree) Put(key, value string) {
	defer log.Operation("LSMTree.Put", "key=%s value=%s", key, value)()
	t.sequence++
	t.memtable[key] = LSMValue{Value: value, Sequence: t.sequence}
	log.Debug("Written to memtable at seq=%d", t.sequence)
	if len(t.memtable) >= t.flushThreshold {
		log.Debug("Memtable threshold reached (%d >= %d), flushing...", len(t.memtable), t.flushThreshold)
		t.Flush()
	}
}

// Delete marks a key as deleted by writing a tombstone entry into the memtable.
// The tombstone prevents reads from returning the key and allows compaction to
// physically remove the data.
// Time complexity: O(1) amortized.
func (t *LSMTree) Delete(key string) {
	defer log.Operation("LSMTree.Delete", "key=%s", key)()
	t.sequence++
	t.memtable[key] = LSMValue{Deleted: true, Sequence: t.sequence}
	log.Debug("Tombstone written to memtable at seq=%d", t.sequence)
	if len(t.memtable) >= t.flushThreshold {
		t.Flush()
	}
}

// Get retrieves the value for a key by searching the memtable first, then
// scanning SSTables from newest to oldest. Returns the value and a boolean
// indicating whether the key exists and is not deleted.
// Time complexity: O(k) where k is the number of SSTables.
func (t *LSMTree) Get(key string) (string, bool) {
	defer log.Operation("LSMTree.Get", "key=%s", key)()
	if entry, ok := t.memtable[key]; ok {
		log.Debug("Found in memtable (seq=%d)", entry.Sequence)
		return entry.Value, !entry.Deleted
	}
	for i := len(t.sstables) - 1; i >= 0; i-- {
		if entry, ok := t.sstables[i][key]; ok {
			log.Debug("Found in SSTable[%d] (seq=%d)", i, entry.Sequence)
			return entry.Value, !entry.Deleted
		}
	}
	log.Warn("Key not found: %s", key)
	return "", false
}

// Flush persists the current memtable as a new SSTable and resets the memtable
// to an empty map. No-op if the memtable is empty.
// Time complexity: O(n) where n is the memtable size.
func (t *LSMTree) Flush() {
	defer log.Operation("LSMTree.Flush", "memtable_size=%d", len(t.memtable))()
	if len(t.memtable) == 0 {
		log.Debug("Memtable is empty, skipping flush")
		return
	}
	table := make(map[string]LSMValue, len(t.memtable))
	for key, value := range t.memtable {
		table[key] = value
	}
	t.sstables = append(t.sstables, table)
	log.Info("Flushed %d entries to SSTable[%d]", len(table), len(t.sstables)-1)
	t.memtable = make(map[string]LSMValue)
}

// Compact merges all SSTables into a single sorted table by keeping the entry
// with the highest sequence number per key. Tombstones (deleted=true) are
// physically removed during compaction.
// Time complexity: O(N) where N is the total number of entries across all SSTables.
func (t *LSMTree) Compact() {
	defer log.Operation("LSMTree.Compact", "sstables=%d", len(t.sstables))()
	start := time.Now()
	merged := make(map[string]LSMValue)
	for _, table := range t.sstables {
		for key, entry := range table {
			current, ok := merged[key]
			if !ok || entry.Sequence > current.Sequence {
				merged[key] = entry
			}
		}
	}
	tombstoneCount := 0
	for key, entry := range merged {
		if entry.Deleted {
			delete(merged, key)
			tombstoneCount++
		}
	}
	t.sstables = []map[string]LSMValue{merged}
	log.Info("Compaction completed: %d live keys, %d tombstones removed in %v",
		len(merged), tombstoneCount, time.Since(start))
}

func main() {
	defer log.Operation("main", "Running LSM-Tree demo")()
	defer log.Info("LSM-Tree demo completed")

	logger.Section("LSM-Tree — Write-Optimized Storage Engine")
	log.Info("Simulating a storage backend for a time-series metrics database")
	log.Info("Flush threshold set low (3) to demonstrate frequent SSTable creation")

	tree := NewLSMTree(3)

	logger.Section("Ingesting Metrics — High-Volume Writes")
	start := time.Now()
	tree.Put("cpu.host1", "0.42")
	tree.Put("cpu.host2", "0.67")
	tree.Put("mem.host1", "0.83")
	tree.Put("cpu.host1", "0.55")
	tree.Put("cpu.host3", "0.31")
	tree.Put("mem.host2", "0.72")
	log.Info("Ingested 6 metrics in %v", time.Since(start))

	logger.Section("Point Reads — Querying Specific Metrics")
	a, aOK := tree.Get("cpu.host1")
	log.Info("cpu.host1 = %s (found=%v)", a, aOK)
	b, bOK := tree.Get("mem.host1")
	log.Info("mem.host1 = %s (found=%v)", b, bOK)
	_, cOK := tree.Get("disk.host1")
	if !cOK {
		log.Warn("disk.host1 not found — metric not yet ingested")
	}

	logger.Section("Deletion — Retiring Stale Metrics")
	tree.Delete("cpu.host3")
	d, dOK := tree.Get("cpu.host3")
	log.Info("cpu.host3 after delete: found=%v (tombstone in effect)", dOK)
	if dOK {
		log.Warn("cpu.host3 unexpectedly returned value=%s", d)
	}

	logger.Section("Compaction — Reclaiming Space")
	log.Info("SSTables before compaction: %d", len(tree.sstables))
	beforeCompact := len(tree.sstables)
	tree.Compact()
	log.Info("SSTables after compaction: %d", len(tree.sstables))

	logger.Section("Stats Summary")
	logger.KeyValue("sstables_created", beforeCompact)
	logger.KeyValue("sstables_after_compact", len(tree.sstables))
	logger.KeyValue("total_sequence", tree.sequence)
	logger.KeyValue("memtable_size", len(tree.memtable))
}
