package strategies

import (
	"context"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
	"github.com/raykavin/backnrun/strategy"
)

// TrailingStrategy implements a trailing stop strategy using EMA and SMA crossovers
type TrailingStrategy struct {
	// Configuration parameters
	emaLength      int
	smaLength      int
	minQuoteAmount float64
	minAssetValue  float64

	// State tracking
	trailingStop map[string]*strategy.TrailingStop
	scheduler    map[string]*strategy.Scheduler
}

// NewTrailingStrategy creates a new instance of the TrailingStrategy with default parameters
func NewTrailingStrategy(pairs []string, log core.Logger) *TrailingStrategy {
	str := &TrailingStrategy{
		emaLength:      8,
		smaLength:      21,
		minQuoteAmount: 10.0,
		minAssetValue:  10.0,
		trailingStop:   make(map[string]*strategy.TrailingStop),
		scheduler:      make(map[string]*strategy.Scheduler),
	}

	// Initialize trailing stops and schedulers for each pair
	for _, pair := range pairs {
		str.trailingStop[pair] = strategy.NewTrailingStop()
		str.scheduler[pair] = strategy.NewScheduler(pair, log)
	}

	return str
}

// Timeframe returns the required timeframe for this strategy
func (t TrailingStrategy) Timeframe() string {
	return "4h"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (t TrailingStrategy) WarmupPeriod() int {
	return t.smaLength // Use the longer of the two moving averages
}

// Indicators calculates and returns the indicators used by this strategy
func (t TrailingStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate indicators
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.emaLength)
	df.Metadata["sma_slow"] = indicator.SMA(df.Close, t.smaLength)

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA " + string(rune(t.emaLength+'0')),
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["sma_slow"],
					Name:   "SMA " + string(rune(t.smaLength+'0')),
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the entry logic
func (t TrailingStrategy) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	// Get current position
	asset, quote, err := broker.Position(ctx, df.Pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	currentPrice := df.Close.Last(0)
	assetValue := asset * currentPrice

	// Check for buy signal
	if t.shouldEnter(df, quote, assetValue) {
		t.executeEntry(ctx, df, broker, quote)
	}
}

// shouldEnter checks if entry conditions are met
func (t TrailingStrategy) shouldEnter(df *core.Dataframe, quote, assetValue float64) bool {
	return quote > t.minQuoteAmount && // Enough quote currency to trade
		assetValue < t.minAssetValue && // No significant position yet
		df.Metadata["ema_fast"].Crossover(df.Metadata["sma_slow"]) // EMA crosses above SMA
}

// executeEntry performs the entry operation and starts trailing stop
func (t TrailingStrategy) executeEntry(ctx context.Context, df *core.Dataframe, broker core.Broker, quoteAmount float64) {
	// Execute market buy using all available quote currency
	_, err := broker.CreateOrderMarketQuote(ctx, core.SideTypeBuy, df.Pair, quoteAmount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  df.Pair,
			"side":  core.SideTypeBuy,
			"quote": quoteAmount,
		}).Error(err)
		return
	}

	// Start trailing stop
	currentPrice := df.Close.Last(0)
	lowestPrice := df.Low.Last(0)
	t.trailingStop[df.Pair].Start(currentPrice, lowestPrice)
}

// OnPartialCandle is called for each tick update within a candle
func (t TrailingStrategy) OnPartialCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	// Check if trailing stop is triggered
	trailing := t.trailingStop[df.Pair]
	if trailing != nil && trailing.Update(df.Close.Last(0)) {
		t.executeExit(ctx, df, broker)
	}
}

// executeExit performs the exit operation when trailing stop is triggered
func (t TrailingStrategy) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	// Get current asset position
	asset, _, err := broker.Position(ctx, df.Pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Sell all assets if we have a position
	if asset > 0 {
		_, err = broker.CreateOrderMarket(ctx, core.SideTypeSell, df.Pair, asset)
		if err != nil {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":  df.Pair,
				"side":  core.SideTypeSell,
				"asset": asset,
			}).Error(err)
			return
		}

		// Stop the trailing stop after exit
		t.trailingStop[df.Pair].Stop()
	}
}
