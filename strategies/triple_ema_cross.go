package strategies

import (
	"context"
	"fmt"
	"sync"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

// TripleMAStrategy implements a trading strategy based on the crossing of three
// moving averages to identify strong trends and entry/exit points
type TripleMAStrategy struct {
	// Configuration parameters
	config TripleMAConfig

	// Internal tracking
	activeOrders map[string]int64   // Maps trading pair to active stop order ID
	entryPrices  map[string]float64 // Tracks entry prices for calculating P&L
	mu           sync.Mutex         // Protects concurrent access to maps
}

// TripleMAConfig holds all configuration parameters for the strategy
type TripleMAConfig struct {
	ShortPeriod  int     // Period for short-term EMA
	MediumPeriod int     // Period for medium-term EMA
	LongPeriod   int     // Period for long-term EMA
	PositionSize float64 // Percentage of available capital to use (0.0-1.0)
	StopLoss     float64 // Stop loss percentage from entry price
	TakeProfit   float64 // Take profit percentage from entry price (0 for none)
	Timeframe    string  // Candlestick timeframe to use
}

// DefaultTripleMAConfig returns a configuration with sensible defaults
func DefaultTripleMAConfig() TripleMAConfig {
	return TripleMAConfig{
		ShortPeriod:  9,
		MediumPeriod: 21,
		LongPeriod:   50,
		PositionSize: 0.5,  // 50% of available capital
		StopLoss:     0.05, // 5% stop loss
		TakeProfit:   0.15, // 15% take profit (new feature)
		Timeframe:    "5m",
	}
}

// NewTripleMAStrategy creates a new instance of the strategy with default parameters
func NewTripleMAStrategy() *TripleMAStrategy {
	return NewTripleMAStrategyWithConfig(DefaultTripleMAConfig())
}

// NewTripleMAStrategyWithConfig creates a new instance with custom configuration
func NewTripleMAStrategyWithConfig(config TripleMAConfig) *TripleMAStrategy {
	validateConfig(&config)

	return &TripleMAStrategy{
		config:       config,
		activeOrders: make(map[string]int64),
		entryPrices:  make(map[string]float64),
	}
}

// validateConfig validates and adjusts configuration parameters
func validateConfig(config *TripleMAConfig) {
	// Ensure periods are in correct ascending order
	if config.ShortPeriod >= config.MediumPeriod {
		config.ShortPeriod = config.MediumPeriod - 2
	}
	if config.MediumPeriod >= config.LongPeriod {
		config.MediumPeriod = config.LongPeriod - 5
	}

	// Limit position size between 0.1 and 1.0
	if config.PositionSize < 0.1 {
		config.PositionSize = 0.1
	} else if config.PositionSize > 1.0 {
		config.PositionSize = 1.0
	}

	// Ensure stop loss is reasonable
	if config.StopLoss < 0.01 {
		config.StopLoss = 0.01
	} else if config.StopLoss > 0.2 {
		config.StopLoss = 0.2
	}
}

// Timeframe returns the required timeframe for this strategy
func (t *TripleMAStrategy) Timeframe() string {
	return t.config.Timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (t *TripleMAStrategy) WarmupPeriod() int {
	// Use the longest period plus margin for reliable indicator values
	return t.config.LongPeriod + 10
}

// Indicators calculates and returns the indicators used by this strategy
func (t *TripleMAStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate the three moving averages
	df.Metadata["short_ma"] = indicator.EMA(df.Close, t.config.ShortPeriod)
	df.Metadata["medium_ma"] = indicator.EMA(df.Close, t.config.MediumPeriod)
	df.Metadata["long_ma"] = indicator.EMA(df.Close, t.config.LongPeriod)

	// Return indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["short_ma"],
					Name:   fmt.Sprintf("EMA(%d)", t.config.ShortPeriod),
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["medium_ma"],
					Name:   fmt.Sprintf("EMA(%d)", t.config.MediumPeriod),
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["long_ma"],
					Name:   fmt.Sprintf("EMA(%d)", t.config.LongPeriod),
					Color:  "red",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (t *TripleMAStrategy) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	logger := bot.DefaultLog.WithFields(map[string]any{"strategy": "TripleMA", "pair": pair})

	// Get moving average values
	shortMA := df.Metadata["short_ma"].Last(0)
	mediumMA := df.Metadata["medium_ma"].Last(0)
	longMA := df.Metadata["long_ma"].Last(0)

	// Get previous values to detect crossovers
	prevShortMA := df.Metadata["short_ma"].Last(1)
	prevMediumMA := df.Metadata["medium_ma"].Last(1)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(ctx, pair)
	if err != nil {
		logger.Error("Failed to get position: ", err)
		return
	}

	// Check entry and exit signals
	if assetPosition > 0 {
		// We have a position, check if we should exit
		t.mu.Lock()
		entryPrice, hasEntryPrice := t.entryPrices[pair]
		t.mu.Unlock()

		// Check for take profit condition if we have the entry price
		if hasEntryPrice && t.config.TakeProfit > 0 {
			profitPercent := (closePrice - entryPrice) / entryPrice
			if profitPercent >= t.config.TakeProfit {
				logger.Info("Take profit triggered")
				t.executeExit(ctx, df, broker, assetPosition, "take_profit")
				return
			}
		}

		// Check for exit based on MA crossover
		if t.shouldExit(shortMA, mediumMA, prevShortMA, prevMediumMA) {
			logger.Info("MA crossover exit signal")
			t.executeExit(ctx, df, broker, assetPosition, "ma_crossover")
		}
	} else {
		// No position, check if we should enter
		if t.shouldEnter(shortMA, mediumMA, longMA, prevShortMA, prevMediumMA) {
			// Also check for additional confirmation like volume increase or RSI
			if t.confirmEntrySignal(df) {
				logger.Info("Entry signal confirmed")
				t.executeEntry(ctx, df, broker, quotePosition, closePrice)
			}
		}
	}
}

// shouldEnter checks if entry conditions are met
func (t *TripleMAStrategy) shouldEnter(shortMA, mediumMA, longMA, prevShortMA, prevMediumMA float64) bool {
	// Enter when short MA crosses above medium MA
	// And both are above long MA (uptrend)
	crossedAbove := prevShortMA <= prevMediumMA && shortMA > mediumMA
	allAligned := shortMA > mediumMA && mediumMA > longMA

	return crossedAbove && allAligned
}

// shouldExit checks if exit conditions are met
func (t *TripleMAStrategy) shouldExit(shortMA, mediumMA, prevShortMA, prevMediumMA float64) bool {
	// Exit when short MA crosses below medium MA
	return prevShortMA >= prevMediumMA && shortMA < mediumMA
}

// confirmEntrySignal applies additional conditions to confirm entry signal
// This helps filter out false signals and improve win rate
func (t *TripleMAStrategy) confirmEntrySignal(df *core.Dataframe) bool {
	// Check if we have enough volume to confirm the trend
	// Simple example: volume should be higher than the previous candle
	if len(df.Volume) >= 2 {
		currentVolume := df.Volume.Last(0)
		previousVolume := df.Volume.Last(1)

		// Volume should be increasing
		if currentVolume <= previousVolume {
			return false
		}
	}

	// We could add more confirmations like:
	// - RSI is not overbought
	// - Price is not at a key resistance level
	// - ADX shows strong trend

	return true
}

// executeEntry executes the entry operation
func (t *TripleMAStrategy) executeEntry(ctx context.Context, df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair
	logger := bot.DefaultLog.WithFields(map[string]any{
		"strategy": "TripleMA",
		"pair":     pair,
		"action":   "entry",
	})

	// Calculate position size based on available capital
	entryAmount := quotePosition * t.config.PositionSize

	// Execute market buy order
	order, err := broker.CreateOrderMarketQuote(ctx, core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		logger.WithFields(map[string]any{
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error("Failed to create entry order: ", err)
		return
	}

	logger.WithFields(map[string]any{
		"orderId": order.ID,
		"amount":  entryAmount,
		"price":   closePrice,
	}).Info("Entry order executed")

	// Get updated position after purchase
	assetPosition, _, err := broker.Position(ctx, pair)
	if err != nil {
		logger.Error("Failed to get updated position: ", err)
		return
	}

	// Store entry price for this pair
	t.mu.Lock()
	t.entryPrices[pair] = closePrice
	t.mu.Unlock()

	// Create stop loss order
	stopPrice := closePrice * (1.0 - t.config.StopLoss)
	stopOrder, err := broker.CreateOrderStop(ctx, pair, assetPosition, stopPrice)
	if err != nil {
		logger.WithFields(map[string]any{
			"asset":     assetPosition,
			"stopPrice": stopPrice,
		}).Error("Failed to create stop loss order: ", err)
	} else {
		// Store stop order ID for future reference
		t.mu.Lock()
		t.activeOrders[pair] = stopOrder.ID
		t.mu.Unlock()

		logger.WithFields(map[string]any{
			"stopOrderId": stopOrder.ID,
			"stopPrice":   stopPrice,
		}).Info("Stop loss order created")
	}
}

// executeExit executes the exit operation
func (t *TripleMAStrategy) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64, reason string) {
	pair := df.Pair
	logger := bot.DefaultLog.WithFields(map[string]any{
		"strategy": "TripleMA",
		"pair":     pair,
		"action":   "exit",
		"reason":   reason,
	})

	// Cancel the stop loss order if active
	t.mu.Lock()
	orderID, exists := t.activeOrders[pair]
	t.mu.Unlock()

	if exists {
		order, err := broker.Order(ctx, pair, orderID)
		if err == nil && order.Status != core.OrderStatusTypeFilled {
			err = broker.Cancel(ctx, order)
			if err != nil {
				logger.WithFields(map[string]any{
					"orderID": orderID,
				}).Error("Failed to cancel stop loss order: ", err)
			} else {
				logger.WithFields(map[string]any{
					"orderID": orderID,
				}).Info("Stop loss order canceled")
			}
		}

		// Remove order reference
		t.mu.Lock()
		delete(t.activeOrders, pair)
		t.mu.Unlock()
	}

	// Sell entire position
	order, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, assetPosition)
	if err != nil {
		logger.WithFields(map[string]any{
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error("Failed to create exit order: ", err)
		return
	}

	// Calculate profit/loss if we have the entry price
	t.mu.Lock()
	entryPrice, hasEntryPrice := t.entryPrices[pair]
	delete(t.entryPrices, pair)
	t.mu.Unlock()

	closePrice := df.Close.Last(0)
	if hasEntryPrice {
		profitPercent := (closePrice - entryPrice) / entryPrice * 100
		logger.WithFields(map[string]any{
			"orderId":       order.ID,
			"exitPrice":     closePrice,
			"entryPrice":    entryPrice,
			"profitPercent": fmt.Sprintf("%.2f%%", profitPercent),
		}).Info("Position closed")
	} else {
		logger.WithFields(map[string]any{
			"orderId":   order.ID,
			"exitPrice": closePrice,
		}).Info("Position closed")
	}
}

// Reset resets the strategy's internal state
func (t *TripleMAStrategy) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeOrders = make(map[string]int64)
	t.entryPrices = make(map[string]float64)
}

// SetConfig updates the strategy configuration
func (t *TripleMAStrategy) SetConfig(config TripleMAConfig) {
	validateConfig(&config)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.config = config
}

// GetConfig returns the current strategy configuration
func (t *TripleMAStrategy) GetConfig() TripleMAConfig {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.config
}
