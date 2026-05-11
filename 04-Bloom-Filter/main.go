// Package main demonstrates a Bloom Filter — a space-efficient probabilistic data structure
// for set membership testing with configurable false positive rate.
// Ideal for username availability checking in a registration system.
package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// BloomFilter is a probabilistic data structure that tests whether an element
// is a member of a set. False positives are possible, but false negatives are not.
// It uses multiple hash functions (via double hashing) to set bits in a bit array.
type BloomFilter struct {
	bits      []byte
	bitSize   uint32
	hashCount uint32
}

// NewBloomFilter creates a Bloom filter optimized for the given expected number of items
// and target false positive rate. Computes the optimal bit array size and hash count.
// Exits on invalid parameters.
func NewBloomFilter(expectedItems int, falsePositiveRate float64) *BloomFilter {
	if expectedItems <= 0 {
		log.Error("expectedItems must be positive (got %d)", expectedItems)
		os.Exit(1)
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		log.Error("falsePositiveRate must be between 0 and 1 (got %f)", falsePositiveRate)
		os.Exit(1)
	}

	ln2 := math.Log(2)
	bitSize := uint32(math.Ceil((-float64(expectedItems) * math.Log(falsePositiveRate)) / (ln2 * ln2)))
	hashCount := uint32(math.Max(1, math.Round((float64(bitSize)/float64(expectedItems))*ln2)))

	log.Debug("Created Bloom filter (expected_items=%d, fp_rate=%.4f, bits=%d, hashes=%d)",
		expectedItems, falsePositiveRate, bitSize, hashCount)
	return &BloomFilter{bits: make([]byte, (bitSize+7)/8), bitSize: bitSize, hashCount: hashCount}
}

// Add inserts a value into the Bloom filter. Time complexity: O(k) where k is the number of hash functions.
func (bf *BloomFilter) Add(value string) {
	log.Debug("BloomFilter add (value=%q)", value)
	for _, index := range bf.indexes(value) {
		bf.bits[index>>3] |= 1 << (index & 7)
	}
}

// Has tests whether a value is possibly in the set. Returns true if the value might be present,
// false if it is definitely not present. Time complexity: O(k).
func (bf *BloomFilter) Has(value string) bool {
	for _, index := range bf.indexes(value) {
		if bf.bits[index>>3]&(1<<(index&7)) == 0 {
			log.Debug("BloomFilter definitely absent (value=%q)", value)
			return false
		}
	}
	log.Debug("BloomFilter possibly present (value=%q)", value)
	return true
}

// EstimatedFalsePositiveRate returns the current estimated false positive rate based on
// the number of items inserted so far.
func (bf *BloomFilter) EstimatedFalsePositiveRate(insertedItems int) float64 {
	rate := math.Pow(1-math.Exp((-float64(bf.hashCount)*float64(insertedItems))/float64(bf.bitSize)), float64(bf.hashCount))
	log.Debug("BloomFilter estimated FP rate (items=%d): %.6f", insertedItems, rate)
	return rate
}

// indexes computes the k hash positions for a value using double hashing with FNV-1a.
// Uses the formula: h1 + i*h2 mod bitSize for each of the k hash functions.
func (bf *BloomFilter) indexes(value string) []uint32 {
	h1 := fnv1aWithSeed(value, 2166136261)
	h2 := fnv1aWithSeed(value, 16777619)
	if h2 == 0 {
		h2 = 1
	}
	indexes := make([]uint32, bf.hashCount)
	for i := uint32(0); i < bf.hashCount; i++ {
		indexes[i] = (h1 + i*h2) % bf.bitSize
	}
	return indexes
}

// fnv1aWithSeed computes the FNV-1a hash of a string with a custom seed.
func fnv1aWithSeed(value string, seed uint32) uint32 {
	hash := seed
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

// main demonstrates the Bloom filter as a username availability checker for a
// social media platform, preventing registration of already-taken usernames.
func main() {
	defer log.Operation("main", "Running Bloom Filter demo")()

	logger.Section("Bloom Filter — Username Availability Checker")
	log.Info("Configuring filter for 1M users at 0.1%% false positive rate")

	filter := NewBloomFilter(1000000, 0.001)
	logger.KeyValue("expected_items", 1000000)
	logger.KeyValue("target_fp_rate", "0.1%")
	logger.KeyValue("bit_size", filter.bitSize)
	logger.KeyValue("hash_functions", filter.hashCount)

	logger.Section("Registering taken usernames")
	takenNames := []string{"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry", "iris", "jack"}
	for _, name := range takenNames {
		filter.Add(name)
		log.Debug("Registered username: %q", name)
	}
	log.Info("Registered %d usernames", len(takenNames))

	logger.Section("Checking username availability")
	log.Info("alice available? %v (should be false)", filter.Has("alice"))
	log.Info("bob available? %v (should be false)", filter.Has("bob"))
	log.Info("carol available? %v (likely true — never registered)", filter.Has("carol"))
	log.Info("zach available? %v (likely true — never registered)", filter.Has("zach"))

	logger.Section("Bulk registration simulation")
	start := time.Now()
	for i := 0; i < 10000; i++ {
		name := fmt.Sprintf("user_%d", i)
		filter.Add(name)
	}
	log.Info("Bulk registered 10,000 usernames in %v", time.Since(start))

	logger.Section("Verification — known positive")
	allFound := true
	for i := 0; i < 10000; i++ {
		if !filter.Has(fmt.Sprintf("user_%d", i)) {
			allFound = false
			break
		}
	}
	log.Info("All 10,000 bulk users found: %v (no false negatives)", allFound)

	logger.Section("False positive demonstration")
	fpCount := 0
	testSize := 100000
	for i := 0; i < testSize; i++ {
		if filter.Has(fmt.Sprintf("nonexistent_%d", i)) {
			fpCount++
		}
	}
	log.Info("False positives in 100,000 unknown names: %d (%.4f%%)", fpCount, float64(fpCount)/float64(testSize)*100)

	logger.Section("Final Stats")
	log.Info("Estimated false positive rate: %.6f%%", filter.EstimatedFalsePositiveRate(10010)*100)
	log.Info("Memory used: %d bytes (bit array)", len(filter.bits))
	log.Info("All operations completed — O(k) add/lookup with k=%d hash functions", filter.hashCount)
}
