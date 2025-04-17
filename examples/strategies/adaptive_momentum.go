package strategies

import (
	"math"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

// AdaptiveMomentumStrategy implements an adaptive momentum strategy that
// combines multiple indicators and adjusts parameters based on market conditions
type AdaptiveMomentumStrategy struct {
	// Base parameters
	rsiPeriod       int
	rsiOverbought   float64
	rsiOversold     float64
	emaFastPeriod   int
	emaSlowPeriod   int
	adxPeriod       int
	adxThreshold    float64
	atrPeriod       int
	atrMultiplier   float64
	bollingerPeriod int
	bollingerDev    float64
	volAvgPeriod    int
	minVolRatio     float64

	// Position sizing and risk management
	initialRisk     float64 // Initial risk per trade as percentage of account
	maxRisk         float64 // Maximum risk allowed per trade
	profitTarget    float64 // Take profit target
	trailingPercent float64 // Trailing stop percentage

	// Adaptive parameters
	volatilityLevels []float64 // Thresholds for different volatility regimes
	trendStrength    []float64 // Thresholds for different trend strength regimes

	// State tracking
	activeOrders     map[string]int64   // Track active orders
	entryPrices      map[string]float64 // Track entry prices
	highestSinceBuy  map[string]float64 // Track highest price since entry for trailing stop
	consecutiveLoss  int                // Track consecutive losses
	winCount         int                // Track win count
	lossCount        int                // Track loss count
	currentVolRegime int                // Current volatility regime
	currentTrendReg  int                // Current trend regime
}

// NewAdaptiveMomentumStrategy creates a new instance of the strategy
func NewAdaptiveMomentumStrategy() *AdaptiveMomentumStrategy {
	return &AdaptiveMomentumStrategy{
		// Base parameters
		rsiPeriod:       14,
		rsiOverbought:   70,
		rsiOversold:     30,
		emaFastPeriod:   9,
		emaSlowPeriod:   21,
		adxPeriod:       14,
		adxThreshold:    25,
		atrPeriod:       14,
		atrMultiplier:   2.0,
		bollingerPeriod: 20,
		bollingerDev:    2.0,
		volAvgPeriod:    20,
		minVolRatio:     1.2,

		// Position sizing and risk management
		initialRisk:     0.01, // 1% risk per trade
		maxRisk:         0.02, // Maximum 2% risk per trade
		profitTarget:    0.03, // 3% take profit
		trailingPercent: 0.02, // 2% trailing stop

		// Adaptive parameters
		volatilityLevels: []float64{0.005, 0.01, 0.02}, // 0.5%, 1%, 2% volatility thresholds
		trendStrength:    []float64{20, 30, 40},        // ADX thresholds for trend strength

		// State tracking
		activeOrders:    make(map[string]int64),
		entryPrices:     make(map[string]float64),
		highestSinceBuy: make(map[string]float64),
	}
}

// Timeframe returns the required timeframe for this strategy
func (s AdaptiveMomentumStrategy) Timeframe() string {
	return "15m" // 15-minute timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s AdaptiveMomentumStrategy) WarmupPeriod() int {
	// Use the maximum period needed for any indicator plus some buffer
	return s.bollingerPeriod + 50
}

// Indicators calculates and returns the indicators used by this strategy
func (s AdaptiveMomentumStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate RSI
	df.Metadata["rsi"] = talib.Rsi(df.Close, s.rsiPeriod)

	// Calculate EMAs
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, s.emaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, s.emaSlowPeriod)

	// Calculate ADX
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, s.adxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, s.adxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, s.adxPeriod)

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

	// Calculate Bollinger Bands
	upper, middle, lower := talib.BBands(
		df.Close,
		s.bollingerPeriod,
		s.bollingerDev,
		s.bollingerDev,
		talib.SMA,
	)
	df.Metadata["bb_upper"] = upper
	df.Metadata["bb_middle"] = middle
	df.Metadata["bb_lower"] = lower

	// Calculate volume average
	df.Metadata["vol_avg"] = indicator.SMA(df.Volume, s.volAvgPeriod)

	// Calculate momentum
	momentum := make([]float64, len(df.Close))
	for i := s.emaFastPeriod; i < len(df.Close); i++ {
		momentum[i] = df.Close[i] - df.Close[i-s.emaFastPeriod]
	}
	df.Metadata["momentum"] = momentum

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA Fast",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_slow"],
					Name:   "EMA Slow",
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
			GroupName: "RSI",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["rsi"],
					Name:   "RSI",
					Color:  "purple",
					Style:  core.StyleLine,
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
					Color:  "black",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["plus_di"],
					Name:   "+DI",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["minus_di"],
					Name:   "-DI",
					Color:  "red",
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
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (s *AdaptiveMomentumStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Update market regime
	s.updateMarketRegime(df)

	// Update trailing stop if in position
	if assetPosition > 0 {
		// Update highest price since entry for trailing stop
		if _, exists := s.highestSinceBuy[pair]; !exists {
			s.highestSinceBuy[pair] = closePrice
		} else if closePrice > s.highestSinceBuy[pair] {
			s.highestSinceBuy[pair] = closePrice
		}

		// Check trailing stop
		if s.checkTrailingStop(pair, closePrice) {
			s.executeExit(df, broker, assetPosition, "Trailing Stop")
			return
		}
	}

	// Check for exit conditions if in position
	if assetPosition > 0 && s.shouldExit(df) {
		s.executeExit(df, broker, assetPosition, "Regular Exit")
		return
	}

	// Check for entry conditions if not in position
	if assetPosition == 0 && s.shouldEnter(df) {
		s.executeEntry(df, broker, quotePosition, closePrice)
	}
}

// updateMarketRegime analyzes current market conditions and updates regime parameters
func (s *AdaptiveMomentumStrategy) updateMarketRegime(df *core.Dataframe) {
	// Get current ATR percentage (volatility)
	atrPercent := df.Metadata["atr_percent"].Last(0)
	
	// Determine volatility regime
	s.currentVolRegime = 0 // Low volatility by default
	for i, threshold := range s.volatilityLevels {
		if atrPercent > threshold {
			s.currentVolRegime = i + 1
		}
	}
	
	// Get current ADX (trend strength)
	adx := df.Metadata["adx"].Last(0)
	
	// Determine trend regime
	s.currentTrendReg = 0 // Weak trend by default
	for i, threshold := range s.trendStrength {
		if adx > threshold {
			s.currentTrendReg = i + 1
		}
	}
}

// shouldEnter checks if entry conditions are met
func (s *AdaptiveMomentumStrategy) shouldEnter(df *core.Dataframe) bool {
	// Get indicator values
	rsi := df.Metadata["rsi"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	adxValue := df.Metadata["adx"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	closePrice := df.Close.Last(0)
	bbLower := df.Metadata["bb_lower"].Last(0)
	volume := df.Volume.Last(0)
	volAvg := df.Metadata["vol_avg"].Last(0)
	
	// Adjust entry conditions based on market regime
	var rsiThreshold float64
	var adxThreshold float64
	var diDiffThreshold float64
	
	// Adjust RSI threshold based on volatility
	switch s.currentVolRegime {
	case 0: // Low volatility
		rsiThreshold = 40 // More conservative in low volatility
	case 1: // Medium volatility
		rsiThreshold = 35
	default: // High volatility
		rsiThreshold = 30 // More aggressive in high volatility
	}
	
	// Adjust ADX threshold based on trend strength
	switch s.currentTrendReg {
	case 0: // Weak trend
		adxThreshold = 30 // Need stronger trend confirmation
		diDiffThreshold = 10
	case 1: // Medium trend
		adxThreshold = 25
		diDiffThreshold = 5
	default: // Strong trend
		adxThreshold = 20 // Less confirmation needed in strong trend
		diDiffThreshold = 3
	}
	
	// Volume filter
	volumeFilter := volume >= volAvg * s.minVolRatio
	
	// Price near lower Bollinger Band (potential bounce)
	priceNearLowerBB := closePrice <= bbLower * 1.01
	
	// RSI oversold condition (adjusted by regime)
	rsiOversold := rsi < rsiThreshold
	
	// Trend conditions
	trendStrength := adxValue > adxThreshold
	bullishTrend := plusDI > minusDI && (plusDI - minusDI) > diDiffThreshold
	
	// EMA alignment
	emaAlignment := emaFast >= emaSlow
	
	// Combine conditions based on market regime
	if s.currentVolRegime >= 2 && s.currentTrendReg >= 2 {
		// High volatility, strong trend - focus on trend following
		return trendStrength && bullishTrend && volumeFilter
	} else if s.currentVolRegime <= 0 && s.currentTrendReg <= 0 {
		// Low volatility, weak trend - focus on mean reversion
		return priceNearLowerBB && rsiOversold && volumeFilter
	} else {
		// Medium conditions - require more confirmation
		return emaAlignment && trendStrength && bullishTrend && 
			(rsiOversold || priceNearLowerBB) && volumeFilter
	}
}

// shouldExit checks if exit conditions are met
func (s *AdaptiveMomentumStrategy) shouldExit(df *core.Dataframe) bool {
	// Get indicator values
	rsi := df.Metadata["rsi"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	closePrice := df.Close.Last(0)
	bbUpper := df.Metadata["bb_upper"].Last(0)
	
	// Adjust exit conditions based on market regime
	var rsiThreshold float64
	
	// Adjust RSI threshold based on volatility
	switch s.currentVolRegime {
	case 0: // Low volatility
		rsiThreshold = 65 // More conservative in low volatility
	case 1: // Medium volatility
		rsiThreshold = 70
	default: // High volatility
		rsiThreshold = 75 // More aggressive in high volatility
	}
	
	// Price near upper Bollinger Band (potential resistance)
	priceNearUpperBB := closePrice >= bbUpper * 0.99
	
	// RSI overbought condition (adjusted by regime)
	rsiOverbought := rsi > rsiThreshold
	
	// Trend reversal conditions
	trendReversal := minusDI > plusDI
	
	// EMA crossover (fast crosses below slow)
	emaCrossdown := emaFast < emaSlow && df.Metadata["ema_fast"].Last(1) >= df.Metadata["ema_slow"].Last(1)
	
	// Take profit check
	pair := df.Pair
	entryPrice, exists := s.entryPrices[pair]
	takeProfitHit := exists && closePrice >= entryPrice * (1 + s.profitTarget)
	
	// Combine conditions based on market regime
	if takeProfitHit {
		return true // Always exit on take profit
	} else if s.currentVolRegime >= 2 {
		// High volatility - exit on trend reversal
		return trendReversal || emaCrossdown
	} else if s.currentTrendReg <= 0 {
		// Weak trend - exit on overbought conditions
		return rsiOverbought || priceNearUpperBB
	} else {
		// Medium conditions
		return (rsiOverbought && priceNearUpperBB) || emaCrossdown || trendReversal
	}
}

// checkTrailingStop checks if the trailing stop has been hit
func (s *AdaptiveMomentumStrategy) checkTrailingStop(pair string, currentPrice float64) bool {
	highestPrice, exists := s.highestSinceBuy[pair]
	if !exists {
		return false
	}
	
	// Calculate trailing stop level
	trailLevel := highestPrice * (1 - s.trailingPercent)
	
	// Check if price has fallen below trailing stop
	return currentPrice < trailLevel
}

// executeEntry performs the entry operation
func (s *AdaptiveMomentumStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair
	
	// Calculate position size based on risk
	atr := df.Metadata["atr"].Last(0)
	
	// Adjust risk based on market conditions and past performance
	riskPercent := s.initialRisk
	
	// Reduce risk after consecutive losses
	if s.consecutiveLoss > 0 {
		riskPercent = math.Max(s.initialRisk * 0.5, s.initialRisk - (float64(s.consecutiveLoss) * 0.002))
	}
	
	// Increase risk if win rate is good
	if s.winCount > 0 && s.lossCount > 0 {
		winRate := float64(s.winCount) / float64(s.winCount + s.lossCount)
		if winRate > 0.6 {
			riskPercent = math.Min(s.maxRisk, riskPercent * 1.2)
		}
	}
	
	// Calculate stop loss based on ATR
	stopLossAmount := atr * s.atrMultiplier
	stopLossPrice := closePrice - stopLossAmount
	
	// Calculate position size based on risk
	riskAmount := quotePosition * riskPercent
	positionSize := riskAmount / stopLossAmount
	
	// Limit position size to available quote
	maxPositionSize := quotePosition / closePrice
	if positionSize > maxPositionSize {
		positionSize = maxPositionSize
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
	
	// Store entry price
	s.entryPrices[pair] = closePrice
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
		"riskPercent":   riskPercent,
		"volRegime":     s.currentVolRegime,
		"trendRegime":   s.currentTrendReg,
	}).Info("Entry executed")
}

// executeExit performs the exit operation
func (s *AdaptiveMomentumStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64, reason string) {
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
		
		// Update win/loss counters
		if currentPrice > entryPrice {
			s.winCount++
			s.consecutiveLoss = 0
		} else {
			s.lossCount++
			s.consecutiveLoss++
		}
		
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":      pair,
			"entryPrice": entryPrice,
			"exitPrice": currentPrice,
			"pnlPercent": pnlPercent,
			"reason":    reason,
		}).Info("Exit executed")
		
		// Clean up
		delete(s.entryPrices, pair)
		delete(s.highestSinceBuy, pair)
	}
}
