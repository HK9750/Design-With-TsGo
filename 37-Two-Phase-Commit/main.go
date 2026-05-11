package main

import (
	"design-with-tsgo/pkg/logger"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// Participant represents a resource manager in a 2PC protocol.
// Each participant must support Prepare (vote phase), Commit
// (apply changes), and Abort (rollback changes).
type Participant interface {
	Prepare(transactionID string) bool
	Commit(transactionID string)
	Abort(transactionID string)
}

// TwoPhaseCommitCoordinator orchestrates the two-phase commit protocol
// across multiple participants. Phase 1: all participants vote
// (Prepare). Phase 2: if all voted yes, Commit; otherwise Abort.
type TwoPhaseCommitCoordinator struct{ participants []Participant }

// NewTwoPhaseCommitCoordinator creates a coordinator for the given participants.
// Time complexity: O(1).
func NewTwoPhaseCommitCoordinator(participants []Participant) *TwoPhaseCommitCoordinator {
	defer log.Operation("NewTwoPhaseCommitCoordinator", "Creating 2PC coordinator with %d participants", len(participants))()
	return &TwoPhaseCommitCoordinator{participants: participants}
}

// Execute runs the two-phase commit protocol for the given transaction ID.
// Phase 1 (Prepare): asks all participants to vote. If any vote no, aborts
// all prepared participants. Phase 2 (Commit): all vote yes, commit all.
// Returns "committed" or "aborted".
// Time complexity: O(n * T_prepare + n * T_commit) in best case.
func (c *TwoPhaseCommitCoordinator) Execute(transactionID string) string {
	defer log.Operation("2PC Execute", "Executing 2PC for transaction %s", transactionID)()
	logger.Section("PHASE 1: Prepare (Voting)")

	phase1Start := time.Now()
	prepared := []Participant{}
	for i, participant := range c.participants {
		vote := participant.Prepare(transactionID)
		log.Debug("Participant %d voted: %v", i, vote)
		if !vote {
			log.Warn("Participant %d voted NO — aborting transaction %s", i, transactionID)
			for _, p := range prepared {
				p.Abort(transactionID)
			}
			log.Info("Phase 1 aborted in %v", time.Since(phase1Start))
			return "aborted"
		}
		prepared = append(prepared, participant)
	}
	log.Info("Phase 1 passed: all %d participants voted YES in %v", len(c.participants), time.Since(phase1Start))

	logger.Section("PHASE 2: Commit")
	phase2Start := time.Now()
	for i, participant := range prepared {
		participant.Commit(transactionID)
		log.Debug("Participant %d committed", i)
	}
	log.Info("Phase 2 completed in %v — transaction %s committed", time.Since(phase2Start), transactionID)
	return "committed"
}

// MemoryParticipant is a simple in-memory participant that always votes YES.
// It implements the Participant interface for testing and demonstration.
type MemoryParticipant struct{}

// Prepare always returns true (votes YES) for any transaction.
// Time complexity: O(1).
func (MemoryParticipant) Prepare(transactionID string) bool {
	log.Debug("MemoryParticipant: prepare vote YES for %s", transactionID)
	return true
}

// Commit applies the transaction (no-op for in-memory).
// Time complexity: O(1).
func (MemoryParticipant) Commit(transactionID string) {
	log.Debug("MemoryParticipant: committed %s", transactionID)
}

// Abort rolls back the transaction (no-op for in-memory).
// Time complexity: O(1).
func (MemoryParticipant) Abort(transactionID string) {
	log.Debug("MemoryParticipant: aborted %s", transactionID)
}

// FailingParticipant is a participant that votes NO, simulating a resource
// that cannot participate in the transaction (e.g., insufficient funds).
type FailingParticipant struct{ reason string }

// Prepare always returns false (votes NO), logging the reason.
// Time complexity: O(1).
func (fp FailingParticipant) Prepare(transactionID string) bool {
	log.Warn("FailingParticipant: vote NO for %s — %s", transactionID, fp.reason)
	return false
}

// Commit is a no-op since this participant never commits.
// Time complexity: O(1).
func (fp FailingParticipant) Commit(transactionID string) {
	log.Debug("FailingParticipant: commit (should never be called)")
}

// Abort is a no-op for this participant.
// Time complexity: O(1).
func (fp FailingParticipant) Abort(transactionID string) {
	log.Debug("FailingParticipant: abort %s", transactionID)
}

func main() {
	defer log.Operation("main", "Running Two-Phase Commit demo")()

	logger.Section("TWO-PHASE COMMIT — Production Scenario")

	// Simulate a distributed payment system: DB, Cache, and Message Queue
	log.Info("Setting up 2PC participants: orders-db, payment-gateway, notification-queue")

	logger.Section("SCENARIO: Successful Multi-Resource Commit")
	start := time.Now()
	successCoordinator := NewTwoPhaseCommitCoordinator([]Participant{
		MemoryParticipant{},
		MemoryParticipant{},
		MemoryParticipant{},
	})
	result := successCoordinator.Execute("txn-payment-1001")
	logger.KeyValue("result", result)
	log.Info("Transaction committed across all 3 resources in %v", time.Since(start))

	logger.Section("SCENARIO: Failed Transaction — Resource Votes NO")
	start = time.Now()
	failCoordinator := NewTwoPhaseCommitCoordinator([]Participant{
		MemoryParticipant{},
		MemoryParticipant{},
		FailingParticipant{reason: "insufficient balance"},
		MemoryParticipant{},
	})
	result = failCoordinator.Execute("txn-payment-1002")
	logger.KeyValue("result", result)
	log.Info("Transaction aborted in %v — all prepared resources rolled back", time.Since(start))

	logger.Section("SCENARIO: Single-Participant Commit (Simple Case)")
	start = time.Now()
	singleCoordinator := NewTwoPhaseCommitCoordinator([]Participant{
		MemoryParticipant{},
	})
	result = singleCoordinator.Execute("txn-config-2001")
	logger.KeyValue("result", result)
	log.Info("Single-participant commit done in %v", time.Since(start))

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("transactions_executed", 3)
	logger.KeyValue("committed", 2)
	logger.KeyValue("aborted", 1)
	log.Info("Two-phase commit demo completed successfully")
}
