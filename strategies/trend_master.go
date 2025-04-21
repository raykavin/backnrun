package strategies

import (
	"context"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

type TrendMasterParameters struct {
}

// TrendMasterStrategy implements a strategy that combines multiple indicators
// to identify strong trends and generate entry and exit signals
type TrendMasterStrategy struct {
	// Indicator parameters
	emaFastPeriod    int
	emaSlowPeriod    int
	emaLongPeriod    int
	macdFast         int
	macdSlow         int
	macdSignal       int
	adxPeriod        int
	adxThreshold     float64
	profitTarget     float64
	stopLoss         float64
	positionSize     float64
	maxRiskPerTrade  float64
	trailStopPercent float64
	atrPeriod        int
	atrMultiplier    float64

	// Trade control
	consecutiveLosses int
	maxTradesPerDay   int
	dailyTradeCount   map[string]int
	lastTradeDate     string
	lastTradeResult   map[string]bool // true = profit, false = loss

	// Additional filters
	useRsiFilter  bool
	rsiPeriod     int
	rsiOverbought float64
	rsiOversold   float64
	useVolFilter  bool
	volAvgPeriod  int
	minVolRatio   float64

	// Internal tracking
	activeOrders map[string]int64   // Tracks active stop orders
	lastPrice    map[string]float64 // Tracks last price for trailing stop
	entryPrice   map[string]float64 // Entry price for each pair
}

// NewTrendMasterStrategy creates a new instance of the strategy with the default parameters
func NewTrendMasterStrategy() *TrendMasterStrategy {
	return &TrendMasterStrategy{
		emaFastPeriod:    9,
		emaSlowPeriod:    21,
		emaLongPeriod:    34,
		macdSlow:         150,
		macdFast:         14,
		macdSignal:       14,
		adxPeriod:        14,
		adxThreshold:     25.0,
		profitTarget:     0.06,
		stopLoss:         0.02,
		positionSize:     0.3,
		maxRiskPerTrade:  0.01,
		trailStopPercent: 0.03,
		atrPeriod:        14,
		atrMultiplier:    2.0,

		// Trade control
		consecutiveLosses: 4,
		maxTradesPerDay:   8,
		dailyTradeCount:   make(map[string]int),
		lastTradeResult:   make(map[string]bool),

		// Additional indicator filters
		useRsiFilter:  true,
		rsiPeriod:     14,
		rsiOverbought: 70.0,
		rsiOversold:   30.0,
		useVolFilter:  true,
		volAvgPeriod:  20,
		minVolRatio:   1.1,

		// Internal tracking
		activeOrders: make(map[string]int64),
		lastPrice:    make(map[string]float64),
		entryPrice:   make(map[string]float64),
	}
}

// Timeframe returns the timeframe required for this strategy
func (t TrendMasterStrategy) Timeframe() string {
	return "5m"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (t TrendMasterStrategy) WarmupPeriod() int {
	// Use the slow MACD period + signal period + extra safety margin
	// MACD needs much more historical data to be calculated correctly
	return t.macdSlow*2 + t.macdSignal + 100
}

// Indicators calculates and returns the indicators used by this strategy
func (t TrendMasterStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate EMAs
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.emaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, t.emaSlowPeriod)
	df.Metadata["ema_long_high"] = indicator.EMA(df.High, t.emaLongPeriod)
	df.Metadata["ema_long_low"] = indicator.EMA(df.Low, t.emaLongPeriod)

	// Calculate MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		t.macdFast,
		t.macdSlow,
		t.macdSignal,
	)

	// Calculate ADX and directional indicators
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, t.adxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, t.adxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, t.adxPeriod)

	// Add ATR for volatility calculation
	df.Metadata["atr"] = talib.Atr(df.High, df.Low, df.Close, t.atrPeriod)

	// Calculate RSI for additional filtering
	if t.useRsiFilter {
		df.Metadata["rsi"] = talib.Rsi(df.Close, t.rsiPeriod)
	}

	// Calculate volume average for filtering
	if t.useVolFilter {
		df.Metadata["vol_avg"] = indicator.SMA(df.Volume, t.volAvgPeriod)
	}

	// Return indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA 9",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_slow"],
					Name:   "EMA 21",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_high"],
					Name:   "EMA 34 (High)",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_low"],
					Name:   "EMA 34 (Low)",
					Color:  "orange",
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
			GroupName: "ATR",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["atr"],
					Name:   "ATR",
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (t *TrendMasterStrategy) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	// Get current date (last candle)
	currentDate := ""
	if len(df.Time) > 0 {
		currentDate = df.Time[len(df.Time)-1].Format("2006-01-02")
	}

	// Reset daily trade count if we're in a new day
	if currentDate != t.lastTradeDate {
		t.dailyTradeCount = make(map[string]int)
		t.lastTradeDate = currentDate
	}

	// Get current position
	assetPosition, quotePosition, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Update trailing stop price if in position
	if assetPosition > 0 {
		lastPrice, exists := t.lastPrice[pair]
		if !exists || closePrice > lastPrice {
			t.lastPrice[pair] = closePrice
		}
	}

	// Check entry and exit signals
	if assetPosition > 0 {
		// We are already in a long position, check for exit
		if t.shouldExit(df) || t.checkTrailingStop(df, pair) {
			t.executeExit(ctx, df, broker, assetPosition)
			// Reset after exit
			delete(t.lastPrice, pair)
		}
	} else {
		// No position, check for entry (only if we haven't exceeded daily limit)
		tradeCount := t.dailyTradeCount[pair]
		if tradeCount < t.maxTradesPerDay && t.shouldEnter(df) {
			t.executeEntry(ctx, df, broker, quotePosition, closePrice)
			// Increment trade counter
			t.dailyTradeCount[pair] = tradeCount + 1
		}
	}
}

// checkTrailingStop checks if the trailing stop has been hit
func (t *TrendMasterStrategy) checkTrailingStop(df *core.Dataframe, pair string) bool {
	closePrice := df.Close.Last(0)
	lastPrice, exists := t.lastPrice[pair]

	if !exists {
		return false
	}

	// If price has fallen below trailing stop, trigger exit
	trailAmount := lastPrice * t.trailStopPercent
	if closePrice <= lastPrice-trailAmount {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":         pair,
			"highestPrice": lastPrice,
			"currentPrice": closePrice,
			"trailAmount":  trailAmount,
		}).Info("Trailing stop activated")
		return true
	}

	return false
}

// shouldEnter checks if entry conditions are met
func (t *TrendMasterStrategy) shouldEnter(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongHigh := df.Metadata["ema_long_high"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)
	atr := df.Metadata["atr"].Last(0)

	// Basic checks (original conditions)
	priceAboveEMA := closePrice > emaLongHigh
	macdAboveSignal := macd > macdSignal
	plusDIAboveMinusDI := plusDI > minusDI
	adxAboveThreshold := adx > t.adxThreshold
	emaFastAboveSlow := emaFast > emaSlow

	// Volatility filter: check if volatility is not too high
	volatilityCheck := true
	if atr > 0 {
		volatilityRatio := atr / closePrice
		volatilityCheck = volatilityRatio < 0.015 // Volatility less than 1.5%
	}

	// RSI filter: avoid buying in overbought market
	rsiCheck := true
	if t.useRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		rsiCheck = rsi < t.rsiOverbought // RSI is not overbought
	}

	// Volume filter: check if volume is sufficient
	volumeCheck := true
	if t.useVolFilter {
		volume := df.Volume.Last(0)
		volumeAvg := df.Metadata["vol_avg"].Last(0)
		if volumeAvg > 0 {
			volumeRatio := volume / volumeAvg
			volumeCheck = volumeRatio >= t.minVolRatio // Volume above average
		}
	}

	// Check for consistent bullish trend
	bullishMarket := true // Simplified to allow more signals

	// Reduce trade frequency after consecutive losses
	riskAdjustment := true
	if t.consecutiveLosses >= 2 {
		// After 2 consecutive losses, require more stringent conditions
		riskAdjustment = adx > (t.adxThreshold+5) && (plusDI-minusDI) > 10
	}

	// All main conditions must be true for entry
	mainConditions := priceAboveEMA && macdAboveSignal && plusDIAboveMinusDI &&
		adxAboveThreshold && emaFastAboveSlow

	// Additional filters to improve signal quality
	additionalFilters := volatilityCheck && rsiCheck && volumeCheck && bullishMarket && riskAdjustment

	return mainConditions && additionalFilters
}

// shouldExit checks if exit conditions are met
func (t *TrendMasterStrategy) shouldExit(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongLow := df.Metadata["ema_long_low"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)

	// Basic exit conditions
	priceBelowEMA := closePrice < emaLongLow
	macdBelowSignal := macd < macdSignal
	minusDIAbovePlusDI := minusDI > plusDI
	adxAboveThreshold := adx > t.adxThreshold
	emaFastBelowSlow := emaFast < emaSlow

	// Check breaking of long moving average calculated on low
	// This is a quick exit signal independent of other conditions
	if priceBelowEMA {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":       df.Pair,
			"closePrice": closePrice,
			"emaLongLow": emaLongLow,
		}).Info("Quick exit: price below EMA Low")
		return true
	}

	// Quick exit if MACD falls very quickly (potential strong reversal)
	macdHist := df.Metadata["macd_hist"].Last(0)
	prevMacdHist := df.Metadata["macd_hist"].Last(1)
	if macdHist < 0 && prevMacdHist > 0 && macdHist < prevMacdHist*-1.5 {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":         df.Pair,
			"macdHist":     macdHist,
			"prevMacdHist": prevMacdHist,
		}).Info("Quick exit: MACD strongly reversing")
		return true
	}

	// Quick exit if ADX starts falling while in position (trend weakening)
	adxFalling := adx < df.Metadata["adx"].Last(1) && df.Metadata["adx"].Last(1) < df.Metadata["adx"].Last(2)
	if adxFalling && minusDIAbovePlusDI {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":    df.Pair,
			"adx":     adx,
			"prevAdx": df.Metadata["adx"].Last(1),
			"plusDI":  plusDI,
			"minusDI": minusDI,
		}).Info("Quick exit: ADX falling and -DI above +DI")
		return true
	}

	// RSI indicating extreme overbought
	if t.useRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		if rsi > 80 { // Extreme overbought
			bot.DefaultLog.WithFields(map[string]any{
				"pair": df.Pair,
				"rsi":  rsi,
			}).Info("Quick exit: RSI in extreme overbought")
			return true
		}
	}

	// All regular exit conditions
	regularExitConditions := macdBelowSignal && minusDIAbovePlusDI && adxAboveThreshold && emaFastBelowSlow

	return regularExitConditions
}

// executeEntry executes the entry operation
func (t *TrendMasterStrategy) executeEntry(ctx context.Context, df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Cancel any previous active order (guarantee)
	if orderID, exists := t.activeOrders[pair]; exists {
		order, err := broker.Order(ctx, pair, orderID)
		if err == nil {
			_ = broker.Cancel(ctx, order)
		}
		delete(t.activeOrders, pair)
	}

	// Adjust position size based on ATR for risk control
	atr := df.Metadata["atr"].Last(0)
	positionSize := t.positionSize

	// Reduce position size after consecutive losses
	if t.consecutiveLosses > 0 {
		// Reduce position size by 20% for each consecutive loss (up to 60%)
		reductionFactor := 1.0 - float64(t.consecutiveLosses)*0.2
		if reductionFactor < 0.4 {
			reductionFactor = 0.4 // Minimum 40% of original size
		}
		positionSize *= reductionFactor
	}

	// Calculate stop loss based on ATR if available
	stopLossPrice := 0.0
	if atr > 0 {
		// Use ATR to calculate dynamic stop loss
		stopLossPrice = closePrice - (atr * t.atrMultiplier)

		// Calculate stop loss percentage based on ATR
		stopLossPercent := (closePrice - stopLossPrice) / closePrice

		// If ATR-based stop loss is greater than maximum allowed, adjust position size
		if stopLossPercent > t.stopLoss {
			// Adjust position size to limit risk
			riskAdjustment := t.stopLoss / stopLossPercent
			positionSize *= riskAdjustment

			// Use fixed stop loss instead of ATR
			stopLossPrice = closePrice * (1.0 - t.stopLoss)
		}
	} else {
		// Use fixed stop loss if ATR is not available
		stopLossPrice = closePrice * (1.0 - t.stopLoss)
	}

	// Limit risk per trade
	maxRiskAmount := quotePosition * t.maxRiskPerTrade
	actualRiskAmount := quotePosition * positionSize * ((closePrice - stopLossPrice) / closePrice)
	if actualRiskAmount > maxRiskAmount {
		// Adjust position size to limit risk
		positionSize *= maxRiskAmount / actualRiskAmount
	}

	// Calculate position size based on available capital
	entryAmount := quotePosition * positionSize

	// Execute market buy order
	_, err := broker.CreateOrderMarketQuote(ctx, core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Record entry price
	t.entryPrice[pair] = closePrice

	// Initialize trailing stop with entry price
	t.lastPrice[pair] = closePrice

	// Get updated position after purchase
	assetPosition, _, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Set take profit and stop loss with OCO order
	takeProfitPrice := closePrice * (1.0 + t.profitTarget)

	orders, err := broker.CreateOrderOCO(
		ctx,
		core.SideTypeSell,
		pair,
		assetPosition,
		takeProfitPrice,
		stopLossPrice,
		stopLossPrice,
	)

	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":       pair,
			"side":       core.SideTypeSell,
			"asset":      assetPosition,
			"takeProfit": takeProfitPrice,
			"stopPrice":  stopLossPrice,
		}).Error(err)
	} else if len(orders) > 0 {
		// Store ID of first order for future reference
		t.activeOrders[pair] = orders[0].ID
	}

	bot.DefaultLog.WithFields(map[string]any{
		"pair":              pair,
		"entryPrice":        closePrice,
		"positionSize":      positionSize,
		"stopLossPrice":     stopLossPrice,
		"takeProfitPrice":   takeProfitPrice,
		"consecutiveLosses": t.consecutiveLosses,
	}).Info("Entry executed")
}

// executeExit executes the exit operation
func (t *TrendMasterStrategy) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair
	currentPrice := df.Close.Last(0)

	// Cancel any active OCO order
	if orderID, exists := t.activeOrders[pair]; exists {
		order, err := broker.Order(ctx, pair, orderID)
		if err == nil {
			err = broker.Cancel(ctx, order)
			if err != nil {
				bot.DefaultLog.WithFields(map[string]any{
					"pair":    pair,
					"orderID": orderID,
				}).Error(err)
			}
		}

		delete(t.activeOrders, pair)
	}

	// Sell entire position
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, assetPosition)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": currentPrice,
		}).Error(err)
		return
	}

	// Check trade result
	entryPrice, exists := t.entryPrice[pair]
	if exists {
		tradeProfit := currentPrice > entryPrice
		t.lastTradeResult[pair] = tradeProfit

		// Update consecutive loss counter
		if tradeProfit {
			t.consecutiveLosses = 0 // Reset after a profitable trade
		} else {
			t.consecutiveLosses++ // Increment loss counter
		}

		bot.DefaultLog.WithFields(map[string]any{
			"pair":        pair,
			"entryPrice":  entryPrice,
			"exitPrice":   currentPrice,
			"profit":      (currentPrice - entryPrice) / entryPrice * 100,
			"isProfit":    tradeProfit,
			"consecutive": t.consecutiveLosses,
		}).Info("Exit executed")

		// Clear entry price
		delete(t.entryPrice, pair)
	}
}
