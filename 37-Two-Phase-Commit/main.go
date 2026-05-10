package main

import "fmt"

type Participant interface {
	Prepare(transactionID string) bool
	Commit(transactionID string)
	Abort(transactionID string)
}

type TwoPhaseCommitCoordinator struct{ participants []Participant }

func NewTwoPhaseCommitCoordinator(participants []Participant) *TwoPhaseCommitCoordinator {
	return &TwoPhaseCommitCoordinator{participants: participants}
}

func (c *TwoPhaseCommitCoordinator) Execute(transactionID string) string {
	prepared := []Participant{}
	for _, participant := range c.participants {
		if !participant.Prepare(transactionID) {
			for _, p := range prepared {
				p.Abort(transactionID)
			}
			return "aborted"
		}
		prepared = append(prepared, participant)
	}
	for _, participant := range prepared {
		participant.Commit(transactionID)
	}
	return "committed"
}

type MemoryParticipant struct{}

func (MemoryParticipant) Prepare(string) bool { return true }
func (MemoryParticipant) Commit(string)       {}
func (MemoryParticipant) Abort(string)        {}

func main() {
	coordinator := NewTwoPhaseCommitCoordinator([]Participant{MemoryParticipant{}})
	fmt.Println(coordinator.Execute("tx1"))
}
