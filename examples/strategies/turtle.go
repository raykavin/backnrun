package strategies

import (
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

// TurtleStrategy implements the classic Turtle Trading system
// Based on: https://www.investopedia.com/articles/trading/08/turtle-trading.asp
type TurtleStrategy struct {
	// Configuration parameters
	entryPeriod  int
	exitPeriod   int
	positionSize float64 // Percentage of account to risk per trade
}

// NewTurtleStrategy creates a new instance of the TurtleStrategy with default parameters
func NewTurtleStrategy() *TurtleStrategy {
	return &TurtleStrategy{
		entryPeriod:  40,
		exitPeriod:   20,
		positionSize: 0.5, // 50% of available funds
	}
}

// Timeframe returns the required timeframe for this strategy
func (t TurtleStrategy) Timeframe() string {
	return "5m"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (t TurtleStrategy) WarmupPeriod() int {
	return t.entryPeriod // Use the longer of the two periods
}

// Indicators calculates and returns the indicators used by this strategy
func (t TurtleStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate indicators
	df.Metadata["max40"] = indicator.Max(df.Close, t.entryPeriod)
	df.Metadata["low20"] = indicator.Min(df.Close, t.exitPeriod)

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Turtle System",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["max40"],
					Name:   "Entry (Max " + string(rune(t.entryPeriod+'0')) + ")",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["low20"],
					Name:   "Exit (Min " + string(rune(t.exitPeriod+'0')) + ")",
					Color:  "red",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (t *TurtleStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	closePrice := df.Close.Last(0)
	highest := df.Metadata["max40"].Last(0)
	lowest := df.Metadata["low20"].Last(0)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Check for entry signal: breakout to new 40-period high
	if t.shouldEnter(assetPosition, closePrice, highest) {
		t.executeEntry(df, broker, quotePosition)
		return
	}

	// Check for exit signal: breakdown to new 20-period low
	if t.shouldExit(assetPosition, closePrice, lowest) {
		t.executeExit(df, broker, assetPosition)
	}
}

// shouldEnter checks if entry conditions are met
func (t *TurtleStrategy) shouldEnter(assetPosition float64, closePrice, highest float64) bool {
	// No position and price breaks above the highest high of the last N periods
	return assetPosition == 0 && closePrice >= highest
}

// shouldExit checks if exit conditions are met
func (t *TurtleStrategy) shouldExit(assetPosition float64, closePrice, lowest float64) bool {
	// Has position and price breaks below the lowest low of the last N periods
	return assetPosition > 0 && closePrice <= lowest
}

// executeEntry performs the entry operation
func (t *TurtleStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition float64) {
	// Calculate position size (half of available quote currency)
	entryAmount := quotePosition * t.positionSize

	// Execute market buy order
	_, err := broker.CreateOrderMarketQuote(core.SideTypeBuy, df.Pair, entryAmount)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  df.Pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": df.Close.Last(0),
		}).Error(err)
	}
}

// executeExit performs the exit operation
func (t *TurtleStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	// Sell entire position
	_, err := broker.CreateOrderMarket(core.SideTypeSell, df.Pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  df.Pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error(err)
	}
}
