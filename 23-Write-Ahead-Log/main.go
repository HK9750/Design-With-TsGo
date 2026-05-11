package main

import (
	"fmt"
	"os"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// WALEntry represents a single entry in the Write-Ahead Log. It contains a
// monotonically increasing Log Sequence Number (LSN), the operation type
// ("put" or "delete"), the affected key/value, and a FNV-1a checksum for
// integrity verification.
type WALEntry struct {
	LSN      int
	Op       string
	Key      string
	Value    string
	Checksum uint32
}

// WriteAheadLog provides durability guarantees by recording all state mutations
// before they are applied to the main data store. On crash recovery, the log is
// replayed from the beginning to reconstruct the consistent state.
// Time complexity: O(1) per append, O(n) per recovery.
type WriteAheadLog struct {
	entries []WALEntry
	nextLSN int
}

// NewWriteAheadLog creates an empty WAL with LSN counter starting at 1.
// Time complexity: O(1).
func NewWriteAheadLog() *WriteAheadLog {
	log.Debug("Initialized Write-Ahead Log, starting LSN=1")
	return &WriteAheadLog{nextLSN: 1}
}

// Append writes a new operation to the WAL with an auto-incremented LSN and a
// computed checksum. Returns the full WALEntry for caller reference.
// Time complexity: O(1).
func (w *WriteAheadLog) Append(op, key, value string) WALEntry {
	defer log.Operation("WAL.Append", "op=%s key=%s value=%s", op, key, value)()
	base := fmt.Sprintf("%d|%s|%s|%s", w.nextLSN, op, key, value)
	entry := WALEntry{LSN: w.nextLSN, Op: op, Key: key, Value: value, Checksum: checksum(base)}
	w.nextLSN++
	w.entries = append(w.entries, entry)
	log.Debug("Appended LSN=%d, checksum=0x%x", entry.LSN, entry.Checksum)
	return entry
}

// Recover replays all WAL entries from the beginning to reconstruct the state.
// Each entry's checksum is verified before applying. Returns the reconstructed
// key-value map, or an error if corruption is detected.
// Time complexity: O(n) where n is the number of entries.
func (w *WriteAheadLog) Recover() (map[string]string, error) {
	defer log.Operation("WAL.Recover", "entries=%d", len(w.entries))()
	state := make(map[string]string)
	for _, entry := range w.entries {
		base := fmt.Sprintf("%d|%s|%s|%s", entry.LSN, entry.Op, entry.Key, entry.Value)
		if checksum(base) != entry.Checksum {
			log.Error("Corrupt WAL entry at LSN=%d, checksum mismatch", entry.LSN)
			return nil, fmt.Errorf("corrupt WAL entry %d", entry.LSN)
		}
		if entry.Op == "put" {
			state[entry.Key] = entry.Value
			log.Debug("Replayed put: %s=%s", entry.Key, entry.Value)
		} else {
			delete(state, entry.Key)
			log.Debug("Replayed delete: %s", entry.Key)
		}
	}
	log.Info("Recovery complete: %d entries replayed, %d keys in final state", len(w.entries), len(state))
	return state, nil
}

// checksum computes a 32-bit FNV-1a hash of the input string. Used for
// integrity verification of WAL entries.
func checksum(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	defer log.Operation("main", "Running Write-Ahead Log demo")()
	defer log.Info("WAL demo completed")

	logger.Section("Write-Ahead Log — Crash Recovery Engine")
	log.Info("Simulating a transactional key-value store that uses WAL for durability")
	log.Info("Scenario: Server crashes mid-operation, WAL replays to restore state")

	wal := NewWriteAheadLog()

	logger.Section("Normal Operation — Recording Mutations")
	log.Info("Transaction T1: Setting up user session data")
	wal.Append("put", "session:1001", "active")
	wal.Append("put", "user:42", "alice")
	wal.Append("put", "balance:42", "150.00")

	log.Info("Transaction T2: Updating balance after purchase")
	wal.Append("put", "balance:42", "120.50")

	log.Info("Transaction T3: Deleting expired session")
	wal.Append("delete", "session:1001", "")

	logger.Section("Crash Recovery — Replaying the Log")
	log.Info("*** SIMULATED CRASH *** — Server restarts, data lost from memory")
	log.Info("Replaying WAL to reconstruct consistent state...")

	state, err := wal.Recover()
	if err != nil {
		log.Error("Recovery failed: %v", err)
		os.Exit(1)
	}

	logger.Section("Post-Recovery State Verification")
	log.Info("Checking reconstructed state after WAL replay:")
	for key, value := range state {
		log.Info("  %s = %s", key, value)
	}

	// Verify expected state
	checks := map[string]string{
		"user:42":    "alice",
		"balance:42": "120.50",
	}
	for key, expected := range checks {
		if val, ok := state[key]; ok {
			if val == expected {
				log.Info("  %s = %s (correct)", key, val)
			} else {
				log.Warn("  %s = %s (expected %s)", key, val, expected)
			}
		} else {
			log.Warn("  %s is missing from recovered state", key)
		}
	}

	_, exists := state["session:1001"]
	if exists {
		log.Warn("session:1001 still exists after delete operation")
	} else {
		log.Info("session:1001 correctly deleted (tombstone applied)")
	}

	logger.Section("Stats Summary")
	logger.KeyValue("total_wal_entries", len(wal.entries))
	logger.KeyValue("next_lsn", wal.nextLSN)
	logger.KeyValue("recovered_keys", len(state))
	logger.KeyValue("session_deleted", !exists)
}
