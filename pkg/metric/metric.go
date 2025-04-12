package metric

import (
	"math"

	"gonum.org/v1/gonum/stat"
)

// Mean calculates the arithmetic mean of the values.
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return stat.Mean(values, nil)
}

// Payoff calculates the ratio of average wins to average losses.
// Returns the absolute value of the ratio.
func Payoff(values []float64) float64 {
	wins, losses := partitionTradeResults(values)

	if len(losses) == 0 {
		return 10 // Default value when no losses
	}

	avgWin := stat.Mean(wins, nil)
	avgLoss := stat.Mean(losses, nil)

	if avgLoss == 0 {
		return 10 // Prevent division by zero
	}

	return math.Abs(avgWin / avgLoss)
}

// ProfitFactor calculates the ratio of total profits to total losses.
// Returns the absolute value of the ratio.
func ProfitFactor(values []float64) float64 {
	var (
		totalWins   float64
		totalLosses float64
	)

	for _, value := range values {
		if value >= 0 {
			totalWins += value
		} else {
			totalLosses += value
		}
	}

	if totalLosses == 0 {
		return 10 // Default value when no losses
	}

	return math.Abs(totalWins / totalLosses)
}

// partitionTradeResults separates trading results into wins and losses.
func partitionTradeResults(values []float64) (wins []float64, losses []float64) {
	for _, value := range values {
		if value >= 0 {
			wins = append(wins, value)
		} else {
			losses = append(losses, math.Abs(value)) // Store absolute values of losses
		}
	}
	return wins, losses
}
