package strategies

import (
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
	"github.com/raykavin/backnrun/pkg/strategy"
)

// Williams91Strategy implements Larry Williams' 9.1 trading setup
// The setup looks for a 9-day low followed by a higher close
type Williams91Strategy struct {
	// Configuration parameters
	lookbackPeriod int     // Period for determining lowest low (typically 9)
	exitBars       int     // Number of bars to hold position (typically 4)
	positionSize   float64 // Percentage of account to risk per trade
	atrPeriod      int     // Period for ATR calculation
	atrMultiplier  float64 // Multiplier for ATR to set stop loss

	// Internal state tracking
	barsSinceEntry map[string]int   // Tracks bars since entry for each pair
	activeOrders   map[string]int64 // Tracks active stop orders by pair
}

// NewWilliams91Strategy creates a new instance of the Williams91Strategy with default parameters
func NewWilliams91Strategy() *Williams91Strategy {
	return &Williams91Strategy{
		lookbackPeriod: 9,
		exitBars:       4,
		positionSize:   0.5, // 50% of available funds
		atrPeriod:      14,
		atrMultiplier:  2.0,
		barsSinceEntry: make(map[string]int),
		activeOrders:   make(map[string]int64),
	}
}

// Timeframe returns the required timeframe for this strategy
func (w Williams91Strategy) Timeframe() string {
	return "1d" // Daily timeframe is typically used for this strategy
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (w Williams91Strategy) WarmupPeriod() int {
	// Need enough data for lookback period, ATR, and some extra bars
	if w.lookbackPeriod > w.atrPeriod {
		return w.lookbackPeriod + 10
	}
	return w.atrPeriod + 10
}

// Indicators calculates and returns the indicators used by this strategy
func (w Williams91Strategy) Indicators(df *core.Dataframe) []strategy.ChartIndicator {
	// Calculate indicators
	df.Metadata["lowest_low"] = indicator.Min(df.Low, w.lookbackPeriod)
	df.Metadata["atr"] = indicator.ATR(df.High, df.Low, df.Close, w.atrPeriod)

	// Return chart indicators for visualization
	return []strategy.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Williams 9.1",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["lowest_low"],
					Name:   "9-Day Low",
					Color:  "red",
					Style:  strategy.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ATR",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["atr"],
					Name:   "ATR(" + string(rune(w.atrPeriod+'0')) + ")",
					Color:  "purple",
					Style:  strategy.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (w *Williams91Strategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	lowPrice := df.Low.Last(0)
	atr := df.Metadata["atr"].Last(0)
	lowestLow := df.Metadata["lowest_low"].Last(1) // Previous day's lowest low

	// Get current position
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.Log.Error(err)
		return
	}

	// Update bar counter for this pair if we're in a position
	if assetPosition > 0 {
		w.barsSinceEntry[pair]++
	} else {
		w.barsSinceEntry[pair] = 0
	}

	// Check for exit conditions: either hit the target exit bar count or stop loss
	if assetPosition > 0 {
		if w.shouldExit(pair, df) {
			w.executeExit(df, broker, assetPosition)
			return
		}
	}

	// Check for entry conditions: today's low equals 9-day low and close is higher
	if w.shouldEnter(closePrice, lowPrice, lowestLow, assetPosition) {
		w.executeEntry(df, broker, quotePosition, atr)
	}
}

// shouldEnter checks if entry conditions are met
func (w *Williams91Strategy) shouldEnter(closePrice, lowPrice, lowestLow, assetPosition float64) bool {
	// No current position, today's low is at or near the 9-day low, and close is higher
	isNearLowestLow := lowPrice <= lowestLow*1.005 // Within 0.5% of lowest low
	isHigherClose := closePrice > lowPrice*1.01    // Close at least 1% above the low

	return assetPosition == 0 && isNearLowestLow && isHigherClose
}

// shouldExit checks if exit conditions are met
func (w *Williams91Strategy) shouldExit(pair string, df *core.Dataframe) bool {
	// Exit after holding for the target number of bars
	return w.barsSinceEntry[pair] >= w.exitBars
}

// executeEntry performs the entry operation
func (w *Williams91Strategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, atr float64) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Calculate position size based on available quote currency
	entryAmount := quotePosition * w.positionSize

	// Execute market buy order
	_, err := broker.CreateOrderMarketQuote(core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		backnrun.Log.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Reset the bar counter for this pair
	w.barsSinceEntry[pair] = 0

	// Set a stop loss order based on ATR
	stopPrice := closePrice - (atr * w.atrMultiplier)
	assetPosition, _, err := broker.Position(pair)
	if err != nil {
		backnrun.Log.Error(err)
		return
	}

	// Create a stop loss order
	stopOrder, err := broker.CreateOrderStop(pair, assetPosition, stopPrice)
	if err != nil {
		backnrun.Log.WithFields(map[string]interface{}{
			"pair":      pair,
			"asset":     assetPosition,
			"stopPrice": stopPrice,
		}).Error(err)
	} else {
		// Store the stop order ID for future reference
		w.activeOrders[pair] = stopOrder.ID
	}
}

// executeExit performs the exit operation
func (w *Williams91Strategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair

	if orderID, exists := w.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			err = broker.Cancel(order)
			if err != nil {
				backnrun.Log.WithFields(map[string]interface{}{
					"pair":    pair,
					"orderID": orderID,
				}).Error(err)
			}
		}
		delete(w.activeOrders, pair)
	}

	// Sell entire position
	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetPosition)
	if err != nil {
		backnrun.Log.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error(err)
	}

	// Reset the bar counter
	w.barsSinceEntry[pair] = 0
}
