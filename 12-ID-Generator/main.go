package main

import (
	"os"
	"sync"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// SnowflakeGenerator produces unique 64-bit IDs using the Twitter Snowflake
// algorithm. Layout: [41-bit timestamp][10-bit machineID][12-bit sequence].
// Supports ~4096 IDs/ms per machine for ~69 years from epoch.
type SnowflakeGenerator struct {
	mu            sync.Mutex
	machineID     uint64
	epochMs       int64
	sequence      uint64
	lastTimestamp int64
}

// NewSnowflakeGenerator creates a Snowflake ID generator. machineID must fit in
// 10 bits (0-1023). epochMs is the custom epoch in Unix milliseconds.
// Time complexity: O(1).
func NewSnowflakeGenerator(machineID uint64, epochMs int64) *SnowflakeGenerator {
	if machineID > 1023 {
		log.Error("machineID %d exceeds 10-bit limit (max 1023)", machineID)
		os.Exit(1)
	}
	log.Debug("Created snowflake generator: machineID=%d epochMs=%d", machineID, epochMs)
	return &SnowflakeGenerator{machineID: machineID, epochMs: epochMs, lastTimestamp: -1}
}

// NextID generates the next unique 64-bit ID. Thread-safe. Will block and spin
// if the sequence overflows within the same millisecond. Returns nonzero on a
// clock regression. Time complexity: O(1) amortized.
func (g *SnowflakeGenerator) NextID() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	timestamp := time.Now().UnixMilli() - g.epochMs
	if timestamp < g.lastTimestamp {
		log.Error("Clock moved backwards: timestamp=%d lastTimestamp=%d", timestamp, g.lastTimestamp)
		os.Exit(1)
	}
	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & 4095
		for g.sequence == 0 {
			log.Debug("Sequence overflow, waiting for next millisecond")
			timestamp = time.Now().UnixMilli() - g.epochMs
			if timestamp > g.lastTimestamp {
				break
			}
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = timestamp
	id := (uint64(timestamp) << 22) | (g.machineID << 12) | g.sequence
	log.Debug("Generated ID: %d (ts=%d machine=%d seq=%d)", id, timestamp, g.machineID, g.sequence)
	return id
}

func main() {
	defer log.Operation("main", "Running Snowflake ID Generator demo")()

	logger.Section("Scenario: Distributed Order ID Generation Service")
	log.Info("Generating unique order IDs across 3 data center machines")

	machines := []uint64{7, 15, 23}
	epochMs := int64(1704067200000) // 2024-01-01T00:00:00Z

	generators := make([]*SnowflakeGenerator, len(machines))
	for i, mid := range machines {
		generators[i] = NewSnowflakeGenerator(mid, epochMs)
	}

	idsPerMachine := 5
	totalIDs := len(machines) * idsPerMachine
	allIDs := make([]uint64, 0, totalIDs)

	start := time.Now()
	for _, gen := range generators {
		for j := 0; j < idsPerMachine; j++ {
			id := gen.NextID()
			allIDs = append(allIDs, id)
		}
	}
	elapsed := time.Since(start)

	logger.Section("Generated IDs")
	for i, id := range allIDs {
		logger.KeyValue("id", id)
		_ = i
	}

	logger.Section("Final Stats Summary")
	logger.KeyValue("totalIdsGenerated", totalIDs)
	logger.KeyValue("machinesSimulated", len(machines))
	logger.KeyValue("idsPerMachine", idsPerMachine)
	logger.KeyValue("epoch", epochMs)
	logger.KeyValue("generationTime", elapsed)
	log.Info("ID generation completed in %v", elapsed)
}
