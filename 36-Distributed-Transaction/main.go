package main

import (
	"design-with-tsgo/pkg/logger"
	"fmt"
	"os"
	"time"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// SagaStep represents a single step in a Saga distributed transaction.
// Each step has an Action (the forward operation) and a Compensate
// function (the rollback operation if a later step fails).
type SagaStep struct {
	Name       string
	Action     func() error
	Compensate func() error
}

// SagaCoordinator orchestrates a sequence of SagaSteps, executing each
// in order. If any step fails, all previously completed steps are
// compensated in reverse order. Returns true if all steps succeeded.
type SagaCoordinator struct{ steps []SagaStep }

// NewSagaCoordinator creates a coordinator for the given saga steps.
// Time complexity: O(1).
func NewSagaCoordinator(steps []SagaStep) *SagaCoordinator {
	defer log.Operation("NewSagaCoordinator", "Creating saga with %d steps", len(steps))()
	return &SagaCoordinator{steps: steps}
}

// Execute runs all saga steps sequentially. If any step's Action returns
// an error, all prior steps are compensated in reverse order and Execute
// returns false. If all steps succeed, returns true.
// Time complexity: O(n * (T_action + T_compensate)) in worst case.
func (s *SagaCoordinator) Execute() bool {
	defer log.Operation("Execute", "Starting saga execution (%d steps)", len(s.steps))()
	completed := make([]SagaStep, 0, len(s.steps))
	for i, step := range s.steps {
		log.Step("Executing step %d/%d: %s", i+1, len(s.steps), step.Name)
		if err := step.Action(); err != nil {
			log.Warn("Step '%s' failed: %v — initiating compensation", step.Name, err)
			compStart := time.Now()
			for j := len(completed) - 1; j >= 0; j-- {
				log.Step("Compensating step: %s", completed[j].Name)
				if compErr := completed[j].Compensate(); compErr != nil {
					log.Error("Compensation for '%s' failed: %v", completed[j].Name, compErr)
				}
			}
			log.Info("Compensation completed in %v", time.Since(compStart))
			return false
		}
		completed = append(completed, step)
		log.Debug("Step '%s' completed successfully", step.Name)
	}
	log.Info("All %d saga steps committed successfully", len(s.steps))
	return true
}

func main() {
	defer log.Operation("main", "Running Distributed Saga Transaction demo")()

	logger.Section("SAGA DISTRIBUTED TRANSACTION — Production Scenario")

	// Simulate an e-commerce order placement saga:
	// 1. Reserve inventory
	// 2. Charge payment
	// 3. Create shipment
	// 4. Send confirmation email

	events := []string{}

	successSaga := NewSagaCoordinator([]SagaStep{
		{
			Name: "reserve-inventory",
			Action: func() error {
				events = append(events, "inventory-reserved")
				return nil
			},
			Compensate: func() error {
				events = append(events, "inventory-released")
				return nil
			},
		},
		{
			Name: "charge-payment",
			Action: func() error {
				events = append(events, "payment-charged")
				return nil
			},
			Compensate: func() error {
				events = append(events, "payment-refunded")
				return nil
			},
		},
		{
			Name: "create-shipment",
			Action: func() error {
				events = append(events, "shipment-created")
				return nil
			},
			Compensate: func() error {
				events = append(events, "shipment-cancelled")
				return nil
			},
		},
		{
			Name: "send-confirmation",
			Action: func() error {
				events = append(events, "confirmation-sent")
				return nil
			},
			Compensate: func() error {
				events = append(events, "confirmation-retracted")
				return nil
			},
		},
	})

	logger.Section("SCENARIO: Successful Order Placement")
	log.Info("Processing order #ORD-1001: 3 items, $150.00 total")
	start := time.Now()
	ok := successSaga.Execute()
	log.Info("Saga execution completed in %v", time.Since(start))
	logger.KeyValue("success", ok)
	logger.KeyValue("events", fmt.Sprintf("%v", events))

	logger.Section("SCENARIO: Failed Order with Compensation")
	events = []string{}
	failSaga := NewSagaCoordinator([]SagaStep{
		{
			Name: "reserve-inventory",
			Action: func() error {
				events = append(events, "inventory-reserved")
				return nil
			},
			Compensate: func() error {
				events = append(events, "inventory-released")
				return nil
			},
		},
		{
			Name: "charge-payment",
			Action: func() error {
				events = append(events, "payment-charged")
				return nil
			},
			Compensate: func() error {
				events = append(events, "payment-refunded")
				return nil
			},
		},
		{
			Name: "create-shipment",
			Action: func() error {
				return fmt.Errorf("warehouse capacity exceeded")
			},
			Compensate: func() error {
				events = append(events, "shipment-cancelled")
				return nil
			},
		},
	})

	log.Info("Processing order #ORD-1002: shipment step will fail")
	start = time.Now()
	ok = failSaga.Execute()
	log.Info("Saga execution completed in %v", time.Since(start))
	logger.KeyValue("success", ok)
	logger.KeyValue("events", fmt.Sprintf("%v", events))
	log.Info("Compensation reversed inventory and payment — system is consistent")

	// Final stats summary
	logger.Section("FINAL STATS")
	logger.KeyValue("orders_processed", 2)
	logger.KeyValue("successful_orders", 1)
	logger.KeyValue("compensated_orders", 1)
	logger.KeyValue("total_saga_steps", 4+3)
	log.Info("Distributed saga transaction demo completed successfully")
}
