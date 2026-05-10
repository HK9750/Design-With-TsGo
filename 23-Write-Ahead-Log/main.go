package main

import "fmt"

type WALEntry struct {
	LSN      int
	Op       string
	Key      string
	Value    string
	Checksum uint32
}

type WriteAheadLog struct {
	entries []WALEntry
	nextLSN int
}

func NewWriteAheadLog() *WriteAheadLog { return &WriteAheadLog{nextLSN: 1} }

func (w *WriteAheadLog) Append(op, key, value string) WALEntry {
	base := fmt.Sprintf("%d|%s|%s|%s", w.nextLSN, op, key, value)
	entry := WALEntry{LSN: w.nextLSN, Op: op, Key: key, Value: value, Checksum: checksum(base)}
	w.nextLSN++
	w.entries = append(w.entries, entry)
	return entry
}

func (w *WriteAheadLog) Recover() (map[string]string, error) {
	state := make(map[string]string)
	for _, entry := range w.entries {
		base := fmt.Sprintf("%d|%s|%s|%s", entry.LSN, entry.Op, entry.Key, entry.Value)
		if checksum(base) != entry.Checksum {
			return nil, fmt.Errorf("corrupt WAL entry %d", entry.LSN)
		}
		if entry.Op == "put" {
			state[entry.Key] = entry.Value
		} else {
			delete(state, entry.Key)
		}
	}
	return state, nil
}

func checksum(value string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(value); i++ {
		hash ^= uint32(value[i])
		hash *= 16777619
	}
	return hash
}

func main() {
	wal := NewWriteAheadLog()
	wal.Append("put", "x", "1")
	wal.Append("delete", "x", "")
	state, _ := wal.Recover()
	_, exists := state["x"]
	fmt.Println(exists)
}
