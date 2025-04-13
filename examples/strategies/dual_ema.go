package strategies

import (
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

// DualEMAStrategy implements a trading strategy using EMAs, MACD and ADX
// to identify trading opportunities
type DualEMAStrategy struct {
	// Configuration parameters
	emaPeriod    int     // Period for both EMAs
	positionSize float64 // Percentage of available capital to use per trade
	stopLoss     float64 // Stop loss percentage

	// MACD parameters
	macdFastPeriod   int
	macdSlowPeriod   int
	macdSignalPeriod int

	// ADX parameters
	adxPeriod int
	diPeriod  int

	// Threshold for ADX and DI
	adxThreshold float64

	// Internal tracking
	activeOrders map[string]int64 // Tracks active stop orders
	inLongTrade  map[string]bool  // Tracks if we're in a long position
	inShortTrade map[string]bool  // Tracks if we're in a short position
}

// NewDualEMAStrategy creates a new instance of the strategy with default parameters
func NewDualEMAStrategy() *DualEMAStrategy {
	return &DualEMAStrategy{
		emaPeriod:        34,
		positionSize:     0.2,  // 20% of available capital - good balance for performance
		stopLoss:         0.02, // 2% stop loss
		macdFastPeriod:   26,   // Fast period for MACD
		macdSlowPeriod:   14,   // Slow period for MACD
		macdSignalPeriod: 14,   // Signal period for MACD
		adxPeriod:        9,    // ADX period
		diPeriod:         9,    // DI period
		adxThreshold:     25.0, // Threshold for ADX and DI
		activeOrders:     make(map[string]int64),
		inLongTrade:      make(map[string]bool),
		inShortTrade:     make(map[string]bool),
	}
}

// Timeframe returns the required timeframe for this strategy
func (d DualEMAStrategy) Timeframe() string {
	return "5m" // 1 hour timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (d DualEMAStrategy) WarmupPeriod() int {
	// Use the maximum between EMA, MACD and ADX periods
	// MACD needs slow period + signal period
	macdPeriod := d.macdSlowPeriod + d.macdSignalPeriod

	// Return the largest period to ensure all indicators are ready
	maxPeriod := d.emaPeriod
	if macdPeriod > maxPeriod {
		maxPeriod = macdPeriod
	}
	if d.adxPeriod > maxPeriod {
		maxPeriod = d.adxPeriod
	}

	// Add a safety margin
	return maxPeriod * 2
}

// Indicators calculates and returns the indicators used by this strategy
func (d DualEMAStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate EMA on highs
	df.Metadata["ema_high"] = indicator.EMA(df.High, d.emaPeriod)

	// Calculate EMA on lows
	df.Metadata["ema_low"] = indicator.EMA(df.Low, d.emaPeriod)

	// Calculate MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		d.macdFastPeriod,
		d.macdSlowPeriod,
		d.macdSignalPeriod,
	)

	// Calculate ADX and DI
	df.Metadata["di_plus"], df.Metadata["di_minus"] = indicator.PlusDI(
		df.High,
		df.Low,
		df.Close,
		d.diPeriod,
	), indicator.MinusDI(
		df.High,
		df.Low,
		df.Close,
		d.diPeriod,
	)

	df.Metadata["adx"] = indicator.ADX(
		df.High,
		df.Low,
		df.Close,
		d.adxPeriod,
	)

	// Return indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Dual EMA Strategy",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_high"],
					Name:   "EMA High",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_low"],
					Name:   "EMA Low",
					Color:  "green",
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
					Values: df.Metadata["macd"],
					Name:   "MACD Line",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_signal"],
					Name:   "Signal Line",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_hist"],
					Name:   "Histogram",
					Color:  "green",
					Style:  core.StyleHistogram,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ADX",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["adx"],
					Name:   "ADX",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["di_plus"],
					Name:   "DI+",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["di_minus"],
					Name:   "DI-",
					Color:  "red",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (d *DualEMAStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair

	// Safety check - make sure we have enough data points
	if df.Close.Length() < 2 ||
		df.Metadata["ema_high"].Length() < 2 ||
		df.Metadata["ema_low"].Length() < 2 ||
		df.Metadata["macd"].Length() < 1 ||
		df.Metadata["di_plus"].Length() < 1 ||
		df.Metadata["di_minus"].Length() < 1 ||
		df.Metadata["adx"].Length() < 1 {
		backnrun.DefaultLog.Debug("Not enough data points for strategy calculation, skipping candle")
		return
	}

	// Current and previous prices
	closePrice := df.Close.Last(0)
	prevClosePrice := df.Close.Last(1)

	// EMAs
	emaHigh := df.Metadata["ema_high"].Last(0)
	prevEmaHigh := df.Metadata["ema_high"].Last(1)
	emaLow := df.Metadata["ema_low"].Last(0)
	prevEmaLow := df.Metadata["ema_low"].Last(1)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Check if we're in a position
	inLong := d.inLongTrade[pair]
	inShort := d.inShortTrade[pair]

	// Check exit signals before entry to avoid double operations
	if inLong {
		// Check exit from long position
		if closePrice < emaHigh {
			d.executeExitLong(df, broker, assetPosition)
			return
		}
	} else if inShort {
		// Check exit from short position
		if closePrice > emaLow {
			d.executeExitShort(df, broker, assetPosition)
			return
		}
	}

	// Check entry signals
	if !inLong && !inShort {
		// Get indicator values
		macd := df.Metadata["macd"].Last(0)
		adx := df.Metadata["adx"].Last(0)
		diPlus := df.Metadata["di_plus"].Last(0)
		diMinus := df.Metadata["di_minus"].Last(0)

		// Log trading conditions for debugging
		backnrun.DefaultLog.Debugf("Trading conditions for %s - ema_high: %.2f, ema_low: %.2f, macd: %.2f, adx: %.2f, di+: %.2f, di-: %.2f",
			pair, emaHigh, emaLow, macd, adx, diPlus, diMinus)

		// Check buy entry with updated conditions
		if prevClosePrice > prevEmaHigh &&
			macd > 0 && // MACD above 0 axis
			adx > d.adxThreshold && // ADX above threshold
			diPlus > d.adxThreshold && // DI+ above threshold
			diPlus > diMinus { // DI+ greater than DI-
			backnrun.DefaultLog.Infof("Buy signal detected for %s", pair)
			d.executeEntryLong(df, broker, quotePosition, closePrice)
			return
		}

		// Check sell entry with updated conditions
		if prevClosePrice < prevEmaLow &&
			macd < 0 && // MACD below 0 axis
			adx > d.adxThreshold && // ADX above threshold
			diMinus > d.adxThreshold && // DI- above threshold
			diMinus > diPlus { // DI- greater than DI+
			backnrun.DefaultLog.Infof("Sell signal detected for %s", pair)
			d.executeEntryShort(df, broker, quotePosition, closePrice)
			return
		}
	}
}

// executeEntryLong executes a long position entry
func (d *DualEMAStrategy) executeEntryLong(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Calculate position size based on available capital
	entryAmount := quotePosition * d.positionSize

	// Add safety checks to prevent excessive position sizes
	maxPositionValue := 5000.0 // Maximum position value in quote currency
	if entryAmount > maxPositionValue {
		entryAmount = maxPositionValue
		backnrun.DefaultLog.Infof("Position size capped at %.2f for %s", maxPositionValue, pair)
	}

	// Add minimum position check
	minPositionValue := 10.0 // Minimum position value in quote currency
	if entryAmount < minPositionValue {
		backnrun.DefaultLog.Infof("Skipping trade - position size too small (%.2f < %.2f)", entryAmount, minPositionValue)
		return
	}

	// Execute market buy order
	_, err := broker.CreateOrderMarketQuote(core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Get updated position after purchase
	assetPosition, _, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Create stop loss order
	stopPrice := closePrice * (1.0 - d.stopLoss)
	stopOrder, err := broker.CreateOrderStop(pair, assetPosition, stopPrice)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":      pair,
			"asset":     assetPosition,
			"stopPrice": stopPrice,
		}).Error(err)
	} else {
		// Store stop order ID for future reference
		d.activeOrders[pair] = stopOrder.ID
	}

	// Update position states
	d.inLongTrade[pair] = true
	d.inShortTrade[pair] = false
}

// executeEntryShort executes a short position entry
func (d *DualEMAStrategy) executeEntryShort(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Calculate position size based on available capital
	entryAmount := quotePosition * d.positionSize

	// Add safety checks to prevent excessive position sizes
	maxPositionValue := 5000.0 // Maximum position value in quote currency
	if entryAmount > maxPositionValue {
		entryAmount = maxPositionValue
		backnrun.DefaultLog.Infof("Position size capped at %.2f for %s", maxPositionValue, pair)
	}

	// Add minimum position check
	minPositionValue := 10.0 // Minimum position value in quote currency
	if entryAmount < minPositionValue {
		backnrun.DefaultLog.Infof("Skipping trade - position size too small (%.2f < %.2f)", entryAmount, minPositionValue)
		return
	}

	// Calculate asset quantity based on quote amount
	assetAmount := entryAmount / closePrice

	// For simplicity, we assume we can sell short
	// In practice, this would depend on broker and market capabilities
	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetAmount)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Create stop loss order (for short position, it's a buy stop)
	stopPrice := closePrice * (1.0 + d.stopLoss)
	stopOrder, err := broker.CreateOrderStop(pair, assetAmount, stopPrice)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":      pair,
			"asset":     assetAmount,
			"stopPrice": stopPrice,
		}).Error(err)
	} else {
		// Store stop order ID for future reference
		d.activeOrders[pair] = stopOrder.ID
	}

	// Update position states
	d.inLongTrade[pair] = false
	d.inShortTrade[pair] = true
}

// executeExitLong executes a long position exit
func (d *DualEMAStrategy) executeExitLong(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair

	// Cancel stop loss order if active
	if orderID, exists := d.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			err = broker.Cancel(order)
			if err != nil {
				backnrun.DefaultLog.WithFields(map[string]interface{}{
					"pair":    pair,
					"orderID": orderID,
				}).Error(err)
			}
		}
		// Remove order reference
		delete(d.activeOrders, pair)
	}

	// Sell entire position
	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error(err)
	}

	// Update position state
	d.inLongTrade[pair] = false
}

// executeExitShort executes a short position exit
func (d *DualEMAStrategy) executeExitShort(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair

	// Cancel stop loss order if active
	if orderID, exists := d.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			err = broker.Cancel(order)
			if err != nil {
				backnrun.DefaultLog.WithFields(map[string]interface{}{
					"pair":    pair,
					"orderID": orderID,
				}).Error(err)
			}
		}
		// Remove order reference
		delete(d.activeOrders, pair)
	}

	// To exit a short position, we need to buy
	// In this case, assetPosition would be the absolute value of the short position
	_, err := broker.CreateOrderMarket(core.SideTypeBuy, pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error(err)
	}

	// Update position state
	d.inShortTrade[pair] = false
}
