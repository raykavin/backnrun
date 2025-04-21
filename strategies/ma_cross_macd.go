package strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

// CrossEMAMACD implements a trading strategy using EMA, SMA, and MACD
// for entry and exit signals
type CrossEMAMACD struct {
	// Configuration parameters
	emaLength      int
	smaLength      int
	macdLongLine   int
	macdShortLine  int
	macdSignalLine int
	emaName        string
	smaName        string
	macdName       string
	macdSignalName string
	macdHistName   string
	minQuoteAmount float64
}

// NewCrossEMAMACD creates a new instance of the CrossEMAMACD strategy with default parameters
func NewCrossEMAMACD(emaLength, smaLength, macdLongLine, macdShortLine, macdSignalLine int, minQuoteAmount float64) *CrossEMAMACD {
	crossMACD := &CrossEMAMACD{
		emaLength:      9,
		smaLength:      21,
		macdLongLine:   150,
		macdShortLine:  14,
		macdSignalLine: 14,
		minQuoteAmount: 10.0,
	}

	// Override defaults with provided values if valid
	if emaLength > 0 {
		crossMACD.emaLength = emaLength
	}
	if smaLength > 0 {
		crossMACD.smaLength = smaLength
	}
	if macdLongLine > 0 {
		crossMACD.macdLongLine = macdLongLine
	}
	if macdShortLine > 0 {
		crossMACD.macdShortLine = macdShortLine
	}
	if macdSignalLine > 0 {
		crossMACD.macdSignalLine = macdSignalLine
	}
	if minQuoteAmount > 0 {
		crossMACD.minQuoteAmount = minQuoteAmount
	}

	// Set indicator names for dataframe metadata
	crossMACD.emaName = fmt.Sprintf("ema%d", crossMACD.emaLength)
	crossMACD.smaName = fmt.Sprintf("sma%d", crossMACD.smaLength)
	crossMACD.macdName = fmt.Sprintf("macd%d_%d", crossMACD.macdShortLine, crossMACD.macdLongLine)
	crossMACD.macdSignalName = fmt.Sprintf("macdsignal%d_%d_%d", crossMACD.macdShortLine, crossMACD.macdLongLine, crossMACD.macdSignalLine)
	crossMACD.macdHistName = fmt.Sprintf("macdhist%d_%d_%d", crossMACD.macdShortLine, crossMACD.macdLongLine, crossMACD.macdSignalLine)

	return crossMACD
}

// Timeframe returns the required timeframe for this strategy
func (s CrossEMAMACD) Timeframe() string {
	return "5m"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s CrossEMAMACD) WarmupPeriod() int {
	// Return the maximum period needed for any of our indicators
	// MACD with a long line of 150 will require more data than the original strategy
	return 300
}

// Indicators calculates and returns the indicators used by this strategy
func (s CrossEMAMACD) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate EMA and SMA indicators
	df.Metadata[s.emaName] = indicator.EMA(df.Close, s.emaLength)
	df.Metadata[s.smaName] = indicator.SMA(df.Close, s.smaLength)

	// Calculate MACD indicators
	macd, signal, hist := indicator.MACD(df.Close, s.macdShortLine, s.macdLongLine, s.macdSignalLine)
	df.Metadata[s.macdName] = macd
	df.Metadata[s.macdSignalName] = signal
	df.Metadata[s.macdHistName] = hist

	// Create zero line for MACD reference
	zeroLine := make([]float64, len(df.Close))

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "MA's",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata[s.emaName],
					Name:   strings.ToUpper(s.emaName),
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata[s.smaName],
					Name:   strings.ToUpper(s.smaName),
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "MACD",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata[s.macdName],
					Name:   "MACD",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata[s.macdSignalName],
					Name:   "Signal",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata[s.macdHistName],
					Name:   "Histogram",
					Color:  "green",
					Style:  core.StyleHistogram,
				},
				{
					Values: zeroLine,
					Name:   "Zero",
					Color:  "gray",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (s *CrossEMAMACD) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	closePrice := df.Close.Last(0)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(ctx, df.Pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Check if we have any position (long or short)
	hasPosition := assetPosition != 0
	hasLongPosition := assetPosition > 0
	hasShortPosition := assetPosition < 0

	// Handle existing positions first
	if hasPosition {
		// If we have a long position, check if we should exit
		if hasLongPosition && s.shouldExitLong(df) {
			s.executeExitLong(ctx, df, broker, assetPosition)
			return
		}

		if hasShortPosition && s.shouldExitShort(df) {
			s.executeExitShort(ctx, df, broker, assetPosition)
			return
		}

		// If we have any position and didn't exit, we don't open new positions
		return
	}

	// We only reach here if we don't have any position
	// Check for buy signal (long entry)
	if s.shouldEnterLong(df, quotePosition) {
		s.executeEnterLong(ctx, df, broker, closePrice, quotePosition)
		return
	}

	if s.shouldEnterShort(df) {
		s.executeEnterShort(ctx, df, broker, closePrice)
		return
	}
}

// shouldEnterLong checks if conditions for a long entry are met
func (s *CrossEMAMACD) shouldEnterLong(df *core.Dataframe, quotePosition float64) bool {
	// Check if we have enough funds to trade
	if quotePosition < s.minQuoteAmount {
		return false
	}

	// Get MACD value
	macdLine := df.Metadata[s.macdName].Last(0)

	// Buy conditions:
	// 1. MACD line value is above the 0 axis
	// 2. EMA is above the SMA
	return macdLine > 0 &&
		df.Metadata[s.emaName].Last(0) > df.Metadata[s.smaName].Last(0)
}

// shouldEnterShort checks if conditions for a short entry are met
func (s *CrossEMAMACD) shouldEnterShort(df *core.Dataframe) bool {
	// Get MACD value
	macdLine := df.Metadata[s.macdName].Last(0)

	// Sell conditions (short):
	// 1. MACD line value is below the 0 axis
	// 2. EMA is below the SMA
	return macdLine < 0 &&
		df.Metadata[s.emaName].Last(0) < df.Metadata[s.smaName].Last(0)
}

// shouldExitLong checks if we should exit a long position
func (s *CrossEMAMACD) shouldExitLong(df *core.Dataframe) bool {
	// Get MACD value
	macdLine := df.Metadata[s.macdName].Last(0)
	macdPrevLine := df.Metadata[s.macdName].Last(1)

	// For a buy position, exit when:
	// 1. The MACD crosses below 0 or
	// 2. The EMA crosses below the SMA
	return (macdPrevLine >= 0 && macdLine < 0) || // MACD crossed below 0
		df.Metadata[s.emaName].Crossunder(df.Metadata[s.smaName]) // EMA crossed below SMA
}

// shouldExitShort checks if we should exit a short position
func (s *CrossEMAMACD) shouldExitShort(df *core.Dataframe) bool {
	// Get MACD value
	macdLine := df.Metadata[s.macdName].Last(0)
	macdPrevLine := df.Metadata[s.macdName].Last(1)

	// For a sell position, exit when:
	// 1. The MACD crosses above 0 or
	// 2. The EMA crosses above the SMA
	return (macdPrevLine <= 0 && macdLine > 0) || // MACD crossed above 0
		df.Metadata[s.emaName].Crossover(df.Metadata[s.smaName]) // EMA crossed above SMA
}

// executeEnterLong performs the buy operation for a long entry
func (s *CrossEMAMACD) executeEnterLong(ctx context.Context, df *core.Dataframe, broker core.Broker, closePrice, quotePosition float64) {
	amount := quotePosition / closePrice // calculate amount of asset to buy
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeBuy, df.Pair, amount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  df.Pair,
			"side":  core.SideTypeBuy,
			"price": closePrice,
			"size":  amount,
		}).Error(err)
	}
}

// executeExitLong performs the sell operation to exit a long position
func (s *CrossEMAMACD) executeExitLong(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, df.Pair, assetPosition)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair": df.Pair,
			"side": core.SideTypeSell,
			"size": assetPosition,
		}).Error(err)
	}
}

// executeEnterShort performs the sell operation for a short entry
func (s *CrossEMAMACD) executeEnterShort(ctx context.Context, df *core.Dataframe, broker core.Broker, closePrice float64) {
	// Note: This is a simplified implementation, actual short selling depends on your broker's API
	// and might require borrowing assets first
	amount := s.minQuoteAmount / closePrice // calculate amount of asset to short
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, df.Pair, amount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  df.Pair,
			"side":  core.SideTypeSell,
			"price": closePrice,
			"size":  amount,
		}).Error(err)
	}
}

// executeExitShort performs the buy operation to exit a short position
func (s *CrossEMAMACD) executeExitShort(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	// Note: For short positions, assetPosition will be negative, so we need to use the absolute value
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeBuy, df.Pair, -assetPosition)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair": df.Pair,
			"side": core.SideTypeBuy,
			"size": -assetPosition,
		}).Error(err)
	}
}
