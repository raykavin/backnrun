package strategy

import (
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/logger"
)

// OrderCondition represents a conditional trading order to be executed when the condition is met.
type OrderCondition struct {
	Condition func(df *core.Dataframe) bool
	Size      float64
	Side      core.SideType
}

// Scheduler manages conditional orders for a trading pair.
type Scheduler struct {
	pair            string
	log             logger.Logger
	orderConditions []OrderCondition
}

// NewScheduler creates a new Scheduler instance for the specified trading pair.
func NewScheduler(pair string, log logger.Logger) *Scheduler {
	return &Scheduler{
		pair:            pair,
		log:             log,
		orderConditions: make([]OrderCondition, 0),
	}
}

// SellWhen adds a new sell order condition to the scheduler.
func (s *Scheduler) SellWhen(size float64, condition func(df *core.Dataframe) bool) {
	s.addOrderCondition(core.SideTypeSell, size, condition)
}

// BuyWhen adds a new buy order condition to the scheduler.
func (s *Scheduler) BuyWhen(size float64, condition func(df *core.Dataframe) bool) {
	s.addOrderCondition(core.SideTypeBuy, size, condition)
}

// addOrderCondition adds a new order condition with the specified parameters.
func (s *Scheduler) addOrderCondition(side core.SideType, size float64, condition func(df *core.Dataframe) bool) {
	s.orderConditions = append(
		s.orderConditions,
		OrderCondition{
			Condition: condition,
			Size:      size,
			Side:      side,
		},
	)
}

// Update evaluates all order conditions against the current dataframe and executes orders when conditions are met.
func (s *Scheduler) Update(df *core.Dataframe, broker core.Broker) {
	var remainingConditions []OrderCondition

	for _, oc := range s.orderConditions {
		if oc.Condition(df) {
			if err := s.executeOrder(broker, oc); err != nil {
				// Keep the condition if execution failed
				remainingConditions = append(remainingConditions, oc)
			}
			// Condition met and order executed successfully, so don't keep it
		} else {
			// Condition not met, keep it for future evaluations
			remainingConditions = append(remainingConditions, oc)
		}
	}

	s.orderConditions = remainingConditions
}

// executeOrder attempts to execute a market order based on the given order condition.
func (s *Scheduler) executeOrder(broker core.Broker, oc OrderCondition) error {
	_, err := broker.CreateOrderMarket(oc.Side, s.pair, oc.Size)
	if err != nil {
		s.log.Errorf("Failed to execute %s order for %s: %v", oc.Side, s.pair, err)
		return err
	}
	s.log.Infof("Successfully executed %s order for %s with size %f", oc.Side, s.pair, oc.Size)
	return nil
}
