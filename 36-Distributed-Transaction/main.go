package main

import "fmt"

type SagaStep struct {
	Name       string
	Action     func() error
	Compensate func() error
}

type SagaCoordinator struct{ steps []SagaStep }

func NewSagaCoordinator(steps []SagaStep) *SagaCoordinator { return &SagaCoordinator{steps: steps} }

func (s *SagaCoordinator) Execute() bool {
	completed := make([]SagaStep, 0, len(s.steps))
	for _, step := range s.steps {
		if err := step.Action(); err != nil {
			for i := len(completed) - 1; i >= 0; i-- {
				_ = completed[i].Compensate()
			}
			return false
		}
		completed = append(completed, step)
	}
	return true
}

func main() {
	events := []string{}
	saga := NewSagaCoordinator([]SagaStep{{Name: "order", Action: func() error { events = append(events, "order"); return nil }, Compensate: func() error { events = append(events, "undo-order"); return nil }}})
	fmt.Println(saga.Execute(), events)
}
