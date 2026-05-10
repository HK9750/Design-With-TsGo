package main

import (
	"fmt"
	"sync"
	"time"
)

type SnowflakeGenerator struct {
	mu            sync.Mutex
	machineID     uint64
	epochMs       int64
	sequence      uint64
	lastTimestamp int64
}

func NewSnowflakeGenerator(machineID uint64, epochMs int64) *SnowflakeGenerator {
	if machineID > 1023 {
		panic("machineID must fit in 10 bits")
	}
	return &SnowflakeGenerator{machineID: machineID, epochMs: epochMs, lastTimestamp: -1}
}

func (g *SnowflakeGenerator) NextID() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	timestamp := time.Now().UnixMilli() - g.epochMs
	if timestamp < g.lastTimestamp {
		panic("clock moved backwards")
	}
	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & 4095
		for g.sequence == 0 {
			timestamp = time.Now().UnixMilli() - g.epochMs
			if timestamp > g.lastTimestamp {
				break
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = timestamp
	return (uint64(timestamp) << 22) | (g.machineID << 12) | g.sequence
}

func main() {
	generator := NewSnowflakeGenerator(7, 1704067200000)
	fmt.Println(generator.NextID())
}
