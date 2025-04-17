package strategies

import (
	"time"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

// VolatilityBreakoutStrategy implements a strategy that trades breakouts
// after periods of low volatility, with dynamic position sizing and risk management
type VolatilityBreakoutStrategy struct {
	// Volatility parameters
	atrPeriod       int
	atrLookback     int     // Period to look back for volatility contraction
	atrThreshold    float64 // Threshold for volatility contraction
	breakoutFactor  float64 // Multiplier for breakout detection
	
	// Trend filter parameters
	emaPeriod       int
	supertrend1     int
	supertrend2     int
	supertrendMult  float64
	
	// Time filter
	tradingHoursOnly bool
	tradingStartHour int
	tradingEndHour   int
	
	// Position sizing and risk management
	riskPerTrade    float64 // Risk per trade as percentage of account
	maxPositions    int     // Maximum number of concurrent positions
	profitTarget    float64 // Take profit target
	stopMultiplier  float64 // Stop loss multiplier based on ATR
	trailingStart   float64 // When to start trailing (% profit)
	trailingFactor  float64 // Trailing stop factor
	
	// State tracking
	activePositions map[string]bool      // Track active positions by pair
	entryPrices     map[string]float64   // Track entry prices
	stopLosses      map[string]float64   // Track stop loss levels
	highestSinceBuy map[string]float64   // Track highest price since entry
	activeOrders    map[string]int64     // Track active orders
	lastVolatility  map[string][]float64 // Track recent volatility
	lastBreakout    map[string]time.Time // Track last breakout time
}

// NewVolatilityBreakoutStrategy creates a new instance of the strategy
func NewVolatilityBreakoutStrategy() *VolatilityBreakoutStrategy {
	return &VolatilityBreakoutStrategy{
		// Volatility parameters
		atrPeriod:       14,
		atrLookback:     10,
		atrThreshold:    0.7,  // 70% of average volatility indicates contraction
		breakoutFactor:  1.5,  // 1.5x ATR for breakout detection
		
		// Trend filter parameters
		emaPeriod:       50,
		supertrend1:     10,
		supertrend2:     20,
		supertrendMult:  3.0,
		
		// Time filter
		tradingHoursOnly: false,
		tradingStartHour: 8,   // 8 AM
		tradingEndHour:   20,  // 8 PM
		
		// Position sizing and risk management
		riskPerTrade:    0.01, // 1% risk per trade
		maxPositions:    3,    // Maximum 3 concurrent positions
		profitTarget:    0.03, // 3% take profit
		stopMultiplier:  2.0,  // 2x ATR for stop loss
		trailingStart:   0.02, // Start trailing at 2% profit
		trailingFactor:  0.5,  // 50% of ATR for trailing
		
		// State tracking
		activePositions: make(map[string]bool),
		entryPrices:     make(map[string]float64),
		stopLosses:      make(map[string]float64),
		highestSinceBuy: make(map[string]float64),
		activeOrders:    make(map[string]int64),
		lastVolatility:  make(map[string][]float64),
		lastBreakout:    make(map[string]time.Time),
	}
}

// Timeframe returns the required timeframe for this strategy
func (s VolatilityBreakoutStrategy) Timeframe() string {
	return "15m" // 15-minute timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s VolatilityBreakoutStrategy) WarmupPeriod() int {
	// Use the maximum period needed for any indicator plus some buffer
	return s.emaPeriod + 50
}

// Indicators calculates and returns the indicators used by this strategy
func (s VolatilityBreakoutStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate ATR
	df.Metadata["atr"] = talib.Atr(df.High, df.Low, df.Close, s.atrPeriod)
	
	// Calculate ATR percentage (ATR/Price)
	atrPercent := make([]float64, len(df.Close))
	for i := 0; i < len(df.Close); i++ {
		if i < s.atrPeriod-1 {
			atrPercent[i] = 0
		} else if df.Close[i] > 0 {
			atrPercent[i] = df.Metadata["atr"][i] / df.Close[i]
		}
	}
	df.Metadata["atr_percent"] = atrPercent
	
	// Calculate EMA
	df.Metadata["ema"] = indicator.EMA(df.Close, s.emaPeriod)
	
	// Calculate custom trend indicators instead of Supertrend
	// Using simple moving averages instead of ATR-based trend lines
	
	// Calculate trend line 1
	trendLine1 := make([]float64, len(df.Close))
	trendDir1 := make([]float64, len(df.Close))
	
	for i := s.supertrend1; i < len(df.Close); i++ {
		// Basic trend calculation: price vs moving average
		sma := indicator.SMA(df.Close, s.supertrend1)
		trendLine1[i] = sma[i]
		if df.Close[i] > sma[i] {
			trendDir1[i] = 1.0
		} else {
			trendDir1[i] = -1.0
		}
	}
	
	df.Metadata["supertrend1"] = trendLine1
	df.Metadata["supertrend1_dir"] = trendDir1
	
	// Calculate trend line 2
	trendLine2 := make([]float64, len(df.Close))
	trendDir2 := make([]float64, len(df.Close))
	
	for i := s.supertrend2; i < len(df.Close); i++ {
		// Basic trend calculation: price vs moving average
		sma := indicator.SMA(df.Close, s.supertrend2)
		trendLine2[i] = sma[i]
		if df.Close[i] > sma[i] {
			trendDir2[i] = 1.0
		} else {
			trendDir2[i] = -1.0
		}
	}
	
	df.Metadata["supertrend2"] = trendLine2
	df.Metadata["supertrend2_dir"] = trendDir2
	
	// Calculate Bollinger Bands
	upper, middle, lower := talib.BBands(
		df.Close,
		20, // Period
		2,  // Standard deviation
		2,  // Standard deviation
		talib.SMA,
	)
	df.Metadata["bb_upper"] = upper
	df.Metadata["bb_middle"] = middle
	df.Metadata["bb_lower"] = lower
	
	// Calculate Bollinger Band width
	bbWidth := make([]float64, len(df.Close))
	for i := 0; i < len(df.Close); i++ {
		if i < 20-1 {
			bbWidth[i] = 0
		} else if middle[i] > 0 {
			bbWidth[i] = (upper[i] - lower[i]) / middle[i]
		}
	}
	df.Metadata["bb_width"] = bbWidth
	
	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema"],
					Name:   "EMA",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["supertrend1"],
					Name:   "Supertrend 1",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["supertrend2"],
					Name:   "Supertrend 2",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_upper"],
					Name:   "BB Upper",
					Color:  "rgba(76, 175, 80, 0.5)",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_middle"],
					Name:   "BB Middle",
					Color:  "rgba(76, 175, 80, 1)",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_lower"],
					Name:   "BB Lower",
					Color:  "rgba(76, 175, 80, 0.5)",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "Volatility",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["atr_percent"],
					Name:   "ATR %",
					Color:  "orange",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_width"],
					Name:   "BB Width",
					Color:  "purple",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (s *VolatilityBreakoutStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	highPrice := df.High.Last(0)
	lowPrice := df.Low.Last(0)
	
	// Get current position
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}
	
	// Check if we're in trading hours (if enabled)
	if s.tradingHoursOnly {
		currentHour := df.Time[len(df.Time)-1].Hour()
		if currentHour < s.tradingStartHour || currentHour >= s.tradingEndHour {
			return // Outside trading hours
		}
	}
	
	// Update volatility tracking
	s.updateVolatilityTracking(pair, df)
	
	// Check for trailing stop if in position
	if assetPosition > 0 {
		// Update highest price since entry for trailing stop
		if _, exists := s.highestSinceBuy[pair]; !exists {
			s.highestSinceBuy[pair] = closePrice
		} else if closePrice > s.highestSinceBuy[pair] {
			s.highestSinceBuy[pair] = closePrice
		}
		
		// Check if trailing stop is active and hit
		if s.checkTrailingStop(pair, df) {
			s.executeExit(df, broker, assetPosition, "Trailing Stop")
			return
		}
		
		// Check for take profit
		if s.checkTakeProfit(pair, closePrice) {
			s.executeExit(df, broker, assetPosition, "Take Profit")
			return
		}
		
		// Check for stop loss
		if s.checkStopLoss(pair, lowPrice) {
			s.executeExit(df, broker, assetPosition, "Stop Loss")
			return
		}
	}
	
	// Check for entry conditions if not in position
	if assetPosition == 0 && len(s.activePositions) < s.maxPositions {
		// Check for volatility contraction followed by breakout
		if s.isVolatilityContraction(pair) && s.isBreakout(df, highPrice) {
			// Check trend filters
			if s.checkTrendFilters(df) {
				s.executeEntry(df, broker, quotePosition, closePrice)
			}
		}
	}
}

// updateVolatilityTracking updates the volatility tracking for a pair
func (s *VolatilityBreakoutStrategy) updateVolatilityTracking(pair string, df *core.Dataframe) {
	// Get current ATR percentage
	atrPercent := df.Metadata["atr_percent"].Last(0)
	
	// Initialize if needed
	if _, exists := s.lastVolatility[pair]; !exists {
		s.lastVolatility[pair] = make([]float64, 0, s.atrLookback)
	}
	
	// Add current volatility to tracking
	volatility := s.lastVolatility[pair]
	volatility = append(volatility, atrPercent)
	
	// Keep only the last N values
	if len(volatility) > s.atrLookback {
		volatility = volatility[len(volatility)-s.atrLookback:]
	}
	
	// Update tracking
	s.lastVolatility[pair] = volatility
}

// isVolatilityContraction checks if we're in a volatility contraction phase
func (s *VolatilityBreakoutStrategy) isVolatilityContraction(pair string) bool {
	volatility, exists := s.lastVolatility[pair]
	if !exists || len(volatility) < s.atrLookback {
		return false // Not enough data
	}
	
	// Calculate average volatility
	sum := 0.0
	for _, v := range volatility[:len(volatility)-1] { // Exclude current value
		sum += v
	}
	avgVolatility := sum / float64(len(volatility)-1)
	
	// Current volatility
	currentVolatility := volatility[len(volatility)-1]
	
	// Check if current volatility is below threshold of average
	return currentVolatility < avgVolatility*s.atrThreshold
}

// isBreakout checks if we have a breakout from low volatility
func (s *VolatilityBreakoutStrategy) isBreakout(df *core.Dataframe, highPrice float64) bool {
	pair := df.Pair
	
	// Get ATR for breakout calculation
	atr := df.Metadata["atr"].Last(0)
	
	// Get previous high
	prevHigh := df.High.Last(1)
	
	// Check if we've had a breakout recently (avoid multiple entries)
	lastBreakoutTime, exists := s.lastBreakout[pair]
	if exists {
		timeSinceBreakout := df.Time[len(df.Time)-1].Sub(lastBreakoutTime)
		if timeSinceBreakout.Hours() < 24 { // Avoid breakouts within 24 hours
			return false
		}
	}
	
	// Check for breakout: current high exceeds previous high by breakout factor * ATR
	isBreakout := highPrice > prevHigh+(atr*s.breakoutFactor)
	
	// If breakout detected, record the time
	if isBreakout {
		s.lastBreakout[pair] = df.Time[len(df.Time)-1]
	}
	
	return isBreakout
}

// checkTrendFilters checks if trend filters allow entry
func (s *VolatilityBreakoutStrategy) checkTrendFilters(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	ema := df.Metadata["ema"].Last(0)
	supertrend1Dir := df.Metadata["supertrend1_dir"].Last(0)
	supertrend2Dir := df.Metadata["supertrend2_dir"].Last(0)
	bbWidth := df.Metadata["bb_width"].Last(0)
	bbWidthPrev := df.Metadata["bb_width"].Last(1)
	
	// Price above EMA (uptrend)
	priceAboveEMA := closePrice > ema
	
	// Both Supertrends in uptrend (value 1)
	supertrendBullish := supertrend1Dir > 0 && supertrend2Dir > 0
	
	// Bollinger Bands starting to expand (volatility increasing)
	bbExpanding := bbWidth > bbWidthPrev
	
	// Combine filters
	return priceAboveEMA && supertrendBullish && bbExpanding
}

// checkTrailingStop checks if the trailing stop has been hit
func (s *VolatilityBreakoutStrategy) checkTrailingStop(pair string, df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	highestPrice, exists := s.highestSinceBuy[pair]
	if !exists {
		return false
	}
	
	entryPrice, exists := s.entryPrices[pair]
	if !exists {
		return false
	}
	
	// Only activate trailing stop after price has moved in our favor by trailingStart
	if highestPrice < entryPrice*(1+s.trailingStart) {
		return false
	}
	
	// Calculate trailing stop level based on ATR
	atr := df.Metadata["atr"].Last(0)
	trailDistance := atr * s.trailingFactor
	trailLevel := highestPrice - trailDistance
	
	// Check if price has fallen below trailing stop
	return closePrice < trailLevel
}

// checkTakeProfit checks if take profit level has been hit
func (s *VolatilityBreakoutStrategy) checkTakeProfit(pair string, currentPrice float64) bool {
	entryPrice, exists := s.entryPrices[pair]
	if !exists {
		return false
	}
	
	// Calculate take profit level
	takeProfitLevel := entryPrice * (1 + s.profitTarget)
	
	// Check if price has reached take profit level
	return currentPrice >= takeProfitLevel
}

// checkStopLoss checks if stop loss level has been hit
func (s *VolatilityBreakoutStrategy) checkStopLoss(pair string, lowPrice float64) bool {
	stopLevel, exists := s.stopLosses[pair]
	if !exists {
		return false
	}
	
	// Check if price has fallen below stop loss level
	return lowPrice <= stopLevel
}

// executeEntry performs the entry operation
func (s *VolatilityBreakoutStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair
	
	// Calculate position size based on risk
	atr := df.Metadata["atr"].Last(0)
	
	// Calculate stop loss based on ATR
	stopLossAmount := atr * s.stopMultiplier
	stopLossPrice := closePrice - stopLossAmount
	
	// Calculate position size based on risk
	riskAmount := quotePosition * s.riskPerTrade
	positionSize := riskAmount / stopLossAmount
	
	// Limit position size to available quote
	maxPositionSize := quotePosition / closePrice
	if positionSize > maxPositionSize {
		positionSize = maxPositionSize * 0.95 // 95% of available to account for fees
	}
	
	// Execute market buy
	_, err := broker.CreateOrderMarket(core.SideTypeBuy, pair, positionSize)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"price": closePrice,
			"size":  positionSize,
		}).Error(err)
		return
	}
	
	// Store entry details
	s.activePositions[pair] = true
	s.entryPrices[pair] = closePrice
	s.stopLosses[pair] = stopLossPrice
	s.highestSinceBuy[pair] = closePrice
	
	// Set stop loss order
	stopOrder, err := broker.CreateOrderStop(pair, positionSize, stopLossPrice)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":      pair,
			"size":      positionSize,
			"stopPrice": stopLossPrice,
		}).Error(err)
	} else {
		s.activeOrders[pair] = stopOrder.ID
	}
	
	backnrun.DefaultLog.WithFields(map[string]interface{}{
		"pair":          pair,
		"price":         closePrice,
		"size":          positionSize,
		"stopLoss":      stopLossPrice,
		"volatility":    atr / closePrice,
	}).Info("Breakout entry executed")
}

// executeExit performs the exit operation
func (s *VolatilityBreakoutStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64, reason string) {
	pair := df.Pair
	currentPrice := df.Close.Last(0)
	
	// Cancel any active stop orders
	if orderID, exists := s.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			_ = broker.Cancel(order)
		}
		delete(s.activeOrders, pair)
	}
	
	// Execute market sell
	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"price": currentPrice,
			"size":  assetPosition,
		}).Error(err)
		return
	}
	
	// Calculate profit/loss
	entryPrice, exists := s.entryPrices[pair]
	if exists {
		pnlPercent := (currentPrice - entryPrice) / entryPrice * 100
		
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":       pair,
			"entryPrice": entryPrice,
			"exitPrice":  currentPrice,
			"pnlPercent": pnlPercent,
			"reason":     reason,
		}).Info("Exit executed")
	}
	
	// Clean up
	delete(s.activePositions, pair)
	delete(s.entryPrices, pair)
	delete(s.stopLosses, pair)
	delete(s.highestSinceBuy, pair)
}
