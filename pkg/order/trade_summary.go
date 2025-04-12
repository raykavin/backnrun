package order

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/raykavin/backnrun/pkg/exchange"
)

// TradeSummary collects statistics about trading performance
type TradeSummary struct {
	Pair             string
	WinLong          []float64
	WinLongPercent   []float64
	WinShort         []float64
	WinShortPercent  []float64
	LoseLong         []float64
	LoseLongPercent  []float64
	LoseShort        []float64
	LoseShortPercent []float64
	Volume           float64
}

// Win returns all winning trades (both long and short)
func (s TradeSummary) Win() []float64 {
	return append(s.WinLong, s.WinShort...)
}

// WinPercent returns the percentage gains of all winning trades
func (s TradeSummary) WinPercent() []float64 {
	return append(s.WinLongPercent, s.WinShortPercent...)
}

// Lose returns all losing trades (both long and short)
func (s TradeSummary) Lose() []float64 {
	return append(s.LoseLong, s.LoseShort...)
}

// LosePercent returns the percentage losses of all losing trades
func (s TradeSummary) LosePercent() []float64 {
	return append(s.LoseLongPercent, s.LoseShortPercent...)
}

// Profit calculates the total profit across all trades
func (s TradeSummary) Profit() float64 {
	allTrades := append(s.Win(), s.Lose()...)
	return sumSlice(allTrades)
}

// SQN (System Quality Number) calculates the quality of the trading system
// SQN = sqrt(n) * (average profit / standard deviation)
func (s TradeSummary) SQN() float64 {
	allTrades := append(s.Win(), s.Lose()...)
	totalTrades := float64(len(allTrades))

	if totalTrades == 0 {
		return 0
	}

	avgProfit := s.Profit() / totalTrades

	// Calculate standard deviation
	variance := 0.0
	for _, profit := range allTrades {
		variance += math.Pow(profit-avgProfit, 2)
	}

	stdDev := math.Sqrt(variance / totalTrades)
	if stdDev == 0 {
		return 0 // Avoid division by zero
	}

	return math.Sqrt(totalTrades) * (avgProfit / stdDev)
}

// Payoff calculates the ratio of average win to average loss
func (s TradeSummary) Payoff() float64 {
	winPercentages := s.WinPercent()
	losePercentages := s.LosePercent()

	if len(winPercentages) == 0 || len(losePercentages) == 0 {
		return 0
	}

	avgWin := average(winPercentages)
	avgLoss := average(losePercentages)

	if avgLoss == 0 {
		return 0 // Avoid division by zero
	}

	return avgWin / math.Abs(avgLoss)
}

// ProfitFactor calculates the ratio of gross profits to gross losses
func (s TradeSummary) ProfitFactor() float64 {
	winPercentages := s.WinPercent()
	losePercentages := s.LosePercent()

	if len(losePercentages) == 0 {
		return 0
	}

	grossProfit := sumSlice(winPercentages)
	grossLoss := sumSlice(losePercentages)

	if grossLoss == 0 {
		return 0 // Avoid division by zero
	}

	return grossProfit / math.Abs(grossLoss)
}

// WinPercentage calculates the percentage of winning trades
func (s TradeSummary) WinPercentage() float64 {
	winCount := len(s.Win())
	totalTrades := winCount + len(s.Lose())

	if totalTrades == 0 {
		return 0
	}

	return float64(winCount) / float64(totalTrades) * 100
}

// String formats the trade summary as a text table
func (s TradeSummary) String() string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)

	_, quote := exchange.SplitAssetQuote(s.Pair)

	data := [][]string{
		{"Coin", s.Pair},
		{"Trades", strconv.Itoa(len(s.Lose()) + len(s.Win()))},
		{"Win", strconv.Itoa(len(s.Win()))},
		{"Loss", strconv.Itoa(len(s.Lose()))},
		{"% Win", fmt.Sprintf("%.1f", s.WinPercentage())},
		{"Payoff", fmt.Sprintf("%.1f", s.Payoff()*100)},
		{"Pr.Fact", fmt.Sprintf("%.1f", s.ProfitFactor()*100)},
		{"Profit", fmt.Sprintf("%.4f %s", s.Profit(), quote)},
		{"Volume", fmt.Sprintf("%.4f %s", s.Volume, quote)},
	}

	table.AppendBulk(data)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	table.Render()

	return tableString.String()
}

// SaveReturns writes the return percentages to a file
func (s TradeSummary) SaveReturns(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write winning trade percentages
	for _, value := range s.WinPercent() {
		if _, err = file.WriteString(fmt.Sprintf("%.4f\n", value)); err != nil {
			return err
		}
	}

	// Write losing trade percentages
	for _, value := range s.LosePercent() {
		if _, err = file.WriteString(fmt.Sprintf("%.4f\n", value)); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

// sumSlice returns the sum of all values in a slice
func sumSlice(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}

// average calculates the mean of a slice of float64 values
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sumSlice(values) / float64(len(values))
}
