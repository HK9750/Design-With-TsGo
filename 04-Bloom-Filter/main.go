package main

import (
	"fmt"
	"math"
)

type BloomFilter struct {
	bits      []byte
	bitSize   uint32
	hashCount uint32
}

func NewBloomFilter(expectedItems int, falsePositiveRate float64) *BloomFilter {
	if expectedItems <= 0 {
		panic("expectedItems must be positive")
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		panic("falsePositiveRate must be between 0 and 1")
	}

	ln2 := math.Log(2)
	bitSize := uint32(math.Ceil((-float64(expectedItems) * math.Log(falsePositiveRate)) / (ln2 * ln2)))
	hashCount := uint32(math.Max(1, math.Round((float64(bitSize)/float64(expectedItems))*ln2)))
	return &BloomFilter{bits: make([]byte, (bitSize+7)/8), bitSize: bitSize, hashCount: hashCount}
}

func (bf *BloomFilter) Add(value string) {
	for _, index := range bf.indexes(value) {
		bf.bits[index>>3] |= 1 << (index & 7)
	}
}

func (bf *BloomFilter) Has(value string) bool {
	for _, index := range bf.indexes(value) {
		if bf.bits[index>>3]&(1<<(index&7)) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) EstimatedFalsePositiveRate(insertedItems int) float64 {
	return math.Pow(1-math.Exp((-float64(bf.hashCount)*float64(insertedItems))/float64(bf.bitSize)), float64(bf.hashCount))
}

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

func fnv1aWithSeed(value string, seed uint32) uint32 {
	hash := seed
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	filter := NewBloomFilter(1000, 0.01)
	filter.Add("alice")
	filter.Add("bob")
	fmt.Println(filter.Has("alice"), filter.Has("carol"))
}
